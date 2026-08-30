package schottky_test

import (
	"errors"
	"testing"

	"gosuda.org/schottky"
)

func TestEncodedSizesMatchBuilder(t *testing.T) {
	bytesValue := []byte{0, 1, 0}
	bytesSize, err := schottky.EncodedBytesSize(bytesValue)
	if err != nil {
		t.Fatalf("EncodedBytesSize() error = %v", err)
	}
	bytesKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Bytes(bytesValue, schottky.AscNullsFirst)
	})
	if bytesSize != len(bytesKey) {
		t.Fatalf("EncodedBytesSize() = %d, encoded length = %d", bytesSize, len(bytesKey))
	}

	stringSize, err := schottky.EncodedStringSize("A\x00Z")
	if err != nil {
		t.Fatalf("EncodedStringSize() error = %v", err)
	}
	stringKey := buildKey(t, func(builder *schottky.Builder) {
		builder.String("A\x00Z", schottky.DescNullsLast)
	})
	if stringSize != len(stringKey) {
		t.Fatalf("EncodedStringSize() = %d, encoded length = %d", stringSize, len(stringKey))
	}

	decimalSize, err := schottky.EncodedDecimalSize("-00123.4500")
	if err != nil {
		t.Fatalf("EncodedDecimalSize() error = %v", err)
	}
	decimalKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Decimal("-00123.4500", schottky.AscNullsFirst)
	})
	if decimalSize != len(decimalKey) {
		t.Fatalf("EncodedDecimalSize() = %d, encoded length = %d", decimalSize, len(decimalKey))
	}
}

func TestMaxEncodedBinarySize(t *testing.T) {
	size, err := schottky.MaxEncodedBinarySize(3)
	if err != nil || size != 9 {
		t.Fatalf("MaxEncodedBinarySize(3) = (%d, %v), want (9, nil)", size, err)
	}
	if _, err := schottky.MaxEncodedBinarySize(-1); !errors.Is(err, schottky.ErrInvalidValue) {
		t.Fatalf("MaxEncodedBinarySize(-1) error = %v, want ErrInvalidValue", err)
	}
}

func TestEncodedDecimalSizeRejectsInvalidValue(t *testing.T) {
	if _, err := schottky.EncodedDecimalSize("1e"); !errors.Is(err, schottky.ErrInvalidValue) {
		t.Fatalf("EncodedDecimalSize() error = %v, want ErrInvalidValue", err)
	}
}
