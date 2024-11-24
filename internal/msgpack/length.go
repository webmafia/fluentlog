package msgpack

import (
	"encoding/binary"
	"fmt"
)

// GetMsgpackValueLength calculates the total number of bytes occupied by the MessagePack-encoded value
// starting at position 0 in the given byte slice, including the header.
func GetMsgpackValueLength(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, fmt.Errorf("buffer is empty")
	}
	b := buf[0]
	switch {
	// Positive FixInt
	case b <= 0x7f:
		return 1, nil

	// FixMap
	case b >= 0x80 && b <= 0x8f:
		numPairs := int(b & 0x0f)
		offset := 1
		for i := 0; i < numPairs; i++ {
			// Key
			keyLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += keyLen

			// Value
			valueLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += valueLen
		}
		return offset, nil

	// FixArray
	case b >= 0x90 && b <= 0x9f:
		length := int(b & 0x0f)
		offset := 1
		for i := 0; i < length; i++ {
			elemLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += elemLen
		}
		return offset, nil

	// FixStr
	case b >= 0xa0 && b <= 0xbf:
		strLen := int(b & 0x1f)
		totalLen := 1 + strLen
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for FixStr")
		}
		return totalLen, nil

	// Nil
	case b == Nil:
		return 1, nil

	// Bool
	case b == False || b == True:
		return 1, nil

	// Binary
	case b == Bin8:
		if len(buf) < 2 {
			return 0, fmt.Errorf("buffer too short for Bin8")
		}
		length := int(buf[1])
		totalLen := 2 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Bin8 data")
		}
		return totalLen, nil

	case b == Bin16:
		if len(buf) < 3 {
			return 0, fmt.Errorf("buffer too short for Bin16")
		}
		length := int(binary.BigEndian.Uint16(buf[1:3]))
		totalLen := 3 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Bin16 data")
		}
		return totalLen, nil

	case b == Bin32:
		if len(buf) < 5 {
			return 0, fmt.Errorf("buffer too short for Bin32")
		}
		length := int(binary.BigEndian.Uint32(buf[1:5]))
		totalLen := 5 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Bin32 data")
		}
		return totalLen, nil

	// Ext
	case b == Fixext1:
		totalLen := 1 + 1 + 1 // header + type + data
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Fixext1")
		}
		return totalLen, nil

	case b == Fixext2:
		totalLen := 1 + 1 + 2
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Fixext2")
		}
		return totalLen, nil

	case b == Fixext4:
		totalLen := 1 + 1 + 4
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Fixext4")
		}
		return totalLen, nil

	case b == Fixext8:
		totalLen := 1 + 1 + 8
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Fixext8")
		}
		return totalLen, nil

	case b == Fixext16:
		totalLen := 1 + 1 + 16
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Fixext16")
		}
		return totalLen, nil

	case b == Ext8:
		if len(buf) < 3 {
			return 0, fmt.Errorf("buffer too short for Ext8")
		}
		length := int(buf[1])
		totalLen := 2 + 1 + length // header + type + data
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Ext8 data")
		}
		return totalLen, nil

	case b == Ext16:
		if len(buf) < 4 {
			return 0, fmt.Errorf("buffer too short for Ext16")
		}
		length := int(binary.BigEndian.Uint16(buf[1:3]))
		totalLen := 3 + 1 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Ext16 data")
		}
		return totalLen, nil

	case b == Ext32:
		if len(buf) < 6 {
			return 0, fmt.Errorf("buffer too short for Ext32")
		}
		length := int(binary.BigEndian.Uint32(buf[1:5]))
		totalLen := 5 + 1 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Ext32 data")
		}
		return totalLen, nil

	// Float
	case b == Float32:
		totalLen := 1 + 4
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Float32")
		}
		return totalLen, nil

	case b == Float64:
		totalLen := 1 + 8
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Float64")
		}
		return totalLen, nil

	// Uint
	case b == Uint8:
		totalLen := 1 + 1
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Uint8")
		}
		return totalLen, nil

	case b == Uint16:
		totalLen := 1 + 2
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Uint16")
		}
		return totalLen, nil

	case b == Uint32:
		totalLen := 1 + 4
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Uint32")
		}
		return totalLen, nil

	case b == Uint64:
		totalLen := 1 + 8
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Uint64")
		}
		return totalLen, nil

	// Int
	case b == Int8:
		totalLen := 1 + 1
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Int8")
		}
		return totalLen, nil

	case b == Int16:
		totalLen := 1 + 2
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Int16")
		}
		return totalLen, nil

	case b == Int32:
		totalLen := 1 + 4
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Int32")
		}
		return totalLen, nil

	case b == Int64:
		totalLen := 1 + 8
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Int64")
		}
		return totalLen, nil

	// Str
	case b == Str8:
		if len(buf) < 2 {
			return 0, fmt.Errorf("buffer too short for Str8")
		}
		length := int(buf[1])
		totalLen := 2 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Str8 data")
		}
		return totalLen, nil

	case b == Str16:
		if len(buf) < 3 {
			return 0, fmt.Errorf("buffer too short for Str16")
		}
		length := int(binary.BigEndian.Uint16(buf[1:3]))
		totalLen := 3 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Str16 data")
		}
		return totalLen, nil

	case b == Str32:
		if len(buf) < 5 {
			return 0, fmt.Errorf("buffer too short for Str32")
		}
		length := int(binary.BigEndian.Uint32(buf[1:5]))
		totalLen := 5 + length
		if len(buf) < totalLen {
			return 0, fmt.Errorf("buffer too short for Str32 data")
		}
		return totalLen, nil

	// Array
	case b == Array16:
		if len(buf) < 3 {
			return 0, fmt.Errorf("buffer too short for Array16")
		}
		length := int(binary.BigEndian.Uint16(buf[1:3]))
		offset := 3
		for i := 0; i < length; i++ {
			elemLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += elemLen
		}
		return offset, nil

	case b == Array32:
		if len(buf) < 5 {
			return 0, fmt.Errorf("buffer too short for Array32")
		}
		length := int(binary.BigEndian.Uint32(buf[1:5]))
		offset := 5
		for i := 0; i < length; i++ {
			elemLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += elemLen
		}
		return offset, nil

	// Map
	case b == Map16:
		if len(buf) < 3 {
			return 0, fmt.Errorf("buffer too short for Map16")
		}
		numPairs := int(binary.BigEndian.Uint16(buf[1:3]))
		offset := 3
		for i := 0; i < numPairs; i++ {
			// Key
			keyLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += keyLen

			// Value
			valueLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += valueLen
		}
		return offset, nil

	case b == Map32:
		if len(buf) < 5 {
			return 0, fmt.Errorf("buffer too short for Map32")
		}
		numPairs := int(binary.BigEndian.Uint32(buf[1:5]))
		offset := 5
		for i := 0; i < numPairs; i++ {
			// Key
			keyLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += keyLen

			// Value
			valueLen, err := GetMsgpackValueLength(buf[offset:])
			if err != nil {
				return 0, err
			}
			offset += valueLen
		}
		return offset, nil

	// Negative FixInt
	case b >= 0xe0 && b <= 0xff:
		return 1, nil

	default:
		return 0, fmt.Errorf("unknown type byte: 0x%02x", b)
	}
}
