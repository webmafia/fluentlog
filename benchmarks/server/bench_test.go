package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/webmafia/fast/ringbuf"
	"github.com/webmafia/fluentlog/forward"
	"github.com/webmafia/fluentlog/forward/transport"
)

func Benchmark_1MB(b *testing.B) {
	payload, numMessages := createPayload(1 * 1024 * 1024)
	payloadSize := int64(len(payload))
	msgSize := len(payload) / numMessages

	b.Run("RingbufReader", func(b *testing.B) {
		b.SetBytes(payloadSize)
		sr := bytes.NewReader(payload)
		r := ringbuf.NewReader(sr)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			sr.Reset(payload)
			r.Reset(r)

			for {
				_, err := r.ReadBytes(msgSize)

				if err == io.EOF {
					break
				}
			}
		}

		b.ReportMetric(float64(b.Elapsed())/float64(b.N*numMessages), "ns/msg")
	})

	// b.Run("MsgpackIter", func(b *testing.B) {
	// 	b.SetBytes(payloadSize)
	// 	iter := msgpack.NewIterator(nil)
	// 	b.ResetTimer()

	// 	for i := 0; i < b.N; i++ {
	// 		iter.ResetBytes(payload)
	// 		count := 0

	// 		for iter.Next() {
	// 			iter.Skip()
	// 			count++
	// 		}

	// 		if err := iter.Error(); err != nil && err != io.EOF {
	// 			b.Fatalf("Iterator error: %v", err)
	// 		}

	// 		if count != numMessages {
	// 			b.Fatalf("Expected %d messages, got %d", numMessages, count)
	// 		}
	// 	}

	// 	b.ReportMetric(float64(b.Elapsed())/float64(b.N*numMessages), "ns/msg")
	// })

	b.Run("ServerConnEntries", func(b *testing.B) {
		b.SetBytes(payloadSize)

		// Create a dummy connection.
		err := tcpDummyConn(payload, func(conn *dummyConn) error {
			s := forward.NewServer(forward.ServerOptions{})
			ss := newServerSession(s, conn)

			// Report allocations.
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var e transport.Entry
				if err := ss.Next(&e); err != nil {
					b.Fatalf("Next error at i=%d: %v", i, err)
				}

				e.Record.Skip()
			}

			return nil
		})

		if err != nil {
			b.Fatal(err)
		}

		b.ReportMetric(float64(b.Elapsed())/float64(b.N*numMessages), "ns/msg")
	})
}
