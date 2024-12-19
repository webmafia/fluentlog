package msgpack

// func ExampleReader() {
// 	var b []byte

// 	b = AppendArray(b, 3)
// 	b = AppendString(b, "foo.bar")
// 	b = AppendTimestamp(b, time.Now())
// 	// b = AppendMap(b, 3)

// 	// b = AppendString(b, "a")
// 	// b = AppendBool(b, true)

// 	// b = AppendString(b, "b")
// 	// b = AppendInt(b, 123)

// 	// b = AppendString(b, "c")
// 	// b = AppendFloat64(b, 456.789)

// 	r := NewReader(bytes.NewReader(b), make([]byte, 4096))

// 	fmt.Println(r.PeekType())
// 	fmt.Println(r.ReadArrayHeader())

// 	fmt.Println(r.PeekType())
// 	fmt.Println(r.ReadString())

// 	fmt.Println(r.PeekType())
// 	fmt.Println(r.ReadTimestamp())

// 	fmt.Println(r.PeekType())
// 	fmt.Println(r.ReadMapHeader())

// 	// Output: TODO
// }

// func TestReader_ReadRaw(t *testing.T) {
// 	// MessagePack-encoded data for an array [1, "hello", [true, false]]
// 	data := []byte{
// 		0x93,                               // Array of length 3
// 		0x01,                               // Integer 1
// 		0xa5, 0x68, 0x65, 0x6c, 0x6c, 0x6f, // String "hello"
// 		0x92, // Array of length 2
// 		0xc3, // True
// 		0xc2, // False
// 	}

// 	buffer := make([]byte, 1024)
// 	reader := NewReader(bytes.NewReader(data), buffer)

// 	rawBytes, err := reader.ReadRaw()
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}

// 	// Verify that rawBytes matches the original data
// 	if !bytes.Equal(rawBytes, data) {
// 		t.Fatalf("expected %x, got %x", data, rawBytes)
// 	}
// }

// func TestReader_ReadTimestamp(t *testing.T) {
// 	tests := []struct {
// 		name           string
// 		input          []byte
// 		expectedSec    int64
// 		expectedNsec   int64
// 		expectedErr    bool
// 		expectedFormat string // To help identify the type (e.g., fixext8 or ext8)
// 	}{
// 		{
// 			name:           "fixext8 timestamp",
// 			input:          createFixExt8Timestamp(1609459200, 123456789), // 2021-01-01T00:00:00.123456789Z
// 			expectedSec:    1609459200,
// 			expectedNsec:   123456789,
// 			expectedErr:    false,
// 			expectedFormat: "fixext8",
// 		},
// 		{
// 			name:           "ext8 timestamp",
// 			input:          createExt8Timestamp(1609459200, 987654321), // 2021-01-01T00:00:00.987654321Z
// 			expectedSec:    1609459200,
// 			expectedNsec:   987654321,
// 			expectedErr:    false,
// 			expectedFormat: "ext8",
// 		},
// 		{
// 			name:           "integer timestamp (seconds only)",
// 			input:          createIntegerTimestamp(1609459200), // 2021-01-01T00:00:00Z
// 			expectedSec:    1609459200,
// 			expectedNsec:   0,
// 			expectedErr:    false,
// 			expectedFormat: "integer",
// 		},
// 		{
// 			name:           "invalid type",
// 			input:          []byte{0xd4, 0x00}, // Invalid fixext1
// 			expectedSec:    0,
// 			expectedNsec:   0,
// 			expectedErr:    true,
// 			expectedFormat: "invalid",
// 		},
// 	}

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			r := NewReader(bytes.NewReader(test.input), make([]byte, 0, 64))

// 			timestamp, err := r.ReadTimestamp()
// 			if test.expectedErr {
// 				if err == nil {
// 					t.Errorf("expected an error but got none")
// 				}
// 				return
// 			}

// 			if err != nil {
// 				t.Errorf("unexpected error: %v", err)
// 				return
// 			}

// 			if timestamp.Unix() != test.expectedSec || timestamp.Nanosecond() != int(test.expectedNsec) {
// 				t.Errorf("expected timestamp %d.%09d but got %d.%09d",
// 					test.expectedSec, test.expectedNsec, timestamp.Unix(), timestamp.Nanosecond())
// 			}
// 		})
// 	}
// }

// // Helper function to create a fixext8 timestamp
// func createFixExt8Timestamp(sec int64, nsec int64) []byte {
// 	buf := make([]byte, 10)
// 	buf[0] = 0xd7 // fixext8
// 	buf[1] = 0x00 // type: EventTime
// 	binary.BigEndian.PutUint32(buf[2:6], uint32(sec))
// 	binary.BigEndian.PutUint32(buf[6:10], uint32(nsec))
// 	return buf
// }

// // Helper function to create an ext8 timestamp
// func createExt8Timestamp(sec int64, nsec int64) []byte {
// 	buf := make([]byte, 11)
// 	buf[0] = 0xc7 // ext8
// 	buf[1] = 0x08 // length: 8
// 	buf[2] = 0x00 // type: EventTime
// 	binary.BigEndian.PutUint32(buf[3:7], uint32(sec))
// 	binary.BigEndian.PutUint32(buf[7:11], uint32(nsec))
// 	return buf
// }

// // Helper function to create an integer timestamp (seconds since epoch)
// func createIntegerTimestamp(sec int64) []byte {
// 	buf := make([]byte, 9)
// 	buf[0] = 0xd3 // int64
// 	binary.BigEndian.PutUint64(buf[1:], uint64(sec))
// 	return buf
// }
