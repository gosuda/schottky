package schottky_test

import (
	"errors"
	"math"
	"testing"

	"gosuda.org/schottky"
)

func TestInt64Ordering(t *testing.T) {
	values := []int64{math.MinInt64, -1, 0, 1, math.MaxInt64}
	ascending := make([][]byte, 0, len(values))
	descending := make([][]byte, 0, len(values))
	for _, value := range values {
		ascending = append(ascending, buildKey(t, func(builder *schottky.Builder) {
			builder.Int64(value, schottky.AscNullsFirst)
		}))
		descending = append(descending, buildKey(t, func(builder *schottky.Builder) {
			builder.Int64(value, schottky.DescNullsFirst)
		}))
	}
	assertIncreasing(t, ascending)
	assertDecreasingValues(t, descending)
}

func TestInt128OrderingAndRoundTrip(t *testing.T) {
	values := []schottky.Int128{
		{High: math.MinInt64},
		{High: -1, Low: math.MaxUint64},
		{},
		{Low: 1},
		{High: math.MaxInt64, Low: math.MaxUint64},
	}
	keys := make([][]byte, 0, len(values))
	for _, value := range values {
		keys = append(keys, buildKey(t, func(builder *schottky.Builder) {
			builder.Int128(value, schottky.AscNullsFirst)
		}))
	}
	assertIncreasing(t, keys)

	decoder := schottky.NewDecoder(keys[1])
	decoded, presence := decoder.Int128(schottky.AscNullsFirst)
	if decoded != values[1] || presence != schottky.Present || decoder.Err() != nil {
		t.Fatalf("Int128() = (%v, %v, %v), want (%v, Present, nil)", decoded, presence, decoder.Err(), values[1])
	}
}

func TestIntegerWidthsRoundTrip(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Int8(math.MinInt8, schottky.AscNullsFirst)
		builder.Int16(math.MaxInt16, schottky.DescNullsLast)
		builder.Int32(-1234567, schottky.AscNullsLast)
		builder.Int64(math.MinInt64, schottky.DescNullsFirst)
		builder.Uint8(math.MaxUint8, schottky.AscNullsFirst)
		builder.Uint16(math.MaxUint16, schottky.DescNullsLast)
		builder.Uint32(math.MaxUint32, schottky.AscNullsLast)
		builder.Uint64(math.MaxUint64, schottky.DescNullsFirst)
	})
	decoder := schottky.NewDecoder(key)
	int8Value, _ := decoder.Int8(schottky.AscNullsFirst)
	int16Value, _ := decoder.Int16(schottky.DescNullsLast)
	int32Value, _ := decoder.Int32(schottky.AscNullsLast)
	int64Value, _ := decoder.Int64(schottky.DescNullsFirst)
	uint8Value, _ := decoder.Uint8(schottky.AscNullsFirst)
	uint16Value, _ := decoder.Uint16(schottky.DescNullsLast)
	uint32Value, _ := decoder.Uint32(schottky.AscNullsLast)
	uint64Value, _ := decoder.Uint64(schottky.DescNullsFirst)

	if int8Value != math.MinInt8 || int16Value != math.MaxInt16 || int32Value != -1234567 || int64Value != math.MinInt64 {
		t.Fatalf("signed round trip = (%d, %d, %d, %d)", int8Value, int16Value, int32Value, int64Value)
	}
	if uint8Value != math.MaxUint8 || uint16Value != math.MaxUint16 || uint32Value != math.MaxUint32 || uint64Value != math.MaxUint64 {
		t.Fatalf("unsigned round trip = (%d, %d, %d, %d)", uint8Value, uint16Value, uint32Value, uint64Value)
	}
	if decoder.Err() != nil || decoder.Remaining() != 0 {
		t.Fatalf("decoder state: error=%v remaining=%d", decoder.Err(), decoder.Remaining())
	}
}

func TestFloat64Ordering(t *testing.T) {
	values := []float64{math.Inf(-1), -math.MaxFloat64, -1, -math.SmallestNonzeroFloat64, 0, math.SmallestNonzeroFloat64, 1, math.MaxFloat64, math.Inf(1), math.NaN()}
	ascending := make([][]byte, 0, len(values))
	descending := make([][]byte, 0, len(values))
	for _, value := range values {
		ascending = append(ascending, buildKey(t, func(builder *schottky.Builder) {
			builder.Float64(value, schottky.AscNullsLast)
		}))
		descending = append(descending, buildKey(t, func(builder *schottky.Builder) {
			builder.Float64(value, schottky.DescNullsLast)
		}))
	}
	assertIncreasing(t, ascending)
	assertDecreasingValues(t, descending)
}

func TestFloatCanonicalValues(t *testing.T) {
	negativeZero := math.Copysign(0, -1)
	positiveZeroKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Float64(0, schottky.AscNullsFirst)
	})
	negativeZeroKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Float64(negativeZero, schottky.AscNullsFirst)
	})
	if string(positiveZeroKey) != string(negativeZeroKey) {
		t.Fatalf("zero keys differ: +0=%x -0=%x", positiveZeroKey, negativeZeroKey)
	}

	firstNaN := math.Float64frombits(0x7ff8000000000001)
	secondNaN := math.Float64frombits(0xfff0000000000001)
	firstKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Float64(firstNaN, schottky.AscNullsFirst)
	})
	secondKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Float64(secondNaN, schottky.AscNullsFirst)
	})
	if string(firstKey) != string(secondKey) {
		t.Fatalf("NaN keys differ: first=%x second=%x", firstKey, secondKey)
	}
}

func TestFloatsRoundTrip(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Float32(-12.5, schottky.DescNullsFirst)
		builder.Float64(math.NaN(), schottky.AscNullsLast)
	})
	decoder := schottky.NewDecoder(key)
	float32Value, float32Presence := decoder.Float32(schottky.DescNullsFirst)
	float64Value, float64Presence := decoder.Float64(schottky.AscNullsLast)
	if float32Value != -12.5 || float32Presence != schottky.Present {
		t.Fatalf("Float32() = (%v, %v), want (-12.5, Present)", float32Value, float32Presence)
	}
	if !math.IsNaN(float64Value) || float64Presence != schottky.Present {
		t.Fatalf("Float64() = (%v, %v), want (NaN, Present)", float64Value, float64Presence)
	}
	if decoder.Err() != nil || decoder.Remaining() != 0 {
		t.Fatalf("decoder state: error=%v remaining=%d", decoder.Err(), decoder.Remaining())
	}
}

func TestFloatDecoderRejectsNonCanonicalValues(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{name: "negative zero", key: []byte{1, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{name: "noncanonical NaN", key: []byte{1, 0xff, 0xf8, 0, 0, 0, 0, 0, 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := schottky.NewDecoder(test.key)
			decoder.Float64(schottky.AscNullsFirst)
			if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
				t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
			}
		})
	}
}
