package msgpack

import (
	"math/bits"
	"sync"
	"sync/atomic"

	"github.com/webmafia/fast/bufio"
	"github.com/webmafia/fast/types"
)

const (
	minBitSize = 6 // 2**6=64 is a CPU cache line size
	steps      = 16

	minSize = 1 << minBitSize
	maxSize = 1 << (minBitSize + steps - 1)

	calibrateCallsThreshold = 1024
	maxPercentile           = 0.95
)

// Pool represents byte buffer pool.
//
// Distinct pools may be used for distinct types of byte buffers.
// Properly determined byte buffer types with their own pools may help reducing
// memory waste.
type IterPool struct {
	calls       [steps]uint32
	calibrating uint32

	defaultSize uint32
	maxSize     uint32

	pool       sync.Pool
	BufMaxSize int
}

// Get returns new byte buffer with zero length.
//
// The byte buffer may be returned to the pool via Put after the use
// in order to minimize GC overhead.
func (p *IterPool) Get() *Iterator {
	v := p.pool.Get()
	if v != nil {
		return v.(*Iterator)
	}

	size := int(atomic.LoadUint32(&p.defaultSize))

	if size == 0 {
		size = min(p.BufMaxSize, 4096)
	}

	iter := &Iterator{r: bufio.NewReader(nil, p.BufMaxSize)}
	iter.r.ResetSize(size)

	return iter
}

// Put releases byte buffer obtained via Get to the pool.
//
// The buffer mustn't be accessed after returning to the pool.
func (p *IterPool) Put(iter *Iterator) {
	idx := index(iter.r.MaxUsed())

	if atomic.AddUint32(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	maxSize := int(atomic.LoadUint32(&p.maxSize))
	if maxSize == 0 || iter.r.Size() <= maxSize {
		iter.Reset(nil, p.BufMaxSize)
		p.pool.Put(iter)
	}
}

func (p *IterPool) calibrate() {
	if !atomic.CompareAndSwapUint32(&p.calibrating, 0, 1) {
		return
	}

	var calls, sizes [steps]uint32
	var callsSum uint32
	for i := uint64(0); i < steps; i++ {
		calls[i] = atomic.SwapUint32(&p.calls[i], 0)
		sizes[i] = minSize << i
		callsSum += calls[i]
	}

	sort(calls[:], sizes[:])

	defaultSize := sizes[0]
	maxSize := defaultSize

	maxSum := uint32(float64(callsSum) * maxPercentile)
	callsSum = 0
	for i := 0; i < steps; i++ {
		if callsSum > maxSum {
			break
		}
		callsSum += calls[i]
		if sizes[i] > maxSize {
			maxSize = sizes[i]
		}
	}

	atomic.StoreUint32(&p.defaultSize, defaultSize)
	atomic.StoreUint32(&p.maxSize, maxSize)

	atomic.StoreUint32(&p.calibrating, 0)
}

func sort[T types.Unsigned](a, b []T) {
	for i := 1; i < len(a); i++ {
		j := i - 1

		// Move elements of `a[0...i-1]` that are smaller than `current` (descending order)
		for j >= 0 && a[j] < a[i] {
			a[j+1] = a[j]
			b[j+1] = b[j]
			j--
		}
		// Place the current element at its correct position
		a[j+1] = a[i]
		b[j+1] = b[i]
	}
}

func index(n int) int {
	n--
	n >>= minBitSize

	// Convert n to 0 if n<=0, else n stays n. This ensures idx=0 if n<=0.
	cleanN := n & ^(n >> 31)

	// idx = number of shifts until zero = bits.Len(n)
	idx := bits.Len64(uint64(cleanN))

	// Clamp idx to [0, steps-1]
	m := steps - 1
	mask := (m - idx) >> 31
	idx = (idx & ^mask) | (m & mask)

	return idx
}
