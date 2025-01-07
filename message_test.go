package fluentlog

// import (
// 	"log"
// 	"testing"
// 	"time"
// )

// func Example() {
// 	msg := NewMessage("foo.bar", time.Now())
// 	msg.AddField("foo", "bar")
// 	msg.AddField("baz", 123)
// 	msg.AddField("baz", "test")

// 	type waza struct {
// 		yaaa string
// 	}

// 	msg.AddField("waza", waza{yaaa: "abc"})

// 	log.Printf("%x", msg.buf)

// 	tag, ts, record := msg.Data()

// 	log.Println("tag:", tag.Str())
// 	log.Println("ts:", ts.Time())
// 	log.Println("record:")

// 	for k, v := range record.Map() {
// 		log.Println(k.Str(), ":", v)
// 	}

// 	// for k, v := range msg.Fields() {
// 	// 	fmt.Println(k, v)
// 	// }

// 	// for i := 0; i < 32; i++ {
// 	// 	fmt.Println(msg.NumFields(), len(msg.buf))
// 	// 	msg.incNumFields()
// 	// }

// 	// Output: TODO
// }

// func BenchmarkMessage(b *testing.B) {
// 	tag := "foo.bar"
// 	now := time.Now()
// 	m := NewMessage(tag, now)

// 	b.ResetTimer()

// 	for range b.N {
// 		m.AddField("foo", "bar")
// 		m.Reset(tag, now)
// 	}
// }

// func BenchmarkMessageUnknown(b *testing.B) {
// 	tag := "foo.bar"
// 	now := time.Now()
// 	m := NewMessage(tag, now)

// 	type waza struct {
// 		yaaa string
// 	}

// 	b.ResetTimer()

// 	for range b.N {
// 		m.AddField("waza", waza{yaaa: "abc"})
// 		m.Reset(tag, now)
// 	}
// }
