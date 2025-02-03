package msgpack

import (
	"fmt"
	"io"
	"math"
	"sync"
	"testing"

	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

// BuildComplexMessage creates a deep, complex MessagePack message using Append* functions.
func buildComplexMessage(withBin ...bool) []byte {
	var data []byte

	items := 3

	if len(withBin) > 0 && withBin[0] {
		items++
	}

	// Example: A MessagePack map with nested structures
	data = AppendMapHeader(data, items) // Map with 3 key-value pairs

	// Key 1: "simple_key" -> "simple_value"
	data = AppendString(data, "simple_key")
	data = AppendString(data, "simple_value")

	if len(withBin) > 0 && withBin[0] {
		data = AppendString(data, "some_binary")
		data = AppendBinary(data, make([]byte, math.MaxInt16))
	}

	// Key 2: "nested_array" -> [1, 2, [3, 4, 5]]
	data = AppendString(data, "nested_array")
	data = AppendArrayHeader(data, 3)
	data = AppendInt(data, 1)
	data = AppendInt(data, 2)
	data = AppendArrayHeader(data, 3)
	data = AppendInt(data, 3)
	data = AppendInt(data, 4)
	data = AppendInt(data, 5)

	// Key 3: "nested_map" -> { "inner_key": [true, false], "float_key": 3.14159 }
	data = AppendString(data, "nested_map")
	data = AppendMapHeader(data, 2)
	data = AppendString(data, "inner_key")
	data = AppendArrayHeader(data, 2)
	data = AppendBool(data, true)
	data = AppendBool(data, false)
	data = AppendString(data, "float_key")
	data = AppendFloat(data, 3.14159)

	return data
}

func Example_buildComplexMessage() {
	data := buildComplexMessage()
	fmt.Println(data)

	// Output:
	//
	// [131 170 115 105 109 112 108 101 95 107 101 121 172 115 105 109 112 108 101 95 118 97 108 117 101 172 110 101 115 116 101 100 95 97 114 114 97 121 147 1 2 147 3 4 5 170 110 101 115 116 101 100 95 109 97 112 130 169 105 110 110 101 114 95 107 101 121 146 195 194 169 102 108 111 97 116 95 107 101 121 203 64 9 33 249 240 27 134 110]
}

func Example_iterateComplexMessage() {
	data := buildComplexMessage()
	iter := NewIterator(nil)
	iter.ResetBytes(data)

	for iter.Next() {
		fmt.Println(iter.Type())

		if iter.Type() != types.Array && iter.Type() != types.Map {
			iter.Skip()
		}
	}

	// Output:
	//
	// map
	// str
	// str
	// str
	// array
	// uint
	// uint
	// array
	// uint
	// uint
	// uint
	// str
	// map
	// str
	// array
	// bool
	// bool
	// str
	// float
}

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
			data:           buildComplexMessage(true),
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
			iter := NewIterator(nil)
			return &iter
		},
	}

	f.Fuzz(func(t *testing.T, msg []byte, maxBufSize uint16, copyN int16, release bool, forceRelease bool, skipExplicitly bool, skipImplicitly bool) {
		iter := pool.Get().(*Iterator)
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
				// iter.Skip()
				if l := iter.Len(); l > 1024*1024 {
					t.Skipf("skipped bin of size %d", l)
				}

				// TODO: Support partial reads
				// _, err := io.CopyN(io.Discard, iter.BinReader(), int64(copyN))
				_, err := io.Copy(io.Discard, iter.Reader())

				if err != nil {
					t.Log(err)
				}

			case types.Ext:
				_ = iter.Time()

			default:
				t.Log("invalid type")

			}

			if err := validateState(iter); err != nil {
				t.Error(err)
			}

			// TODO: Test again once this fuzz issue has been fixed: https://github.com/golang/go/issues/56238
			// if release {
			// 	iter.Release(forceRelease)
			// }
		}
	})
}

func validateState(iter *Iterator) (err error) {
	// if iter.n < 0 {
	// 	return fmt.Errorf("iter.n (%d) is negative", iter.n)
	// }

	// if iter.n > len(iter.buf) {
	// 	return fmt.Errorf("iter.n (%d) overflows iter.buf[0:%d]", iter.n, len(iter.buf))
	// }

	// if iter.t0 < 0 {
	// 	return fmt.Errorf("iter.t0 (%d) is negative", iter.t0)
	// }

	// if iter.t0 > iter.t1 {
	// 	return fmt.Errorf("iter.t0 (%d) exceeds iter.t1 (%d)", iter.t0, iter.t1)
	// }

	// if iter.t1 > iter.t2 {
	// 	return fmt.Errorf("iter.t1 (%d) exceeds iter.t2 (%d)", iter.t1, iter.t2)
	// }

	// if cap(iter.buf) > iter.max {
	// 	return fmt.Errorf("iter.buf[0:%d:%d] exceeds iter.max (%d)", len(iter.buf), cap(iter.buf), iter.max)
	// }

	// if iter.remain < 0 {
	// 	return fmt.Errorf("iter.remain (%d) is negative", iter.remain)
	// }

	// if iter.remain > (iter.t2 - iter.t1) {
	// 	return fmt.Errorf("iter.remain (%d) exceeds value size (%d)", iter.remain, iter.t2-iter.t1)
	// }

	// if iter.rp < 0 {
	// 	return fmt.Errorf("iter.rp (%d) is negative", iter.rp)
	// }

	// if iter.rp > iter.n {
	// 	return fmt.Errorf("iter.rp (%d) exceeds iter.n (%d)", iter.rp, iter.n)
	// }

	return
}
