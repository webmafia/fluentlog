package fallback

import (
	"io"
	"os"
	"path"
	"sync"

	"github.com/klauspost/compress/gzip"
)

const perm os.FileMode = 0600

var _ Fallback = (*DirBuffer)(nil)

type DirBuffer struct {
	dir     string
	rName   string
	wName   string
	read    *os.File
	write   *os.File
	writeGz *gzip.Writer
	mu      sync.Mutex
}

// A disk-based ping-pong buffer. Reads and writes can be done simultaneously.
func NewDirBuffer(dir string) *DirBuffer {
	return &DirBuffer{
		dir:   dir,
		rName: path.Join(dir, "ping.bin"),
		wName: path.Join(dir, "pong.bin"),
	}
}

func (f *DirBuffer) ensureReadFile() (err error) {
	if f.read != nil {
		return
	}

	return f.initReadFile()
}

func (f *DirBuffer) initReadFile() (err error) {
	if err = f.initDir(); err != nil {
		return
	}

	if f.read, err = os.OpenFile(f.rName, os.O_RDWR, perm); err != nil {
		return
	}

	return
}

func (f *DirBuffer) closeReadFile() (err error) {
	if f.read != nil {
		err = f.read.Close()
		f.read = nil
	}

	return
}

func (f *DirBuffer) ensureWriteFile() (err error) {
	if f.write != nil {
		return
	}

	return f.initWriteFile()
}

func (f *DirBuffer) initWriteFile() (err error) {
	if err = f.initDir(); err != nil {
		return
	}

	if f.write, err = os.OpenFile(f.wName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, perm); err != nil {
		return
	}

	if f.writeGz == nil {
		f.writeGz = gzip.NewWriter(f.write)
	} else {
		f.writeGz.Reset(f.write)
	}

	return
}

func (f *DirBuffer) closeWriteFile() (err error) {
	if f.write != nil {
		if f.writeGz != nil {
			if err = f.writeGz.Close(); err != nil {
				return
			}

			f.writeGz.Reset(nil)
		}

		err = f.write.Close()
		f.write = nil
	}

	return
}

func (d *DirBuffer) initDir() (err error) {
	return os.MkdirAll(d.dir, 0700)
}

// Write implements io.WriteCloser.
func (f *DirBuffer) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err = f.ensureWriteFile(); err != nil {
		return
	}

	return f.writeGz.Write(p)
}

func (d *DirBuffer) HasData() (ok bool, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err = d.close(); err != nil {
		return
	}

	rSize := d.fileSize(d.rName)
	wSize := d.fileSize(d.wName)

	// Nothing to read
	if rSize == 0 && wSize == 0 {
		return
	}

	ok = true

	if wSize > 0 {
		if rSize == 0 {
			return
		}

		// If we came here, it means we have data on both files. Merge them.
		if err = d.initReadFile(); err != nil {
			return
		}

		if err = d.initWriteFile(); err != nil {
			return
		}

		if _, err = io.Copy(d.write, d.read); err != nil {
			return
		}

		if err = d.read.Truncate(0); err != nil {
			return
		}

		err = d.close()
		return
	}

	// If we came here, it means we have data in the "wrong" file. Switch them.
	d.rName, d.wName = d.wName, d.rName
	return
}

func (*DirBuffer) fileSize(name string) int64 {
	fi, err := os.Stat(name)

	if err != nil {
		return 0
	}

	return fi.Size()
}

func (d *DirBuffer) Reader(fn func(n int, r io.Reader) error) (err error) {
	if err = d.switchFiles(); err != nil {
		return
	}

	if err = d.ensureReadFile(); err != nil {
		return
	}

	defer d.closeReadFile()

	stat, err := d.read.Stat()

	if err != nil {
		return
	}

	if err = fn(int(stat.Size()), d.read); err != nil {
		return
	}

	// return nil
	return d.read.Truncate(0)
}

func (d *DirBuffer) switchFiles() (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err = d.close(); err != nil {
		return
	}

	// Do the "ping-pong" switch
	d.rName, d.wName = d.wName, d.rName

	return
}

// Close implements io.WriteCloser.
func (f *DirBuffer) Close() (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.close()
}

func (f *DirBuffer) close() (err error) {
	if err = f.closeReadFile(); err != nil {
		return
	}

	return f.closeWriteFile()
}
