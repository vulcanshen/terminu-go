package terminu

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/dslipak/pdf"
)

// previewBytes is the most of a file a preview reads.
const previewBytes = 64 << 10

// fileKind is what a file is, for its preview and its TYPE
// (tdp file-picker).
func fileKind(p string, head []byte) string {
	name := strings.ToLower(filepath.Base(p))
	switch {
	case isArchive(name):
		return "archive"
	case slices.Contains([]string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".tiff", ".tif", ".ico", ".avif", ".heic"}, path.Ext(name)):
		return "image"
	case path.Ext(name) == ".pdf":
		return "PDF"
	case isText(head):
		if isPrivateKey(head) {
			return "private key"
		}
		return "text"
	}
	return "binary"
}

// isPrivateKey reports a first line that is a private key's header: it
// starts with -----BEGIN and has PRIVATE KEY (OpenSSH, RSA, EC, PKCS#8, PGP),
// or it starts with PuTTY-User-Key-File-.
func isPrivateKey(head []byte) bool {
	first := firstLine(head)
	return strings.HasPrefix(first, "-----BEGIN") && strings.Contains(first, "PRIVATE KEY") || strings.HasPrefix(first, "PuTTY-User-Key-File-")
}

func firstLine(b []byte) string {
	first, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimRight(first, "\r")
}

func isArchive(name string) bool {
	for _, ext := range []string{".zip", ".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

func isText(b []byte) bool {
	return !bytes.Contains(b, []byte{0}) && utf8.Valid(trimPartialRune(b))
}

// filePreview is a file or a directory as a preview shows it, in w × h
// (tdp file-picker): a directory's or an archive's tree, an
// image as a data URI, a PDF's first pages, a text file in colour with line
// numbers, anything else in hex. With hideSecrets, a private key shows its
// first line only and any other file only its owner may read shows nothing.
func filePreview(p string, w, h int, hideSecrets bool, icon func(string, bool) (string, Color), tabW int) []string {
	return previewIn(nil, p, 0, w, h, hideSecrets, icon, tabW)
}

// previewIn is filePreview of p in fsys (the local disk when nil); a text
// file starts near line when it is set: a search's hit on the third row, the
// two lines above it in sight, nothing else marked, the line numbers the
// file's own (tdp finder).
func previewIn(fsys PreviewFS, p string, line, w, h int, hideSecrets bool, icon func(string, bool) (string, Color), tabW int) []string {
	fsys = orLocal(fsys)
	grey := func(s string) []string { return []string{Fg(Overlay0).Render(s)} }
	st, err := fsys.Stat(p)
	if err != nil {
		return grey("(unreadable)")
	}
	if st.IsDir() {
		return dirTree(fsys, p, h, icon)
	}
	f, err := fsys.Open(p)
	if err != nil {
		return grey("(unreadable)")
	}
	defer f.Close()
	head := make([]byte, previewBytes)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	kind := fileKind(p, head)
	if hideSecrets {
		if kind == "private key" {
			return []string{Fg(Text).Render(firstLine(head)), Fg(Overlay0).Render("(sensitive content not shown)")}
		}
		if st.Mode().Perm()&0o044 == 0 {
			return grey("(sensitive content not shown)")
		}
	}
	switch kind {
	case "archive":
		return archiveTree(fsys, p, h, icon)
	case "image":
		return imagePreview(fsys, p, st.Size(), w, h)
	case "PDF":
		return pdfText(fsys, p, st.Size(), h)
	case "text", "private key":
		if line > 3 && kind == "text" {
			first, text := linesFrom(io.MultiReader(bytes.NewReader(head), f), line-2, h)
			return highlight(p, text, h, tabW, first)
		}
		return highlight(p, string(trimPartialRune(head)), h, tabW, 1)
	}
	return hexDump(head, h)
}

// imagePreview is the type and the size in grey, a blank line, then the
// image as a data URI folded at the preview's width; over 1 MB, a line
// saying so. Only what is in sight is read.
func imagePreview(fsys PreviewFS, p string, sz int64, w, h int) []string {
	if sz > 1<<20 {
		return []string{Fg(Overlay0).Render("(" + size(sz) + " — too large to inline as a data URI)")}
	}
	w = max(w, 1)
	mime := imageMIME(p)
	out := []string{Fg(Overlay0).Render("data:" + mime + " · " + size(sz)), ""}
	prefix := "data:" + mime + ";base64,"
	need := max((h-len(out))*w-len(prefix), 0)
	f, err := fsys.Open(p)
	if err != nil {
		return []string{Fg(Overlay0).Render("(unreadable)")}
	}
	defer f.Close()
	b := make([]byte, min(sz, int64(need/4+1)*3))
	n, _ := io.ReadFull(f, b)
	uri := prefix + base64.StdEncoding.EncodeToString(b[:n])
	for i := 0; i < len(uri) && len(out) < h; i += w {
		out = append(out, Fg(Text).Render(uri[i:min(i+w, len(uri))]))
	}
	return out
}

func imageMIME(p string) string {
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(p)), ".")
	switch ext {
	case "jpg":
		ext = "jpeg"
	case "tif":
		ext = "tiff"
	case "ico":
		ext = "x-icon"
	}
	return "image/" + ext
}

