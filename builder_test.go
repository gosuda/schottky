package schottky_test

import (
	"errors"
	"testing"

	"gosuda.org/schottky"
)

func TestBuilderAppendsToExistingBuffer(t *testing.T) {
	storage := make([]byte, 2, 16)
	storage[0], storage[1] = 0xaa, 0xbb
	builder := schottky.NewBuilder(storage)
	builder.Bool(true, schottky.AscNullsFirst)
	key, err := builder.Key()
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	want := []byte{0xaa, 0xbb, 0x01, 0x01}
	if string(key) != string(want) {
		t.Fatalf("key = %x, want %x", key, want)
	}
}

func TestBuilderShortFieldIsAtomic(t *testing.T) {
	storage := make([]byte, 1, 2)
	storage[0] = 0xaa
	builder := schottky.NewBuilder(storage)
	builder.Bool(true, schottky.AscNullsFirst)
	builder.Null(schottky.AscNullsFirst)
	key, err := builder.Key()
	if !errors.Is(err, schottky.ErrShortBuffer) {
		t.Fatalf("Key() error = %v, want ErrShortBuffer", err)
	}
	if len(key) != 1 || key[0] != 0xaa {
		t.Fatalf("key after failed field = %x, want aa", key)
	}
}

func TestBuilderRejectsInvalidOrder(t *testing.T) {
	storage := make([]byte, 0, 8)
	builder := schottky.NewBuilder(storage)
	builder.Int64(1, schottky.Order(255))
	if !errors.Is(builder.Err(), schottky.ErrInvalidOrder) {
		t.Fatalf("Err() = %v, want ErrInvalidOrder", builder.Err())
	}
	if builder.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", builder.Len())
	}
}

func TestBuilderResetClearsStickyError(t *testing.T) {
	builder := schottky.NewBuilder(nil)
	builder.Null(schottky.AscNullsFirst)
	if !errors.Is(builder.Err(), schottky.ErrShortBuffer) {
		t.Fatalf("Err() = %v, want ErrShortBuffer", builder.Err())
	}

	var storage [1]byte
	builder.Reset(storage[:0])
	builder.Null(schottky.AscNullsLast)
	key, err := builder.Key()
	if err != nil {
		t.Fatalf("Key() after Reset error = %v", err)
	}
	if len(key) != 1 || key[0] != 1 {
		t.Fatalf("null key = %x, want 01", key)
	}
}

func TestNullPlacementIsIndependentOfDirection(t *testing.T) {
	orders := []struct {
		name       string
		order      schottky.Order
		nullBefore bool
	}{
		{name: "ascending nulls first", order: schottky.AscNullsFirst, nullBefore: true},
		{name: "ascending nulls last", order: schottky.AscNullsLast, nullBefore: false},
		{name: "descending nulls first", order: schottky.DescNullsFirst, nullBefore: true},
		{name: "descending nulls last", order: schottky.DescNullsLast, nullBefore: false},
	}
	for _, test := range orders {
		t.Run(test.name, func(t *testing.T) {
			nullKey := buildKey(t, func(builder *schottky.Builder) {
				builder.Null(test.order)
			})
			valueKey := buildKey(t, func(builder *schottky.Builder) {
				builder.Int64(0, test.order)
			})
			got := string(nullKey) < string(valueKey)
			if got != test.nullBefore {
				t.Fatalf("null before value = %t, want %t; null=%x value=%x", got, test.nullBefore, nullKey, valueKey)
			}
		})
	}
}

func TestDecoderReturnsZeroForNullScalars(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Null(schottky.AscNullsFirst)
		builder.Null(schottky.DescNullsLast)
	})
	decoder := schottky.NewDecoder(key)
	integer, integerPresence := decoder.Int64(schottky.AscNullsFirst)
	boolean, booleanPresence := decoder.Bool(schottky.DescNullsLast)
	if integer != 0 || integerPresence != schottky.Null {
		t.Fatalf("Int64() = (%d, %v), want (0, Null)", integer, integerPresence)
	}
	if boolean || booleanPresence != schottky.Null {
		t.Fatalf("Bool() = (%t, %v), want (false, Null)", boolean, booleanPresence)
	}
	if decoder.Err() != nil || decoder.Remaining() != 0 {
		t.Fatalf("decoder state: error=%v remaining=%d", decoder.Err(), decoder.Remaining())
	}
}
