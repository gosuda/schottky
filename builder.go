package schottky

// Builder appends fields to caller-owned storage without growing it.
type Builder struct {
	buf []byte
	err error
}

// NewBuilder starts after the existing length of dst and uses only its capacity.
func NewBuilder(dst []byte) Builder {
	return Builder{buf: dst}
}

// Reset discards builder state and starts after the existing length of dst.
func (b *Builder) Reset(dst []byte) {
	b.buf = dst
	b.err = nil
}

// Len returns the current key length.
func (b *Builder) Len() int {
	return len(b.buf)
}

// Key returns the caller-owned key and the first encoding error.
func (b *Builder) Key() ([]byte, error) {
	return b.buf, b.err
}

// Err returns the first encoding error.
func (b *Builder) Err() error {
	return b.err
}

// Null appends a null field with the requested placement.
func (b *Builder) Null(order Order) {
	if b.err != nil {
		return
	}
	if !order.valid() {
		b.err = ErrInvalidOrder
		return
	}
	if len(b.buf) == cap(b.buf) {
		b.err = ErrShortBuffer
		return
	}

	_, null := order.tags()
	b.buf = b.buf[:len(b.buf)+1]
	b.buf[len(b.buf)-1] = null
}

func (b *Builder) begin(order Order, payloadSize int) ([]byte, bool) {
	if b.err != nil {
		return nil, false
	}
	if !order.valid() {
		b.err = ErrInvalidOrder
		return nil, false
	}
	if payloadSize < 0 || payloadSize >= cap(b.buf)-len(b.buf) {
		b.err = ErrShortBuffer
		return nil, false
	}

	start := len(b.buf)
	present, _ := order.tags()
	b.buf = b.buf[:start+1+payloadSize]
	b.buf[start] = present
	return b.buf[start+1:], true
}

func (b *Builder) fail(err error) {
	if b.err == nil {
		b.err = err
	}
}
