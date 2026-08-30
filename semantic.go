package schottky

const microsPerDay int64 = 86_400_000_000

// Date appends signed days since 1970-01-01.
func (b *Builder) Date(days int32, order Order) {
	b.Int32(days, order)
}

// Time appends microseconds since midnight.
func (b *Builder) Time(microseconds int64, order Order) {
	if microseconds < 0 || microseconds >= microsPerDay {
		b.fail(ErrInvalidValue)
		return
	}
	b.Int64(microseconds, order)
}

// Timestamp appends microseconds from the schema-defined epoch.
func (b *Builder) Timestamp(microseconds int64, order Order) {
	b.Int64(microseconds, order)
}

// Duration appends signed elapsed microseconds.
func (b *Builder) Duration(microseconds int64, order Order) {
	b.Int64(microseconds, order)
}

// Enum appends an immutable enum rank.
func (b *Builder) Enum(rank uint32, order Order) {
	b.Uint32(rank, order)
}

// LSN appends an unsigned 64-bit log position.
func (b *Builder) LSN(value uint64, order Order) {
	b.Uint64(value, order)
}

func (d *Decoder) Date(order Order) (int32, Presence) {
	return d.Int32(order)
}

func (d *Decoder) Time(order Order) (int64, Presence) {
	value, presence := d.Int64(order)
	if d.err != nil || presence != Present {
		return value, presence
	}
	if value < 0 || value >= microsPerDay {
		d.err = ErrMalformedKey
		return 0, Present
	}
	return value, Present
}

func (d *Decoder) Timestamp(order Order) (int64, Presence) {
	return d.Int64(order)
}

func (d *Decoder) Duration(order Order) (int64, Presence) {
	return d.Int64(order)
}

func (d *Decoder) Enum(order Order) (uint32, Presence) {
	return d.Uint32(order)
}

func (d *Decoder) LSN(order Order) (uint64, Presence) {
	return d.Uint64(order)
}
