package fuzz

import (
	"io"
	"math"
	"sync"
	"testing"

	"github.com/webmafia/fluentlog/internal/msgpack"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func FuzzVaryingIterator(f *testing.F) {
	type testCase struct {
		data           []byte
		maxBufSize     uint16
		copyN          int16
		release        bool
		forceRelease   bool
		skipExplicitly bool
		skipImplicitly bool
	}

	cases := []testCase{
		{
			data:           buildComplexMessage(),
			maxBufSize:     4096,
			copyN:          math.MaxInt16,
			release:        false,
			forceRelease:   false,
			skipExplicitly: false,
			skipImplicitly: false,
		},
	}

	for _, c := range cases {
		f.Add(c.data, c.maxBufSize, c.copyN, c.release, c.forceRelease, c.skipExplicitly, c.skipImplicitly)
	}

	pool := sync.Pool{
		New: func() any {
			iter := msgpack.NewIterator(nil)
			return &iter
		},
	}

	f.Fuzz(func(t *testing.T, msg []byte, maxBufSize uint16, copyN int16, release bool, forceRelease bool, skipExplicitly bool, skipImplicitly bool) {
		iter := pool.Get().(*msgpack.Iterator)
		defer pool.Put(iter)

		iter.ResetBytes(msg, int(maxBufSize))

		for iter.Next() {
			if skipExplicitly {
				iter.Skip()
				continue
			}

			if skipImplicitly {
				continue
			}

			switch iter.Type() {

			case types.Bool:
				_ = iter.Bool()

			case types.Int:
				_ = iter.Int()

			case types.Uint:
				_ = iter.Uint()

			case types.Float:
				_ = iter.Float()

			case types.Str:
				_ = iter.Str()

			case types.Bin:
				if l := iter.Len(); l > 1024*1024 {
					t.Skipf("skipped bin of size %d", l)
				}

				_, err := io.CopyN(io.Discard, iter.BinReader(), int64(copyN))

				if err != nil {
					t.Log(err)
				}

			case types.Ext:
				_ = iter.Time()

			default:
				t.Log("invalid type")

			}

			if release {
				iter.Release(forceRelease)
			}
		}
	})
}
