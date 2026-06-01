// Package ringbuffer implements a lock-free ring buffer.
// Used for communication between user and translator.
package ringbuffer

import (
	"sync/atomic"
)

// RingBuffer struct with cursors and buffer
type RingBuffer[T any] struct {
	wPos, rPos uint64
	bufSize    uint64
	buf        []string
	isClosed   atomic.Int32 // 0 - not closed, 1 - closed
}

// NewRB creates a new RingBuffer with a given buffer size.
func NewRB[T any](bufSize uint64) *RingBuffer[T] {
	return &RingBuffer[T]{
		wPos:    0,
		rPos:    0,
		bufSize: bufSize,
		buf:     make([]string, bufSize),
	}
}

// Write writes a float32 slice to the buffer.
// Returns number of float32s written.
func (b *RingBuffer[T]) Write(val string) {
	if val == "" {
		return
	}

	for {
		w := atomic.LoadUint64(&b.wPos)
		r := atomic.LoadUint64(&b.rPos)

		if (w - r) >= b.bufSize {
			atomic.CompareAndSwapUint64(&b.wPos, w, w+1)
			continue
		}

		pos := w % b.bufSize
		b.buf[pos] = val

		atomic.AddUint64(&b.wPos, 1)
		return
	}
}

// Read reads a float32 slice from the buffer.
// Returns number of float32s read.
func (b *RingBuffer[T]) Read(p *string) int {
	for {
		w := atomic.LoadUint64(&b.wPos)
		r := atomic.LoadUint64(&b.rPos)

		available := w - r
		if available > 0 {
			n := min(uint64(len(*p)), available)

			pos := r % b.bufSize

			*p = b.buf[pos]

			atomic.AddUint64(&b.rPos, n)
			return int(n)
		}

		if b.IsClosed() {
			return -1
		}
	}
}

// ForEach iterates over all values in the buffer.
func (b *RingBuffer[T]) ForEach(yield func(val string)) {
	for i := 0; i < b.Len(); i++ {
		yield(b.buf[(b.rPos+uint64(i))%b.bufSize])
	}
}

// Reset resets the buffer.
func (b *RingBuffer[T]) Reset() {
	atomic.StoreUint64(&b.wPos, 0)
	atomic.StoreUint64(&b.rPos, 0)
	b.isClosed.Store(0)
}

// Len returns the current length of the buffer.
func (b *RingBuffer[T]) Len() int {
	return int(atomic.LoadUint64(&b.wPos) - atomic.LoadUint64(&b.rPos))
}

// Close the buffer.
func (b *RingBuffer[T]) Close() {
	b.isClosed.Store(1)
}

// IsClosed returns true if the buffer is closed.
func (b *RingBuffer[T]) IsClosed() bool {
	return b.isClosed.Load() == 1
}

// Open the buffer.
func (b *RingBuffer[T]) Open() {
	b.isClosed.Store(0)
}
