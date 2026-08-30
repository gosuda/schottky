package schottky

// Fixed field sizes include the presence tag.
const (
	NullSize       = 1
	BoolSize       = 2
	Int8Size       = 2
	Int16Size      = 3
	Int32Size      = 5
	Int64Size      = 9
	Uint8Size      = 2
	Uint16Size     = 3
	Uint32Size     = 5
	Uint64Size     = 9
	Float32Size    = 5
	Float64Size    = 9
	DateSize       = Int32Size
	TimeSize       = Int64Size
	TimestampSize  = Int64Size
	DurationSize   = Int64Size
	EnumSize       = Uint32Size
	LSNSize        = Uint64Size
	UUIDSize       = 17
	MAC48Size      = 7
	MAC64Size      = 9
	IPv4Size       = 6
	IPv6Size       = 18
	IPv4PrefixSize = 7
	IPv6PrefixSize = 19
)

// EncodedBytesSize returns the exact field size for value, including presence.
func EncodedBytesSize(value []byte) (int, error) {
	payloadSize, ok := escapedBytesSize(value)
	if !ok || payloadSize == int(^uint(0)>>1) {
		return 0, ErrInvalidValue
	}
	return payloadSize + 1, nil
}

// EncodedStringSize returns the exact field size for value, including presence.
func EncodedStringSize(value string) (int, error) {
	payloadSize, ok := escapedStringSize(value)
	if !ok || payloadSize == int(^uint(0)>>1) {
		return 0, ErrInvalidValue
	}
	return payloadSize + 1, nil
}

// MaxEncodedBinarySize returns the worst-case field size for a binary length.
func MaxEncodedBinarySize(length int) (int, error) {
	maxInt := int(^uint(0) >> 1)
	if length < 0 || length > (maxInt-3)/2 {
		return 0, ErrInvalidValue
	}
	return 2*length + 3, nil
}

// EncodedDecimalSize returns the exact normalized decimal field size.
func EncodedDecimalSize(value string) (int, error) {
	parts, ok := parseDecimal(value)
	if !ok {
		return 0, ErrInvalidValue
	}
	if parts.class == decimalNegativeFinite || parts.class == decimalPositiveFinite {
		return 1 + 1 + 4 + parts.digitCount + 1, nil
	}
	return 2, nil
}
