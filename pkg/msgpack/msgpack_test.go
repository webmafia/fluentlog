package msgpack

// func Example_msg() {
// 	var b []byte
// 	b = AppendArrayHeader(b, 3)
// 	b = AppendString(b, "foo.bar")
// 	b = AppendTimestamp(b, time.Date(2025, 1, 1, 1, 2, 3, 4, time.UTC), TsFluentd)
// 	b = AppendMapHeader(b, 3)

// 	b = AppendString(b, "a")
// 	b = AppendBool(b, true)

// 	b = AppendString(b, "b")
// 	b = AppendInt(b, -123)

// 	b = AppendString(b, "c")
// 	b = AppendFloat(b, 456.789)

// 	fmt.Println(len(b), ":", b)

// 	r := NewReader(bytes.NewReader(b), buffer.NewBuffer(64), 4096)

// 	fmt.Println(r.Read()) // AppendArrayHeader(b, 3)
// 	fmt.Println(r.Read()) // AppendString(b, "foo.bar")
// 	fmt.Println(r.Read()) // AppendEventTime(b, time.Now())
// 	fmt.Println(r.Read()) // AppendMapHeader(b, 3)

// 	fmt.Println(r.Read())
// 	fmt.Println(r.Read())

// 	fmt.Println(r.Read())
// 	fmt.Println(r.Read())

// 	fmt.Println(r.Read())
// 	fmt.Println(r.Read())

// 	// Output:
// 	//
// 	// 38 : [147 167 102 111 111 46 98 97 114 215 0 103 116 148 11 0 0 0 4 131 161 97 195 161 98 208 133 161 99 203 64 124 140 159 190 118 200 180]
// 	// Array<3> <nil>
// 	// foo.bar <nil>
// 	// 2025-01-01 02:02:03 <nil>
// 	// Map<3> <nil>
// 	// a <nil>
// 	// true <nil>
// 	// b <nil>
// 	// -123 <nil>
// 	// c <nil>
// 	// 456.789000 <nil>
// }
