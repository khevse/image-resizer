package buffer

// Buffer with limit
type Buffer struct {
	data []byte
	size int
}

// New buffer object
func New(maxSize int) *Buffer {
	return &Buffer{
		data: make([]byte, 0, maxSize),
	}
}

// Write to buffer
func (b *Buffer) Write(p []byte) (n int, err error) {

	b.size += len(p)

	if b.isOverflow() {
		b.data = b.data[:0]
		return 0, nil
	}

	b.data = append(b.data, p...)

	return len(p), nil
}

// Get buffer data
func (b Buffer) Get() ([]byte, bool) {
	return b.data, !b.isOverflow()
}

func (b Buffer) isOverflow() bool {
	return b.size > cap(b.data)
}
