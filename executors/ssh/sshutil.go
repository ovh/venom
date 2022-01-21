package ssh

import (
	"bytes"
	"sync"
)

// Buffer thread safe
type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}

// Read thread safe
func (b *Buffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

// Write thread safe
func (b *Buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

// String thread safe
func (b *Buffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}

// Bytes thread safe
func (b *Buffer) Bytes() []byte {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Bytes()
}

// Len thread safe
func (b *Buffer) Len() int {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Len()
}

// Truncate thread safe
func (b *Buffer) Truncate(n int) {
	b.m.Lock()
	defer b.m.Unlock()
	b.b.Truncate(n)
}
