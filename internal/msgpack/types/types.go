package types

import "fmt"

type Type uint8

// Value constants for MessagePack
const (
	Nil Type = iota
	Bool
	Int
	Uint
	Float
	Str
	Bin
	Array
	Map
	Ext
	Reserved // Reserved type for 0xc1
)

var typeStrings = [...]string{
	"nil",
	"bool",
	"int",
	"uint",
	"float",
	"str",
	"bin",
	"array",
	"map",
	"ext",
	"reserved",
}

func (t Type) String() string {
	if int(t) >= len(typeStrings) {
		return fmt.Sprintf("(invalid type %d)", t)
	}

	return typeStrings[t]
}

// Precomputed arrays for fast lookup
var typeLookup [256]Type
var lengthLookup [256]byte
var isLengthValue [256]bool

func init() {
	// Initialize typeLookup based on MessagePack specification
	for i := 0x00; i <= 0x7f; i++ { // Positive FixInt
		typeLookup[i] = Uint
	}
	for i := 0xe0; i <= 0xff; i++ { // Negative FixInt
		typeLookup[i] = Int
	}
	for i := 0xa0; i <= 0xbf; i++ { // FixStr
		typeLookup[i] = Str
	}
	for i := 0x80; i <= 0x8f; i++ { // FixMap
		typeLookup[i] = Map
	}
	for i := 0x90; i <= 0x9f; i++ { // FixArray
		typeLookup[i] = Array
	}

	// Fixed types
	typeLookup[0xc0] = Nil      // nil
	typeLookup[0xc1] = Reserved // Reserved
	typeLookup[0xc2] = Bool     // false
	typeLookup[0xc3] = Bool     // true
	typeLookup[0xc4] = Bin      // bin8
	typeLookup[0xc5] = Bin      // bin16
	typeLookup[0xc6] = Bin      // bin32
	typeLookup[0xc7] = Ext      // ext8
	typeLookup[0xc8] = Ext      // ext16
	typeLookup[0xc9] = Ext      // ext32
	typeLookup[0xca] = Float    // float32
	typeLookup[0xcb] = Float    // float64
	typeLookup[0xcc] = Uint     // uint8
	typeLookup[0xcd] = Uint     // uint16
	typeLookup[0xce] = Uint     // uint32
	typeLookup[0xcf] = Uint     // uint64
	typeLookup[0xd0] = Int      // int8
	typeLookup[0xd1] = Int      // int16
	typeLookup[0xd2] = Int      // int32
	typeLookup[0xd3] = Int      // int64
	typeLookup[0xd4] = Ext      // fixext1
	typeLookup[0xd5] = Ext      // fixext2
	typeLookup[0xd6] = Ext      // fixext4
	typeLookup[0xd7] = Ext      // fixext8
	typeLookup[0xd8] = Ext      // fixext16
	typeLookup[0xd9] = Str      // str8
	typeLookup[0xda] = Str      // str16
	typeLookup[0xdb] = Str      // str32
	typeLookup[0xdc] = Array    // array16
	typeLookup[0xdd] = Array    // array32
	typeLookup[0xde] = Map      // map16
	typeLookup[0xdf] = Map      // map32

	// Initialize lengthLookup and isLengthValue
	for i := 0x00; i <= 0x7f; i++ { // Positive FixInt
		lengthLookup[i] = 0
		isLengthValue[i] = true
	}
	for i := 0xe0; i <= 0xff; i++ { // Negative FixInt
		lengthLookup[i] = 0
		isLengthValue[i] = true
	}
	for i := 0xa0; i <= 0xbf; i++ { // FixStr
		lengthLookup[i] = byte(i - 0xa0) // Length embedded in the type byte
		isLengthValue[i] = true
	}
	for i := 0x80; i <= 0x8f; i++ { // FixMap
		lengthLookup[i] = byte(i - 0x80) // Number of key-value pairs
		isLengthValue[i] = true
	}
	for i := 0x90; i <= 0x9f; i++ { // FixArray
		lengthLookup[i] = byte(i - 0x90) // Number of elements
		isLengthValue[i] = true
	}

	// Fixed lengths
	lengthLookup[0xc0] = 0 // nil
	isLengthValue[0xc0] = true
	lengthLookup[0xc1] = 0 // Reserved
	isLengthValue[0xc1] = true
	lengthLookup[0xc2] = 0 // false
	isLengthValue[0xc2] = true
	lengthLookup[0xc3] = 0 // true
	isLengthValue[0xc3] = true
	lengthLookup[0xc4] = 1 // bin8
	isLengthValue[0xc4] = false
	lengthLookup[0xc5] = 2 // bin16
	isLengthValue[0xc5] = false
	lengthLookup[0xc6] = 4 // bin32
	isLengthValue[0xc6] = false
	lengthLookup[0xc7] = 2 // ext8 (1 byte for type + 1 byte for length)
	isLengthValue[0xc7] = false
	lengthLookup[0xc8] = 3 // ext16 (1 byte for type + 2 bytes for length)
	isLengthValue[0xc8] = false
	lengthLookup[0xc9] = 5 // ext32 (1 byte for type + 4 bytes for length)
	isLengthValue[0xc9] = false
	lengthLookup[0xca] = 4 // float32
	isLengthValue[0xca] = true
	lengthLookup[0xcb] = 8 // float64
	isLengthValue[0xcb] = true
	lengthLookup[0xcc] = 1 // uint8
	isLengthValue[0xcc] = true
	lengthLookup[0xcd] = 2 // uint16
	isLengthValue[0xcd] = true
	lengthLookup[0xce] = 4 // uint32
	isLengthValue[0xce] = true
	lengthLookup[0xcf] = 8 // uint64
	isLengthValue[0xcf] = true
	lengthLookup[0xd0] = 1 // int8
	isLengthValue[0xd0] = true
	lengthLookup[0xd1] = 2 // int16
	isLengthValue[0xd1] = true
	lengthLookup[0xd2] = 4 // int32
	isLengthValue[0xd2] = true
	lengthLookup[0xd3] = 8 // int64
	isLengthValue[0xd3] = true
	lengthLookup[0xd4] = 2 // fixext1 (1 byte for type + 1 byte for data)
	isLengthValue[0xd4] = false
	lengthLookup[0xd5] = 3 // fixext2 (1 byte for type + 2 bytes for data)
	isLengthValue[0xd5] = false
	lengthLookup[0xd6] = 5 // fixext4 (1 byte for type + 4 bytes for data)
	isLengthValue[0xd6] = false
	lengthLookup[0xd7] = 9 // fixext8 (1 byte for type + 8 bytes for data)
	isLengthValue[0xd7] = false
	lengthLookup[0xd8] = 17 // fixext16 (1 byte for type + 16 bytes for data)
	isLengthValue[0xd8] = false
	lengthLookup[0xd9] = 1 // str8
	isLengthValue[0xd9] = false
	lengthLookup[0xda] = 2 // str16
	isLengthValue[0xda] = false
	lengthLookup[0xdb] = 4 // str32
	isLengthValue[0xdb] = false
	lengthLookup[0xdc] = 2 // array16
	isLengthValue[0xdc] = false
	lengthLookup[0xdd] = 4 // array32
	isLengthValue[0xdd] = false
	lengthLookup[0xde] = 2 // map16
	isLengthValue[0xde] = false
	lengthLookup[0xdf] = 4 // map32
	isLengthValue[0xdf] = false
}

// Get returns the type of the MessagePack value, its length, and a boolean indicating the nature of the length.
// The length represents either the actual value's length (if the boolean is true),
// or the number of bytes used to encode the value's length (if the boolean is false).
func Get(c byte) (typ Type, length int, isValueLength bool) {
	return typeLookup[c], int(lengthLookup[c]), isLengthValue[c]
}
