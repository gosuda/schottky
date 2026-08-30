package schottky_test

import (
	"errors"
	"testing"

	"gosuda.org/schottky"
)

func TestDecoderRejectsTruncatedFixedField(t *testing.T) {
	decoder := schottky.NewDecoder([]byte{1, 0, 1, 2})
	decoder.Int64(schottky.AscNullsFirst)
	if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
		t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
	}
}

func TestDecoderRejectsNonCanonicalBoolean(t *testing.T) {
	decoder := schottky.NewDecoder([]byte{1, 2})
	decoder.Bool(schottky.AscNullsFirst)
	if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
		t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
	}
}

func TestDecoderRejectsTrailingSchemaMismatch(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Int64(1, schottky.AscNullsFirst)
		builder.Int64(2, schottky.AscNullsFirst)
	})
	decoder := schottky.NewDecoder(key)
	decoder.Int64(schottky.AscNullsFirst)
	if decoder.Err() != nil {
		t.Fatalf("Err() = %v", decoder.Err())
	}
	if decoder.Remaining() == 0 {
		t.Fatal("Remaining() = 0, want trailing field bytes")
	}
}

func TestDecoderResetClearsStickyError(t *testing.T) {
	decoder := schottky.NewDecoder(nil)
	decoder.Uint8(schottky.AscNullsFirst)
	if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
		t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
	}
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Uint8(7, schottky.AscNullsFirst)
	})
	decoder.Reset(key)
	value, presence := decoder.Uint8(schottky.AscNullsFirst)
	if value != 7 || presence != schottky.Present || decoder.Err() != nil || decoder.Remaining() != 0 {
		t.Fatalf("Uint8() after Reset = (%d, %v), error=%v remaining=%d", value, presence, decoder.Err(), decoder.Remaining())
	}
}
