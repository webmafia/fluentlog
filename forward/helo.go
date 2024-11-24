package forward

// import (
// 	"fmt"

// 	"github.com/webmafia/fluentlog/internal/msgpack"
// )

// type Helo struct {
// 	Nonce     [32]byte
// 	KeepAlive bool
// }

// func (h Helo) Encode(w *msgpack.Writer) {
// 	w.WriteArrayHeader(2)
// 	w.WriteString("HELO")
// 	w.WriteMapHeader(2)

// 	w.WriteString("nonce")
// 	w.WriteBinary(h.Nonce[:])

// 	w.WriteString("keepalive")
// 	w.WriteBool(h.KeepAlive)
// }

// func (h Helo) Decode(r *msgpack.Reader) (err error) {
// 	n, err := r.ReadArrayHeader()

// 	if err != nil {
// 		return
// 	}

// 	if n != 2 {
// 		return fmt.Errorf("expected array with 2 items")
// 	}

// 	w.WriteArrayHeader(2)
// 	w.WriteString("HELO")
// 	w.WriteMapHeader(2)

// 	w.WriteString("nonce")
// 	w.WriteBinary(h.Nonce[:])

// 	w.WriteString("keepalive")
// 	w.WriteBool(h.KeepAlive)
// }
