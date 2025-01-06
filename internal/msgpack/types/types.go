package types

import "fmt"

// Type represents the MessagePack value type.
type Type uint8

// Value constants for MessagePack.
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
}

func (t Type) String() string {
	if int(t) >= len(typeStrings) {
		return fmt.Sprintf("(invalid type %d)", t)
	}

	return typeStrings[t]
}

// Precomputed arrays for fast lookup.
var typeLookup [256]Type
var lengthLookup [256]byte
var isLengthValue [256]bool

func init() {
	// Initialize typeLookup based on MessagePack specification.

	// Positive FixInt (0x00 to 0x7f)
	for i := 0x00; i <= 0x7f; i++ {
		typeLookup[i] = Uint
		lengthLookup[i] = 0
		isLengthValue[i] = true
	}

	// Negative FixInt (0xe0 to 0xff)
	for i := 0xe0; i <= 0xff; i++ {
		typeLookup[i] = Int
		lengthLookup[i] = 0
		isLengthValue[i] = true
	}

	// FixStr (0xa0 to 0xbf)
	for i := 0xa0; i <= 0xbf; i++ {
		typeLookup[i] = Str
		lengthLookup[i] = byte(i - 0xa0)
		isLengthValue[i] = true
	}

	// FixMap (0x80 to 0x8f)
	for i := 0x80; i <= 0x8f; i++ {
		typeLookup[i] = Map
		lengthLookup[i] = byte(i - 0x80)
		isLengthValue[i] = true
	}

	// FixArray (0x90 to 0x9f)
	for i := 0x90; i <= 0x9f; i++ {
		typeLookup[i] = Array
		lengthLookup[i] = byte(i - 0x90)
		isLengthValue[i] = true
	}

	// Fixed types
	typeLookup[0xc0] = Nil // nil
	lengthLookup[0xc0] = 0
	isLengthValue[0xc0] = true

	typeLookup[0xc2] = Bool // false
	lengthLookup[0xc2] = 0
	isLengthValue[0xc2] = true

	typeLookup[0xc3] = Bool // true
	lengthLookup[0xc3] = 0
	isLengthValue[0xc3] = true

	// Binary types
	typeLookup[0xc4] = Bin // bin8
	lengthLookup[0xc4] = 1
	isLengthValue[0xc4] = false

	typeLookup[0xc5] = Bin // bin16
	lengthLookup[0xc5] = 2
	isLengthValue[0xc5] = false

	typeLookup[0xc6] = Bin // bin32
	lengthLookup[0xc6] = 4
	isLengthValue[0xc6] = false

	// Extension types
	typeLookup[0xc7] = Ext // ext8
	lengthLookup[0xc7] = 1
	isLengthValue[0xc7] = false

	typeLookup[0xc8] = Ext // ext16
	lengthLookup[0xc8] = 2
	isLengthValue[0xc8] = false

	typeLookup[0xc9] = Ext // ext32
	lengthLookup[0xc9] = 4
	isLengthValue[0xc9] = false

	// Float types
	typeLookup[0xca] = Float // float32
	lengthLookup[0xca] = 4
	isLengthValue[0xca] = true

	typeLookup[0xcb] = Float // float64
	lengthLookup[0xcb] = 8
	isLengthValue[0xcb] = true

	// Unsigned integers
	typeLookup[0xcc] = Uint // uint8
	lengthLookup[0xcc] = 1
	isLengthValue[0xcc] = true

	typeLookup[0xcd] = Uint // uint16
	lengthLookup[0xcd] = 2
	isLengthValue[0xcd] = true

	typeLookup[0xce] = Uint // uint32
	lengthLookup[0xce] = 4
	isLengthValue[0xce] = true

	typeLookup[0xcf] = Uint // uint64
	lengthLookup[0xcf] = 8
	isLengthValue[0xcf] = true

	// Signed integers
	typeLookup[0xd0] = Int // int8
	lengthLookup[0xd0] = 1
	isLengthValue[0xd0] = true

	typeLookup[0xd1] = Int // int16
	lengthLookup[0xd1] = 2
	isLengthValue[0xd1] = true

	typeLookup[0xd2] = Int // int32
	lengthLookup[0xd2] = 4
	isLengthValue[0xd2] = true

	typeLookup[0xd3] = Int // int64
	lengthLookup[0xd3] = 8
	isLengthValue[0xd3] = true

	// String types
	typeLookup[0xd9] = Str // str8
	lengthLookup[0xd9] = 1
	isLengthValue[0xd9] = false

	typeLookup[0xda] = Str // str16
	lengthLookup[0xda] = 2
	isLengthValue[0xda] = false

	typeLookup[0xdb] = Str // str32
	lengthLookup[0xdb] = 4
	isLengthValue[0xdb] = false

	// Array types
	typeLookup[0xdc] = Array // array16
	lengthLookup[0xdc] = 2
	isLengthValue[0xdc] = false

	typeLookup[0xdd] = Array // array32
	lengthLookup[0xdd] = 4
	isLengthValue[0xdd] = false

	// Map types
	typeLookup[0xde] = Map // map16
	lengthLookup[0xde] = 2
	isLengthValue[0xde] = false

	typeLookup[0xdf] = Map // map32
	lengthLookup[0xdf] = 4
	isLengthValue[0xdf] = false

	// Fixext types
	typeLookup[0xd4] = Ext // fixext 1
	lengthLookup[0xd4] = 2
	isLengthValue[0xd4] = true

	typeLookup[0xd5] = Ext // fixext 2
	lengthLookup[0xd5] = 3
	isLengthValue[0xd5] = true

	typeLookup[0xd6] = Ext // fixext 4
	lengthLookup[0xd6] = 5
	isLengthValue[0xd6] = true

	typeLookup[0xd7] = Ext // fixext 8
	lengthLookup[0xd7] = 9
	isLengthValue[0xd7] = true

	typeLookup[0xd8] = Ext // fixext 16
	lengthLookup[0xd8] = 17
	isLengthValue[0xd8] = true
}

// Get returns the type of the MessagePack value, its length, and a boolean indicating the nature of the length.
// The length represents either the actual value's length (if the boolean is true),
// or the number of bytes used to encode the value's length (if the boolean is false).
func Get(c byte) (typ Type, length int, isValueLength bool) {
	return typeLookup[c], int(lengthLookup[c]), isLengthValue[c]
}
