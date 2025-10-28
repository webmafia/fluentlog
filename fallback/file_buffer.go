package fallback

import (
	"io"
	"os"

	"github.com/klauspost/compress/gzip"
)

var _ Fallback = (*FileBuffer)(nil)

type FileBuffer struct {
	path string
	f    *os.File
	w    *gzip.Writer
}

func NewFileBuffer(path string) *FileBuffer {
	return &FileBuffer{
		path: path,
	}
}

func (f *FileBuffer) ensureFile() (err error) {
	if f.f != nil {
		return
	}

	return f.initFile()
}

func (f *FileBuffer) initFile() (err error) {
	if f.f, err = os.OpenFile(f.path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600); err != nil {
		return
	}

	f.w = gzip.NewWriter(f.f)

	return
}

// Write implements io.WriteCloser.
func (f *FileBuffer) Write(p []byte) (n int, err error) {
	if err = f.ensureFile(); err != nil {
		return
	}

	return f.w.Write(p)
}

// Close implements io.WriteCloser.
func (f *FileBuffer) Close() (err error) {
	if f.f == nil {
		return
	}

	if err = f.w.Close(); err != nil {
		return
	}

	if err = f.f.Close(); err != nil {
		return
	}

	f.f = nil
	return
}

// HasData implements Fallback.
func (f *FileBuffer) HasData() (ok bool, err error) {
	// TODO: Implement this method.
	panic("unimplemented")
}

// Reader implements Fallback.
func (f *FileBuffer) Reader(fn func(n int, r io.Reader) error) (err error) {
	// TODO: Implement this method.
	panic("unimplemented")
}
