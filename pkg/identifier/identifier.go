package identifier

import "time"

var gen AtomicGenerator

func init() {
	gen = NewAtomicGenerator()
}

// Atomically generates the next ID based on current time. Thread-safe.
func Generate() ID {
	return gen.ID()
}

// Atomically generates the next ID based on provided timestamp. Thread-safe.
func IDFromTime(ts time.Time) ID {
	return gen.IDFromTime(ts)
}
