package msgpack

import (
	"bytes"
	"testing"

	"github.com/webmafia/fast/buffer"
)

func TestRelease(t *testing.T) {
	tests := []struct {
		name        string
		initialData []byte
		rn          int
		releaseN    int
		expectedBuf []byte
		expectedN   int
	}{
		{
			name:        "Release with valid n",
			initialData: []byte{'A', 'B', 'C', 'D', 'E', 'F'},
			rn:          3,
			releaseN:    2,
			expectedBuf: []byte{'A', 'B', 'D', 'E', 'F'},
			expectedN:   5,
		},
		{
			name:        "Release everything",
			initialData: []byte{'A', 'B', 'C'},
			rn:          3,
			releaseN:    0,
			expectedBuf: []byte{},
			expectedN:   0,
		},
		{
			name:        "Release with n beyond r.n",
			initialData: []byte{'A', 'B', 'C'},
			rn:          2,
			releaseN:    3,
			expectedBuf: []byte{'A', 'B', 'C'},
			expectedN:   2,
		},
		{
			name:        "Release with negative n",
			initialData: []byte{'A', 'B', 'C'},
			rn:          2,
			releaseN:    -1,
			expectedBuf: []byte{'A', 'B', 'C'},
			expectedN:   2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := buffer.NewBuffer(64)
			buf.B = append(buf.B, test.initialData...)

			r := Reader2{
				b: buf,
				n: test.rn,
			}

			r.Release(test.releaseN)

			if !bytes.Equal(r.b.B, test.expectedBuf) {
				t.Errorf("buffer mismatch: got %v, want %v", r.b.B, test.expectedBuf)
			}

			if r.n != test.expectedN {
				t.Errorf("position mismatch: got %d, want %d", r.n, test.expectedN)
			}
		})
	}
}
