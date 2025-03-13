package identifier

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/webmafia/fast"
	"github.com/webmafia/fluentlog/internal"
)

const (
	multiplier    uint64 = 0x9e3779b97f4a7c15
	invMultiplier uint64 = 0xf1de83e19937733d
)

var (
	_ internal.TextAppender   = ID(0)
	_ internal.BinaryAppender = ID(0)
	_ json.Marshaler          = ID(0)
	_ json.Unmarshaler        = (*ID)(nil)
)

func IDFromString(str string) (id ID, err error) {
	var buf [8]byte
	n, err := hex.Decode(buf[:], fast.StringToBytes(str))

	if err != nil {
		return
	}

	if n != 8 {
		return 0, errors.New("invalid ID")
	}

	scrambled := binary.BigEndian.Uint64(buf[:])

	// Multiply by the precomputed multiplicative inverse to recover the original value.
	// The multiplication is performed modulo 2^64.
	original := scrambled * invMultiplier

	return ID(original), nil
}

func (id ID) String() string {
	b, _ := id.AppendText(make([]byte, 0, 16))
	return fast.BytesToString(b)
}

// AppendBinary implements internal.TextAppender.
func (id ID) AppendText(b []byte) ([]byte, error) {
	var buf [8]byte
	scrambled := uint64(id) * multiplier
	binary.BigEndian.PutUint64(buf[:], scrambled)
	b = hex.AppendEncode(b, buf[:])
	return b, nil
}

// AppendBinary implements internal.BinaryAppender.
func (id ID) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint64(b, uint64(id)), nil
}

// MarshalJSON implements json.Marshaler.
func (id ID) MarshalJSON() (b []byte, err error) {
	b = make([]byte, 0, 18)
	b = append(b, '"')
	b, err = id.AppendText(b)
	b = append(b, '"')

	return
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *ID) UnmarshalJSON(b []byte) (err error) {
	if len(b) != 18 || b[0] != '"' || b[17] != '"' {
		return errors.New("invalid ID")
	}

	*id, err = IDFromString(fast.BytesToString(b[1:17]))
	return
}
