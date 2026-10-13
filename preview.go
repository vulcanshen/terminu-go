package terminu

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
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
// (components/input/file-picker).
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
// (components/input/file-picker): a directory's or an archive's tree, an
// image as a data URI, a PDF's first pages, a text file in colour with line
// numbers, anything else in hex. With hideSecrets, a private key shows its
// first line only and any other file only its owner may read shows nothing.
func filePreview(p string, w, h int, hideSecrets bool, icon func(string, bool) (string, Color)) []string {
	grey := func(s string) []string { return []string{Fg(Overlay0).Render(s)} }
	st, err := os.Stat(p)
	if err != nil {
		return grey("(unreadable)")
	}
	if st.IsDir() {
		return dirTree(p, h, icon)
	}
	f, err := os.Open(p)
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
		return archiveTree(p, h, icon)
	case "image":
		return imagePreview(p, st.Size(), w, h)
	case "PDF":
		return pdfText(p, st.Size(), h)
	case "text", "private key":
		return highlight(p, string(trimPartialRune(head)), h)
	}
	return hexDump(head, h)
}

// imagePreview is the type and the size in grey, a blank line, then the
// image as a data URI folded at the preview's width; over 1 MB, a line
// saying so. Only what is in sight is read.
func imagePreview(p string, sz int64, w, h int) []string {
	if sz > 1<<20 {
		return []string{Fg(Overlay0).Render("(" + size(sz) + " — too large to inline as a data URI)")}
	}
	w = max(w, 1)
	mime := imageMIME(p)
	out := []string{Fg(Overlay0).Render("data:" + mime + " · " + size(sz)), ""}
	prefix := "data:" + mime + ";base64,"
	need := max((h-len(out))*w-len(prefix), 0)
	f, err := os.Open(p)
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

// highlight is text in the colours of catppuccin-mocha (foreground only),
// its line numbers in Overlay0.
func highlight(p, text string, h int) []string {
	text = strings.ReplaceAll(text, "\t", "    ")
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	lexer := lexers.Match(filepath.Base(p))
	if lexer == nil {
		lexer = lexers.Fallback
	}
	coloured := colourise(lexer, lines)
	numW := len(strconv.Itoa(len(lines)))
	out := make([]string, len(lines))
	for i := range lines {
		out[i] = Fg(Overlay0).Render(PadLeft(strconv.Itoa(i+1), numW)) + " " + coloured[i]
	}
	return out
}

// colourise is lines in the colours of catppuccin-mocha as lexer reads them,
// foreground only; what it leaves uncoloured is Text. A viewer colours its
// content with it too (components/dialog/note).
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
func pdfText(p string, sz int64, h int) (out []string) {
	defer func() {
		if recover() != nil {
			out = []string{Fg(Overlay0).Render("(unreadable)")}
		}
	}()
	f, err := os.Open(p)
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
// last one "…" when there is more (components/input/file-picker).
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
			lead, next := "", ""
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

func dirTree(p string, h int, icon func(string, bool) (string, Color)) []string {
	if icon == nil {
		icon = defaultIcon
	}
	load := func(n *treeNode) []*treeNode {
		entries, err := dirEntries(n.path, false)
		if err != nil {
			return nil
		}
		var out []*treeNode
		for _, e := range entries {
			out = append(out, &treeNode{name: e.name, path: e.path, dir: e.dir})
		}
		return out
	}
	if _, err := os.ReadDir(p); err != nil {
		return []string{Fg(Overlay0).Render("(permission denied)")}
	}
	top := load(&treeNode{path: p})
	if len(top) == 0 {
		return []string{Fg(Overlay0).Render("(empty)")}
	}
	return drawTree(top, h, load, icon)
}

// archiveTree lists what an archive holds, as a directory's tree.
func archiveTree(p string, h int, icon func(string, bool) (string, Color)) []string {
	if icon == nil {
		icon = defaultIcon
	}
	names, err := archiveNames(p)
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

func archiveNames(p string) ([]string, error) {
	lower := strings.ToLower(p)
	if strings.HasSuffix(lower, ".zip") {
		r, err := zip.OpenReader(p)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		var names []string
		for _, f := range r.File {
			names = append(names, f.Name)
		}
		return names, nil
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
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

// fileInfo is the four rows under a preview (components/input/file-picker):
// TYPE, SIZE, MODIFIED, MODE; labels in the layer colour. TYPE is the
// preview's kind, the size in 1024s with one decimal (a directory's in items,
// hidden ones left out), the time local to the minute, the mode written as
// ls -l does. A link's size, time and mode are its target's; what cannot be
// had is —.
func fileInfo(p string, w int, layer Color) []string {
	rows := [4][2]string{{"TYPE", ""}, {"SIZE", ""}, {"MODIFIED", ""}, {"MODE", ""}}
	if lst, err := os.Lstat(p); err == nil {
		st := lst
		if lst.Mode()&fs.ModeSymlink != 0 {
			target, _ := os.Readlink(p)
			rows[0][1] = "link → " + target
			s, err := os.Stat(p)
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
			if entries, err := dirEntries(p, false); err == nil {
				rows[1][1] = strconv.Itoa(len(entries)) + " items"
			} else {
				rows[1][1] = "—" // it cannot be read: not 0 items, which reads as empty
			}
		default:
			if rows[0][1] == "" {
				head := make([]byte, 4096)
				if f, err := os.Open(p); err == nil {
					n, _ := io.ReadFull(f, head)
					head = head[:n]
					f.Close()
				}
				rows[0][1] = fileKind(p, head)
			}
			rows[1][1] = size(st.Size())
		}
		rows[2][1] = st.ModTime().Format("2006-01-02 15:04")
		rows[3][1] = st.Mode().String()
	}
	return infoRows(rows, w, layer)
}

func infoRows(rows [4][2]string, w int, layer Color) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = Fit(Fg(layer).Render(Pad(r[0], 8))+"  "+Fg(Text).Render(r[1]), w)
	}
	return out
}
