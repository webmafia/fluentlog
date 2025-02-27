package main

import (
	"io"
	"testing"

	"github.com/webmafia/fluentlog/pkg/msgpack"
)

func TestWrap(t *testing.T) {
	payload, numMessages := createPayload(100 * 1024 * 1024)
	iter := msgpack.NewIterator(nil)

	iter.ResetBytes(payload)
	count := 0

	for {
		if err := validateMessage(&iter); err != nil {
			if err == io.EOF {
				break
			}

			t.Fatal(err)
		}

		count++
	}

	if err := iter.Error(); err != nil && err != io.EOF {
		t.Fatalf("Iterator error: %v", err)
	}

	if count != numMessages {
		t.Fatalf("Expected %d messages, got %d", numMessages, count)
	}

}
