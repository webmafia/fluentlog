package transport

import "github.com/webmafia/fast/ringbuf"

func isGzip(r *ringbuf.LimitedReader) (ok bool, err error) {
	magicNumbers, err := r.Peek(3)

	if err != nil {
		return
	}

	// Bounds check hint to compiler; see golang.org/issue/14808
	_ = magicNumbers[2]

	ok = (magicNumbers[0] == 0x1f &&
		magicNumbers[1] == 0x8b &&
		magicNumbers[2] == 8)

	return
}
