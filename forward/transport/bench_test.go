package transport

import (
	"bytes"
	stdGzip "compress/gzip"
	"slices"
	"testing"
	"time"

	"github.com/webmafia/fluentlog/internal/gzip"
	"github.com/webmafia/fluentlog/pkg/msgpack"
)

func mockMessageMode(count int) []byte {
	ts := time.Now()

	var msg []byte
	msg = msgpack.AppendArrayHeader(msg, 3)
	msg = msgpack.AppendString(msg, "tag")
	msg = msgpack.AppendTimestamp(msg, ts)
	msg = msgpack.AppendMapHeader(msg, 1)
	msg = msgpack.AppendString(msg, "key")
	msg = msgpack.AppendString(msg, "value")

	buf := make([]byte, 0, len(msg)*count)

	for range count {
		buf = append(buf, msg...)
	}

	return buf
}

func mockForwardMode(msg []byte, count int) []byte {
	var buf []byte
	buf = msgpack.AppendArrayHeader(buf, 2)
	buf = msgpack.AppendString(buf, "tag")
	buf = msgpack.AppendArrayHeader(buf, count)

	buf = slices.Grow(buf, len(msg)*count)

	for range count {
		buf = append(buf, msg...)
	}

	return buf
}

func mockPackedForwardMode(msg []byte, count int) []byte {
	payload := make([]byte, 0, len(msg)*count)

	for range count {
		payload = append(payload, msg...)
	}

	var buf []byte
	buf = msgpack.AppendArrayHeader(buf, 2)
	buf = msgpack.AppendString(buf, "tag")
	buf = msgpack.AppendBinary(buf, payload)

	return buf
}

func mockCompressedPackedForwardMode(msg []byte, count int) []byte {
	var w bytes.Buffer
	wr := stdGzip.NewWriter(&w)

	for range count {
		wr.Write(msg)
	}

	wr.Close()

	var buf []byte
	buf = msgpack.AppendArrayHeader(buf, 2)
	buf = msgpack.AppendString(buf, "tag")
	buf = msgpack.AppendBinary(buf, w.Bytes())

	return buf
}

func mockMsg() (msg []byte) {
	msg = msgpack.AppendArrayHeader(msg, 2)
	msg = msgpack.AppendTimestamp(msg, time.Now())
	msg = msgpack.AppendMapHeader(msg, 1)
	msg = msgpack.AppendString(msg, "key")
	msg = msgpack.AppendString(msg, "value")

	return
}

func BenchmarkMessageMode(b *testing.B) {
	payload := mockMessageMode(b.N)
	b.SetBytes(int64(len(payload) / b.N))
	bench(b, payload)
}

func BenchmarkForwardMode(b *testing.B) {
	msg := mockMsg()
	b.SetBytes(int64(len(msg)))
	bench(b, mockForwardMode(msg, b.N))
}

func BenchmarkPackedForwardMode(b *testing.B) {
	msg := mockMsg()
	b.SetBytes(int64(len(msg)))
	bench(b, mockPackedForwardMode(msg, b.N))
}

func BenchmarkCompressedPackedForwardMode(b *testing.B) {
	msg := mockMsg()
	b.SetBytes(int64(len(msg)))
	bench(b, mockCompressedPackedForwardMode(msg, b.N))
}

func bench(b *testing.B, data []byte) {
	var (
		t        TransportPhase
		iterPool msgpack.IterPool
		gzipPool gzip.Pool
	)

	iter := msgpack.NewIterator(bytes.NewReader(data))
	t.Init(&iterPool, &gzipPool, func(chunk string) error { return nil })
	b.ResetTimer()

	// 4) We'll read b.N sub-events, ignoring them but measuring parse overhead
	for i := 0; i < b.N; i++ {
		var e Entry

		if err := t.Next(&iter, &e); err != nil {
			b.Fatalf("Next error at %d of %d: %v", i, b.N, err)
		}

		e.Record.Skip()
		e.Record.Flush()
	}

	elapsed := b.Elapsed().Seconds()
	b.StopTimer()

	b.ReportMetric(float64(b.N)/elapsed, "msgs/sec")
}
