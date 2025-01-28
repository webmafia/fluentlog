package fuzz

import (
	"sync"
	"testing"

	"github.com/webmafia/fluentlog/internal/msgpack"
	"github.com/webmafia/fluentlog/internal/msgpack/types"
)

func FuzzIterator(f *testing.F) {
	f.Add(buildComplexMessage())

	pool := sync.Pool{
		New: func() any {
			iter := msgpack.NewIterator(nil)
			return &iter
		},
	}

	f.Fuzz(func(t *testing.T, msg []byte) {
		iter := pool.Get().(*msgpack.Iterator)
		defer pool.Put(iter)

		iter.ResetBytes(msg)

		for iter.Next() {
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
				_ = iter.Bin()

			case types.Ext:
				_ = iter.Time()

			}
		}

		if err := iter.Error(); err != nil {
			t.Error(err)
		}
	})
}
