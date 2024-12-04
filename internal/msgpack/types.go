package msgpack

const (
	PosFixint = 0x00
	Fixmap    = 0x80
	Fixarray  = 0x90
	Fixstr    = 0xa0
	Nil       = 0xc0
	False     = 0xc2
	True      = 0xc3
	Bin8      = 0xc4
	Bin16     = 0xc5
	Bin32     = 0xc6
	Ext8      = 0xc7
	Ext16     = 0xc8
	Ext32     = 0xc9
	Float32   = 0xca
	Float64   = 0xcb
	Uint8     = 0xcc
	Uint16    = 0xcd
	Uint32    = 0xce
	Uint64    = 0xcf
	Int8      = 0xd0
	Int16     = 0xd1
	Int32     = 0xd2
	Int64     = 0xd3
	Fixext1   = 0xd4
	Fixext2   = 0xd5
	Fixext4   = 0xd6
	Fixext8   = 0xd7
	Fixext16  = 0xd8
	Str8      = 0xd9
	Str16     = 0xda
	Str32     = 0xdb
	Array16   = 0xdc
	Array32   = 0xdd
	Map16     = 0xde
	Map32     = 0xdf
	NegFixint = 0xe0
)

// Type represents the MessagePack type of the next value.
type Type uint8

const (
	TypeUnknown Type = iota
	TypeNil
	TypeBool
	TypeInt
	TypeUint
	TypeFloat32
	TypeFloat64
	TypeString
	TypeBinary
	TypeArray
	TypeMap
	TypeExt
	TypeTimestamp
)

func (t Type) String() string {
	switch t {
	case TypeNil:
		return "nil"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeUint:
		return "uint"
	case TypeFloat32:
		return "float32"
	case TypeFloat64:
		return "float64"
	case TypeString:
		return "string"
	case TypeBinary:
		return "binary"
	case TypeArray:
		return "array"
	case TypeMap:
		return "map"
	case TypeExt:
		return "exp"
	case TypeTimestamp:
		return "timestamp"
	}

	return "unknown"
}
