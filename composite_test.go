package schottky_test

import (
	"bytes"
	"errors"
	"testing"

	"gosuda.org/schottky"
)

type accountRow struct {
	account int64
	name    string
	version uint32
}

func TestCompositeTupleOrdering(t *testing.T) {
	rows := []accountRow{
		{account: 1, name: "Ada", version: 2},
		{account: 1, name: "Ada", version: 1},
		{account: 1, name: "Bob", version: 9},
		{account: 2, name: "Ada", version: 3},
	}
	keys := make([][]byte, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, buildKey(t, func(builder *schottky.Builder) {
			builder.Int64(row.account, schottky.AscNullsLast)
			builder.String(row.name, schottky.AscNullsLast)
			builder.Uint32(row.version, schottky.DescNullsLast)
		}))
	}
	assertIncreasing(t, keys)
}

func TestCompletedFieldSequenceIsLiteralPrefix(t *testing.T) {
	var storage [128]byte
	builder := schottky.NewBuilder(storage[:0])
	builder.Int64(42, schottky.AscNullsLast)
	prefixLength := builder.Len()
	builder.String("Ada", schottky.DescNullsFirst)
	key, err := builder.Key()
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}
	prefix := bytes.Clone(key[:prefixLength])

	other := buildKey(t, func(builder *schottky.Builder) {
		builder.Int64(42, schottky.AscNullsLast)
		builder.String("Zoe", schottky.DescNullsFirst)
	})
	if !bytes.HasPrefix(key, prefix) || !bytes.HasPrefix(other, prefix) {
		t.Fatalf("prefix %x not shared by keys %x and %x", prefix, key, other)
	}
}

func TestNestedTuplePreservesOrder(t *testing.T) {
	firstNested := buildKey(t, func(builder *schottky.Builder) {
		builder.Int32(1, schottky.AscNullsFirst)
		builder.String("A", schottky.AscNullsFirst)
	})
	secondNested := buildKey(t, func(builder *schottky.Builder) {
		builder.Int32(1, schottky.AscNullsFirst)
		builder.String("B", schottky.AscNullsFirst)
	})
	firstOuter := buildKey(t, func(builder *schottky.Builder) {
		builder.Tuple(firstNested, schottky.AscNullsFirst)
	})
	secondOuter := buildKey(t, func(builder *schottky.Builder) {
		builder.Tuple(secondNested, schottky.AscNullsFirst)
	})
	if bytes.Compare(firstOuter, secondOuter) >= 0 {
		t.Fatalf("nested tuple order reversed: first=%x second=%x", firstOuter, secondOuter)
	}
}

func TestPrefixUpperBound(t *testing.T) {
	tests := []struct {
		name   string
		prefix []byte
		want   []byte
		ok     bool
	}{
		{name: "empty", prefix: nil, ok: false},
		{name: "simple", prefix: []byte{0x12, 0x34}, want: []byte{0x12, 0x35}, ok: true},
		{name: "trim suffix", prefix: []byte{0x12, 0xff, 0xff}, want: []byte{0x13}, ok: true},
		{name: "unbounded", prefix: []byte{0xff, 0xff}, ok: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var storage [8]byte
			bound, ok, err := schottky.PrefixUpperBound(storage[:0], test.prefix)
			if err != nil {
				t.Fatalf("PrefixUpperBound() error = %v", err)
			}
			if ok != test.ok || !bytes.Equal(bound, test.want) {
				t.Fatalf("PrefixUpperBound() = (%x, %t), want (%x, %t)", bound, ok, test.want, test.ok)
			}
		})
	}
}

func TestPrefixUpperBoundSupportsInPlaceUse(t *testing.T) {
	storage := []byte{0x12, 0x34, 0xff}
	bound, ok, err := schottky.PrefixUpperBound(storage[:0], storage)
	if err != nil || !ok || !bytes.Equal(bound, []byte{0x12, 0x35}) {
		t.Fatalf("PrefixUpperBound() = (%x, %t, %v), want (1235, true, nil)", bound, ok, err)
	}
}

func TestPrefixUpperBoundShortBufferIsAtomic(t *testing.T) {
	storage := make([]byte, 1, 2)
	storage[0] = 0xaa
	bound, ok, err := schottky.PrefixUpperBound(storage, []byte{1, 2})
	if !errors.Is(err, schottky.ErrShortBuffer) || ok {
		t.Fatalf("PrefixUpperBound() = (%x, %t, %v), want ErrShortBuffer", bound, ok, err)
	}
	if len(bound) != 1 || bound[0] != 0xaa {
		t.Fatalf("bound after short buffer = %x, want aa", bound)
	}
}
