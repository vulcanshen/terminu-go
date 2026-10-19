package terminu

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// PreviewFS is where a file preview reads: the local disk when nil, or a file
// system the app gives, such as a remote host's over SFTP, so its files get
// the same preview, tree and facts (components/input/file-picker, the
// preview). Paths are the app's own, slash-separated as SFTP's are.
type PreviewFS interface {
	Open(name string) (PreviewFile, error)
	Stat(name string) (fs.FileInfo, error)
	// Lstat does not follow a link, ReadLink says where one points: a link's
	// TYPE is "link → <target>".
	Lstat(name string) (fs.FileInfo, error)
	ReadLink(name string) (string, error)
	ReadDir(name string) ([]fs.FileInfo, error)
}

// PreviewFile is an open file of a PreviewFS: read from its start, or at an
// offset (a PDF, a zip).
type PreviewFile interface {
	io.Reader
	io.ReaderAt
	io.Closer
}

// localFS is the local disk.
type localFS struct{}

func (localFS) Open(name string) (PreviewFile, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (localFS) Stat(name string) (fs.FileInfo, error)  { return os.Stat(name) }
func (localFS) Lstat(name string) (fs.FileInfo, error) { return os.Lstat(name) }
func (localFS) ReadLink(name string) (string, error)   { return os.Readlink(name) }

func (localFS) ReadDir(name string) ([]fs.FileInfo, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}
	var out []fs.FileInfo
	for _, e := range entries {
		if info, err := e.Info(); err == nil {
			out = append(out, info)
		}
	}
	return out, nil
}

// orLocal is fsys, the local disk when nil.
func orLocal(fsys PreviewFS) PreviewFS {
	if fsys == nil {
		return localFS{}
	}
	return fsys
}

// joinIn joins a directory and a name as fsys writes paths.
func joinIn(fsys PreviewFS, dir, name string) string {
	if _, ok := fsys.(localFS); ok {
		return filepath.Join(dir, name)
	}
	return path.Join(dir, name)
}
