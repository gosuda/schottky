package schottky_test

import (
	"bytes"
	"testing"

	"gosuda.org/schottky"
)

func TestArrayCompatibilityLayout(t *testing.T) {
	one, two, seven, zero := int32(1), int32(2), int32(7), int32(0)
	withValue := compatibilityArrayKey(t, []*int32{&one, &seven}, []int32{2}, []int32{1})
	withNull := compatibilityArrayKey(t, []*int32{&one, nil}, []int32{2}, []int32{1})
	if bytes.Compare(withValue, withNull) >= 0 {
		t.Fatalf("non-null/null element keys = (%x, %x), want non-null < null", withValue, withNull)
	}

	shorter := compatibilityArrayKey(t, []*int32{&one, &two}, []int32{2}, []int32{1})
	longer := compatibilityArrayKey(t, []*int32{&one, &two, &zero}, []int32{3}, []int32{1})
	if bytes.Compare(shorter, longer) >= 0 {
		t.Fatalf("array prefix keys = (%x, %x), want shorter < longer", shorter, longer)
	}

	oneDimension := compatibilityArrayKey(t, []*int32{&one, &two}, []int32{2}, []int32{1})
	twoDimensions := compatibilityArrayKey(t, []*int32{&one, &two}, []int32{1, 2}, []int32{1, 1})
	if bytes.Compare(oneDimension, twoDimensions) >= 0 {
		t.Fatalf("array rank keys = (%x, %x), want rank one < rank two", oneDimension, twoDimensions)
	}
}

func TestRangeCompatibilityLayout(t *testing.T) {
	one, two := int32(1), int32(2)
	empty := compatibilityRangeKey(t, true, nil, false, nil, false)
	singleton := compatibilityRangeKey(t, false, &one, true, &one, true)
	if bytes.Compare(empty, singleton) >= 0 {
		t.Fatalf("range class keys = (%x, %x), want empty < non-empty", empty, singleton)
	}

	rightOpen := compatibilityRangeKey(t, false, &one, true, &two, false)
	closed := compatibilityRangeKey(t, false, &one, true, &two, true)
	leftOpen := compatibilityRangeKey(t, false, &one, false, &two, true)
	if bytes.Compare(rightOpen, closed) >= 0 || bytes.Compare(closed, leftOpen) >= 0 {
		t.Fatalf("range bound keys = (%x, %x, %x), want [1,2) < [1,2] < (1,2]", rightOpen, closed, leftOpen)
	}
}

func TestRecordCompatibilityLayout(t *testing.T) {
	one, nine := int32(1), int32(9)
	withValue := compatibilityRecordKey(t, []*int32{&one, &nine})
	withNull := compatibilityRecordKey(t, []*int32{&one, nil})
	if bytes.Compare(withValue, withNull) >= 0 {
		t.Fatalf("record keys = (%x, %x), want non-null < null", withValue, withNull)
	}
}

func compatibilityArrayKey(t *testing.T, elements []*int32, dimensions, lowerBounds []int32) []byte {
	t.Helper()
	elementKey := buildKey(t, func(builder *schottky.Builder) {
		for _, element := range elements {
			if element == nil {
				builder.Null(schottky.AscNullsLast)
			} else {
				builder.Int32(*element, schottky.AscNullsLast)
			}
		}
	})
	return buildKey(t, func(builder *schottky.Builder) {
		builder.Tuple(elementKey, schottky.AscNullsLast)
		builder.Int32(int32(len(elements)), schottky.AscNullsLast)
		builder.Int32(int32(len(dimensions)), schottky.AscNullsLast)
		for _, dimension := range dimensions {
			builder.Int32(dimension, schottky.AscNullsLast)
		}
		for _, lowerBound := range lowerBounds {
			builder.Int32(lowerBound, schottky.AscNullsLast)
		}
	})
}

func compatibilityRangeKey(
	t *testing.T,
	empty bool,
	lower *int32,
	lowerInclusive bool,
	upper *int32,
	upperInclusive bool,
) []byte {
	t.Helper()
	return buildKey(t, func(builder *schottky.Builder) {
		if empty {
			builder.Uint8(0, schottky.AscNullsLast)
			return
		}
		builder.Uint8(1, schottky.AscNullsLast)
		builder.Tuple(compatibilityBoundKey(t, lower, lowerInclusive, true), schottky.AscNullsLast)
		builder.Tuple(compatibilityBoundKey(t, upper, upperInclusive, false), schottky.AscNullsLast)
	})
}

func compatibilityBoundKey(t *testing.T, value *int32, inclusive, lower bool) []byte {
	t.Helper()
	return buildKey(t, func(builder *schottky.Builder) {
		if value == nil {
			if lower {
				builder.Uint8(0, schottky.AscNullsLast)
			} else {
				builder.Uint8(1, schottky.AscNullsLast)
			}
			return
		}
		if lower {
			builder.Uint8(1, schottky.AscNullsLast)
		} else {
			builder.Uint8(0, schottky.AscNullsLast)
		}
		valueKey := buildKey(t, func(valueBuilder *schottky.Builder) {
			valueBuilder.Int32(*value, schottky.AscNullsLast)
		})
		builder.Tuple(valueKey, schottky.AscNullsLast)
		rank := uint8(0)
		if lower != inclusive {
			rank = 1
		}
		builder.Uint8(rank, schottky.AscNullsLast)
	})
}

func compatibilityRecordKey(t *testing.T, fields []*int32) []byte {
	t.Helper()
	return buildKey(t, func(builder *schottky.Builder) {
		for _, field := range fields {
			if field == nil {
				builder.Null(schottky.AscNullsLast)
			} else {
				builder.Int32(*field, schottky.AscNullsLast)
			}
		}
	})
}
