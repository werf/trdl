package worker

import (
	"bytes"
	"sync"
)

// SafeBuffer prevents possible conflicts when shared by goroutines.
// In the current version, locking is implemented for
//
// - Write, which is used by logboek
//
// - Bytes, which can be used to get the log of the running job
type SafeBuffer struct {
	*bytes.Buffer
	m sync.Mutex
}

func NewSafeBuffer() *SafeBuffer {
	return &SafeBuffer{
		Buffer: bytes.NewBuffer([]byte{}),
		m:      sync.Mutex{},
	}
}

func (b *SafeBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.Buffer.Write(p)
}

func (b *SafeBuffer) Bytes() []byte {
	b.m.Lock()
	defer b.m.Unlock()
	return b.Buffer.Bytes()
}