// drawn is a character of a file preview or a viewer as it is drawn
// (tdp note): a tab as tabW spaces (input_tab_width), any
// other control character as its name (\r, \x07), never sent to the
// terminal, so a row keeps its width (tdp L4).
func drawn(c string, tabW int) string {
	if c == "\t" {
		return strings.Repeat(" ", tabW)
	}
	if name, ok := ctrlName(c); ok && c != "\n" {
		return name
	}
	return c
}

// visibleText is a line as drawn gives it, the SGR sequences that style it
// kept; any other escape sequence is characters, its ESC drawn by name, so
// it is shown, not obeyed (tdp note).
func visibleText(s string, tabW int) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r >= 0x7f && r <= 0x9f }) {
		return s
	}
	var b strings.Builder
	walk(s, func(seq string, cells int) {
		if cells == 0 && !isSGR(seq) {
			for _, c := range graphemes(seq) {
				b.WriteString(drawn(c, tabW))
			}
			return
		}
		b.WriteString(drawn(seq, tabW))
	})
	return b.String()
}

// isSGR reports whether seq is one SGR sequence: a style.
func isSGR(seq string) bool { return sgrAt.FindString(seq) == seq }

// stripSGR is s without its SGR sequences: the characters of a line that
// may carry styles, every other escape sequence among them.
func stripSGR(s string) string {
	if !strings.Contains(s, "\x1b[") {
		return s
	}
	var b strings.Builder
	walk(s, func(seq string, cells int) {
		if cells > 0 || !isSGR(seq) {
			b.WriteString(seq)
		}
	})
	return b.String()
}

// fileText is a line of a file as drawn: every control character by its name,
// ESC too, so an escape sequence written in the file is shown, not obeyed,
// and nothing of it is lost (tdp file-picker).
func fileText(s string, tabW int) string {
	var b strings.Builder
	for _, c := range graphemes(s) {
		b.WriteString(drawn(c, tabW))
	}
	return b.String()
}

