package msgpack

import (
	"bytes"
	"io"
	"testing"
)

const msgpackDataSize = 1 * 1024 * 1024 // 1 MB

// generateMsgpackData returns a 1 MB payload consisting entirely of the byte 0x01,
// which is a valid positive fixint in MessagePack.
func generateMsgpackData() []byte {
	return bytes.Repeat([]byte{0x01}, msgpackDataSize)
}

// BenchmarkMsgpackIterator benchmarks the low-level MessagePack iterator over a 1 MB payload.
// It creates a new iterator for each iteration, loops over all tokens, and verifies that
// the total number of tokens equals the payload size.
func BenchmarkMsgpackIterator_1Byte(b *testing.B) {
	data := generateMsgpackData()
	iter := NewIterator(nil)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter.ResetBytes(data)
		tokens := 0
		for iter.Next() {
			tokens++
		}
		// If an error occurred that is not EOF, report it.
		if err := iter.Error(); err != nil && err != io.EOF {
			b.Fatalf("Iterator error: %v", err)
		}
		// Since each token is one byte, we expect tokens == len(data)
		if tokens != len(data) {
			b.Fatalf("Expected %d tokens, got %d", len(data), tokens)
		}
	}
}

// generateMsgpackData16 returns a payload of roughly 1 MB consisting of 16-byte tokens.
// Each token is encoded as a fixstr with total length 16: a 1-byte header (0xaf) followed by a 15-byte payload.
func generateMsgpackData16() []byte {
	// fixstr header for a string of length 15 is 0xa0 | 15 = 0xaf.
	tokenHeader := byte(0xaf)
	// Example 15-byte payload.
	payload := []byte("abcdefghijklmno")             // 15 bytes
	token := append([]byte{tokenHeader}, payload...) // total 16 bytes

	// Determine the number of tokens needed to produce ~1MB of data.
	numTokens := (1 * 1024 * 1024) / len(token)
	buf := make([]byte, 0, numTokens*len(token))
	for i := 0; i < numTokens; i++ {
		buf = append(buf, token...)
	}
	return buf
}

// BenchmarkMsgpackIterator_16Byte benchmarks the MessagePack iterator over a ~1MB payload
// composed of 16-byte tokens.
func BenchmarkMsgpackIterator_16Byte(b *testing.B) {
	data := generateMsgpackData16()
	// Each token is 16 bytes (1 header + 15 payload), so expected token count is:
	expectedTokens := len(data) / 16

	iter := NewIterator(nil)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter.ResetBytes(data)
		tokens := 0
		for iter.Next() {
			_ = iter.Str()
			tokens++
		}
		if err := iter.Error(); err != nil && err != io.EOF {
			b.Fatalf("Iterator error: %v", err)
		}
		if tokens != expectedTokens {
			b.Fatalf("Expected %d tokens, got %d", expectedTokens, tokens)
		}
	}
}
