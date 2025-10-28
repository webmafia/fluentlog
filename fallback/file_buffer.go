package fallback

import (
	"io"
	"os"
	"sync"

	"github.com/klauspost/compress/gzip"
)

var _ Fallback = (*FileBuffer)(nil)

// A file-based buffer. Reads and writes are blocking each other - always prefer using the
// DirBuffer for this reason.
type FileBuffer struct {
	path string
	f    *os.File
	w    *gzip.Writer
	mu   sync.Mutex
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if err = f.ensureFile(); err != nil {
		return
	}

	return f.w.Write(p)
}

// Close implements io.WriteCloser.
func (f *FileBuffer) Close() (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()

	fi, err := os.Stat(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			// No file means no data.
			return false, nil
		}
		return false, err
	}

	return fi.Size() > 0, nil
}

// Reader implements Fallback.
func (f *FileBuffer) Reader(fn func(n int, r io.Reader) error) (err error) {
	f.mu.Lock()
	// Ensure any pending gzip data is flushed to disk.
	if f.f != nil {
		if err = f.w.Close(); err != nil {
			f.mu.Unlock()
			return
		}
		if err = f.f.Close(); err != nil {
			f.mu.Unlock()
			return
		}
		f.f = nil
	}
	f.mu.Unlock()

	// Open file for read/write so we can truncate after reading.
	var rf *os.File
	if rf, err = os.OpenFile(f.path, os.O_RDWR, 0600); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return
	}
	defer rf.Close()

	var fi os.FileInfo
	if fi, err = rf.Stat(); err != nil {
		return
	}

	size := int(fi.Size())
	if size == 0 {
		// Nothing to read; ensure truncated.
		_ = rf.Truncate(0)
		return nil
	}

	if err = fn(size, rf); err != nil {
		return
	}

	// Clear consumed data.
	return rf.Truncate(0)
}