// lsMode is a mode as ls -l writes it: the kind (- d l c b p s), then rwx
// three times, s, S, t, T where setuid, setgid and sticky are
// (tdp file-picker).
func lsMode(m fs.FileMode) string {
	kind := byte('-')
	switch {
	case m&fs.ModeDir != 0:
		kind = 'd'
	case m&fs.ModeSymlink != 0:
		kind = 'l'
	case m&fs.ModeCharDevice != 0:
		kind = 'c'
	case m&fs.ModeDevice != 0:
		kind = 'b'
	case m&fs.ModeNamedPipe != 0:
		kind = 'p'
	case m&fs.ModeSocket != 0:
		kind = 's'
	}
	b := []byte{kind}
	for i, who := range []fs.FileMode{0o700, 0o070, 0o007} {
		p := m.Perm() & who
		shift := uint(6 - 3*i)
		r, w, x := p>>shift&4 != 0, p>>shift&2 != 0, p>>shift&1 != 0
		b = append(b, "-r"[btoi(r)], "-w"[btoi(w)])
		special := i == 0 && m&fs.ModeSetuid != 0 || i == 1 && m&fs.ModeSetgid != 0 || i == 2 && m&fs.ModeSticky != 0
		switch {
		case special && i < 2:
			b = append(b, "Ss"[btoi(x)])
		case special:
			b = append(b, "Tt"[btoi(x)])
		default:
			b = append(b, "-x"[btoi(x)])
		}
	}
	return string(b)
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// redNames is line, chars drawn as visibleText draws them and coloured, with
// the names of the control characters in Red (tdp note).
func redNames(line string, chars []string, tabW int) string {
	var b strings.Builder
	cell, from := 0, 0
	for _, c := range chars {
		d := drawn(c, tabW)
		w := Width(d)
		if c != "\t" && d != c {
			b.WriteString(Cut(line, from, cell))
			b.WriteString(Fg(Red).Render(d))
			from = cell + w
		}
		cell += w
	}
	if from == 0 {
		return line
	}
	b.WriteString(Cut(line, from, cell))
	return b.String()
}

// crlf is line without the \r of a CRLF line break: it is part of the
// break, not drawn (tdp note).
func crlf(line string) (string, bool) { return strings.CutSuffix(line, "\r") }

// linesFrom is h lines of r from line first on (counted from 1), and the
// number of the first one, which is earlier when r ends sooner; it reads at
// most 8 MB.
func linesFrom(r io.Reader, first, h int) (int, string) {
	sc := bufio.NewScanner(io.LimitReader(r, 8<<20))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	var window []string
	n := 0
	for sc.Scan() && n < first-1+h {
		n++
		window = append(window, sc.Text())
		if len(window) > h {
			window = window[1:]
		}
	}
	return max(n-len(window)+1, 1), strings.Join(window, "\n")
}

// highlight is text in the colours of catppuccin-mocha (foreground only),
// its line numbers in Overlay0; tabs and control characters as a viewer
// draws them. A file with colour codes of its own is drawn in them instead,
// not lexed; a \r inside a line leaves its last state (tdp
// file-picker).
func highlight(p, text string, h, tabW, first int) []string {
	raw := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(raw) > h {
		raw = raw[:h]
	}
	numW := len(strconv.Itoa(first + len(raw) - 1))
	if strings.Contains(text, "\x1b[") {
		out := make([]string, len(raw))
		for i, l := range raw {
			l = lastState(cutCR(l))
			out[i] = Fg(Overlay0).Render(PadLeft(strconv.Itoa(first+i), numW)) + " " + onText(readingLine(l, tabW))
		}
		return out
	}
	lines := make([]string, len(raw))
	for i := range raw {
		raw[i], _ = crlf(raw[i])
		raw[i] = lastState(raw[i])
		lines[i] = fileText(raw[i], tabW)
	}
	lexer := lexers.Match(filepath.Base(p))
	if lexer == nil {
		lexer = lexers.Fallback
	}
	coloured := colourise(lexer, lines)
	out := make([]string, len(lines))
	for i := range lines {
		out[i] = Fg(Overlay0).Render(PadLeft(strconv.Itoa(first+i), numW)) + " " + redNames(coloured[i], graphemes(raw[i]), tabW)
	}
	return out
}

// colourise is lines in the colours of catppuccin-mocha as lexer reads them,
// foreground only; what it leaves uncoloured is Text. A viewer colours its
// content with it too (tdp note).
func colourise(lexer chroma.Lexer, lines []string) []string {
	style := styles.Get("catppuccin-mocha")
	coloured := make([]strings.Builder, len(lines))
	row := 0
	if it, err := chroma.Coalesce(lexer).Tokenise(nil, strings.Join(lines, "\n")); err == nil {
		for tok := it(); tok != chroma.EOF; tok = it() {
			c := style.Get(tok.Type).Colour
			parts := strings.Split(tok.Value, "\n")
			for i, part := range parts {
				if i > 0 {
					row++
				}
				if row >= len(coloured) || part == "" {
					continue
				}
				if c.IsSet() {
					coloured[row].WriteString(Fg(Color{RGB: RGB{c.Red(), c.Green(), c.Blue()}, Set: true}).Render(part))
				} else {
					coloured[row].WriteString(Fg(Text).Render(part))
				}
			}
		}
	}
	out := make([]string, len(lines))
	for i := range lines {
		out[i] = coloured[i].String()
		if out[i] == "" && lines[i] != "" {
			out[i] = Fg(Text).Render(lines[i])
		}
	}
	return out
}

// hexDump is eight bytes a row in hex, then their ASCII: 33 cells, whatever
// the preview's width.
func hexDump(b []byte, h int) []string {
	var out []string
	for off := 0; off < len(b) && len(out) < h; off += 8 {
		row := b[off:min(off+8, len(b))]
		var hexes, ascii strings.Builder
		for i := range 8 {
			if i < len(row) {
				fmt.Fprintf(&hexes, "%02x ", row[i])
				c := row[i]
				if c < 0x20 || c > 0x7e {
					c = '.'
				}
				ascii.WriteByte(c)
			} else {
				hexes.WriteString("   ")
			}
		}
		out = append(out, Fg(Text).Render(hexes.String())+" "+Fg(Overlay0).Render(ascii.String()))
	}
	return out
}

// pdfText is the page count in grey, a blank line, then the text of the
// first 20 pages, 128 KB of it at most.
func pdfText(fsys PreviewFS, p string, sz int64, h int) (out []string) {
	defer func() {
		if recover() != nil {
			out = []string{Fg(Overlay0).Render("(unreadable)")}
		}
	}()
	f, err := fsys.Open(p)
	if err != nil {
		return []string{Fg(Overlay0).Render("(unreadable)")}
	}
	defer f.Close()
	r, err := pdf.NewReader(f, sz)
	if err != nil {
		return []string{Fg(Overlay0).Render("(unreadable)")}
	}
	pages := r.NumPage()
	unit := "pages"
	if pages == 1 {
		unit = "page"
	}
	out = []string{Fg(Overlay0).Render(fmt.Sprintf("%d %s", pages, unit)), ""}
	read, found := 0, false
	for i := 1; i <= min(pages, 20) && len(out) < h && read < 128<<10; i++ {
		text, err := r.Page(i).GetPlainText(nil)
		if err != nil {
			continue
		}
		text = text[:min(len(text), 128<<10-read)]
		read += len(text)
		for _, l := range strings.Split(text, "\n") {
			if strings.TrimSpace(l) != "" && len(out) < h {
				out = append(out, Fg(Text).Render(l))
				found = true
			}
		}
	}
	if !found {
		out = append(out, Fg(Overlay0).Render("(no extractable text)"))
	}
	return out
}

// treeNode is one entry of a tree: a directory's or an archive's.
type treeNode struct {
	name, path string
	dir        bool
	kids       []*treeNode
	loaded     bool
}

// drawTree draws entries three levels deep, depth first: the first level
// without branches, the rest with ├─ └─ │ in Overlay0; at most h rows, the
// last one "…" when there is more (tdp file-picker).
func drawTree(top []*treeNode, h int, load func(*treeNode) []*treeNode, icon func(string, bool) (string, Color)) []string {
	var out []string
	more := false
	var walk func(nodes []*treeNode, depth int, prefix string)
	walk = func(nodes []*treeNode, depth int, prefix string) {
		for i, n := range nodes {
			if len(out) >= h {
				more = true
				return
			}
			last := i == len(nodes)-1
			lead, next := "", "  " // the second level two cells in (tdp file-picker)
			if depth > 0 {
				lead, next = prefix+"├─ ", prefix+"│  "
				if last {
					lead, next = prefix+"└─ ", prefix+"   "
				}
			}
			glyph, colour := icon(n.path, n.dir)
			out = append(out, Fg(Overlay0).Render(lead)+Fg(cmpColor(colour, Text)).Render(glyph)+" "+Fg(Text).Render(n.name))
			if n.dir && depth < 2 {
				walk(load(n), depth+1, next)
			}
		}
	}
	walk(top, 0, "")
	if more && len(out) > 0 {
		out[len(out)-1] = Fg(Overlay0).Render("…")
	}
	return out
}

func defaultIcon(_ string, dir bool) (string, Color) {
	if dir {
		return glyphFolder, Color{}
	}
	return glyphFile, Color{}
}

func dirTree(fsys PreviewFS, p string, h int, icon func(string, bool) (string, Color)) []string {
	if icon == nil {
		icon = defaultIcon
	}
	load := func(n *treeNode) []*treeNode {
		entries, err := dirEntriesIn(fsys, n.path, false)
		if err != nil {
			return nil
		}
		var out []*treeNode
		for _, e := range entries {
			out = append(out, &treeNode{name: e.name, path: e.path, dir: e.dir})
		}
		return out
	}
	if _, err := fsys.ReadDir(p); err != nil {
		return []string{Fg(Overlay0).Render("(permission denied)")}
	}
	top := load(&treeNode{path: p})
	if len(top) == 0 {
		return []string{Fg(Overlay0).Render("(empty)")}
	}
	return drawTree(top, h, load, icon)
}

// archiveTree lists what an archive holds, as a directory's tree.
func archiveTree(fsys PreviewFS, p string, h int, icon func(string, bool) (string, Color)) []string {
	if icon == nil {
		icon = defaultIcon
	}
	names, err := archiveNames(fsys, p)
	if err != nil {
		return []string{Fg(Overlay0).Render("(unreadable)")}
	}
	root := &treeNode{dir: true}
	for _, name := range names {
		dir := strings.HasSuffix(name, "/")
		parts := strings.Split(strings.Trim(name, "/"), "/")
		node := root
		for i, part := range parts {
			if part == "" || strings.HasPrefix(part, ".") {
				break
			}
			var kid *treeNode
			for _, k := range node.kids {
				if k.name == part {
					kid = k
				}
			}
			if kid == nil {
				kid = &treeNode{name: part, path: part, dir: dir || i < len(parts)-1}
				node.kids = append(node.kids, kid)
			}
			node = kid
		}
	}
	var sortTree func(n *treeNode)
	sortTree = func(n *treeNode) {
		slices.SortStableFunc(n.kids, func(a, b *treeNode) int {
			if a.dir != b.dir {
				if a.dir {
					return -1
				}
				return 1
			}
			return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
		})
		for _, k := range n.kids {
			sortTree(k)
		}
	}
	sortTree(root)
	if len(root.kids) == 0 {
		return []string{Fg(Overlay0).Render("(empty)")}
	}
	return drawTree(root.kids, h, func(n *treeNode) []*treeNode { return n.kids }, icon)
}

func archiveNames(fsys PreviewFS, p string) ([]string, error) {
	lower := strings.ToLower(p)
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if strings.HasSuffix(lower, ".zip") {
		st, err := fsys.Stat(p)
		if err != nil {
			return nil, err
		}
		r, err := zip.NewReader(f, st.Size())
		if err != nil {
			return nil, err
		}
		var names []string
		for _, f := range r.File {
			names = append(names, f.Name)
		}
		return names, nil
	}
	var src io.Reader = f
	switch {
	case strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".tgz"):
		z, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		src = z
	case strings.HasSuffix(lower, ".bz2") || strings.HasSuffix(lower, ".tbz2"):
		src = bzip2.NewReader(f)
	}
	tr := tar.NewReader(src)
	var names []string
	for len(names) < 10000 {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return names, err
		}
		name := hdr.Name
		if hdr.Typeflag == tar.TypeDir && !strings.HasSuffix(name, "/") {
			name += "/"
		}
		names = append(names, name)
	}
	return names, nil
}

// fileInfo is the four rows under a preview (tdp file-picker):
// TYPE, SIZE, MODIFIED, MODE; labels in the layer colour. TYPE is the
// preview's kind, the size in 1024s with one decimal (a directory's in items,
// hidden ones left out), the time local to the minute, the mode written as
// ls -l does. A link's size, time and mode are its target's; what cannot be
// had is —.
func fileInfo(p string, w int, layer Color) []string { return fileInfoIn(nil, p, w, layer) }

// fileInfoIn is fileInfo of p in fsys (the local disk when nil).
func fileInfoIn(fsys PreviewFS, p string, w int, layer Color) []string {
	fsys = orLocal(fsys)
	rows := [4][2]string{{"TYPE", ""}, {"SIZE", ""}, {"MODIFIED", ""}, {"MODE", ""}}
	if lst, err := fsys.Lstat(p); err == nil {
		st := lst
		if lst.Mode()&fs.ModeSymlink != 0 {
			target, _ := fsys.ReadLink(p)
			rows[0][1] = "link → " + target
			s, err := fsys.Stat(p)
			if err != nil {
				// A broken link: there is no target to tell of.
				rows[1][1], rows[2][1], rows[3][1] = "—", "—", "—"
				return infoRows(rows, w, layer)
			}
			st = s
		}
		switch {
		case st.IsDir():
			if rows[0][1] == "" {
				rows[0][1] = "directory"
			}
			if entries, err := dirEntriesIn(fsys, p, false); err == nil {
				rows[1][1] = items(len(entries))
			} else {
				rows[1][1] = "—" // it cannot be read: not 0 items, which reads as empty
			}
		default:
			if rows[0][1] == "" {
				head := make([]byte, 4096)
				if f, err := fsys.Open(p); err == nil {
					n, _ := io.ReadFull(f, head)
					head = head[:n]
					f.Close()
				}
				rows[0][1] = fileKind(p, head)
			}
			rows[1][1] = size(st.Size())
		}
		rows[2][1] = st.ModTime().Format("2006-01-02 15:04")
		rows[3][1] = lsMode(st.Mode())
	} else {
		// Gone since it was listed (deleted, moved): nothing to tell of.
		rows[0][1], rows[1][1], rows[2][1], rows[3][1] = "—", "—", "—", "—"
	}
	return infoRows(rows, w, layer)
}

// items is a directory's size: "12 items", "1 item".
func items(n int) string {
	if n == 1 {
		return "1 item"
	}
	return strconv.Itoa(n) + " items"
}

func infoRows(rows [4][2]string, w int, layer Color) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = Fit(Fg(layer).Render(Pad(r[0], 8))+"  "+Fg(Text).Render(r[1]), w)
	}
	return out
}
