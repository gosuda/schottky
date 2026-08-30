package schottky_test

import (
	"errors"
	"testing"

	"gosuda.org/schottky"
)

func TestDecimalOrdering(t *testing.T) {
	values := []string{
		"-Infinity",
		"-100",
		"-1.21",
		"-1.2",
		"-0.001",
		"0",
		"0.001",
		"1.19",
		"1.2",
		"10",
		"100",
		"Infinity",
		"NaN",
	}
	ascending := make([][]byte, 0, len(values))
	descending := make([][]byte, 0, len(values))
	for _, value := range values {
		ascending = append(ascending, buildKey(t, func(builder *schottky.Builder) {
			builder.Decimal(value, schottky.AscNullsFirst)
		}))
		descending = append(descending, buildKey(t, func(builder *schottky.Builder) {
			builder.Decimal(value, schottky.DescNullsFirst)
		}))
	}
	assertIncreasing(t, ascending)
	assertDecreasingValues(t, descending)
}

func TestDecimalCanonicalization(t *testing.T) {
	groups := [][]string{
		{"0", "-0", "+0.000e999"},
		{"1.2", "1.20", "12e-1", "+0001.2000E+0"},
		{"-1200", "-12e2", "-001200.000"},
		{"NaN", "+NaN", "-NaN"},
		{"Infinity", "+Infinity"},
	}
	for _, group := range groups {
		first := buildKey(t, func(builder *schottky.Builder) {
			builder.Decimal(group[0], schottky.AscNullsLast)
		})
		for _, value := range group[1:] {
			key := buildKey(t, func(builder *schottky.Builder) {
				builder.Decimal(value, schottky.AscNullsLast)
			})
			if string(key) != string(first) {
				t.Fatalf("Decimal(%q) = %x, want canonical key %x for %q", value, key, first, group[0])
			}
		}
	}
}

func TestDecimalRoundTrip(t *testing.T) {
	tests := []struct {
		input string
		want  string
		order schottky.Order
	}{
		{input: "-1200.00", want: "-12e2", order: schottky.AscNullsFirst},
		{input: "0.00120", want: "12e-4", order: schottky.DescNullsLast},
		{input: "0", want: "0", order: schottky.AscNullsLast},
		{input: "Infinity", want: "Infinity", order: schottky.DescNullsFirst},
		{input: "-Infinity", want: "-Infinity", order: schottky.AscNullsFirst},
		{input: "NaN", want: "NaN", order: schottky.DescNullsLast},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			key := buildKey(t, func(builder *schottky.Builder) {
				builder.Decimal(test.input, test.order)
			})
			decoder := schottky.NewDecoder(key)
			var storage [64]byte
			decoded, presence := decoder.Decimal(storage[:0], test.order)
			if string(decoded) != test.want || presence != schottky.Present {
				t.Fatalf("Decimal() = (%q, %v), want (%q, Present)", decoded, presence, test.want)
			}
			if decoder.Err() != nil || decoder.Remaining() != 0 {
				t.Fatalf("decoder state: error=%v remaining=%d", decoder.Err(), decoder.Remaining())
			}
		})
	}
}

func TestDecimalRejectsInvalidInputAtomically(t *testing.T) {
	invalid := []string{"", ".", "+", "1e", "1.2.3", "1e2e3", "one", "1e999999999999999999999"}
	for _, value := range invalid {
		t.Run(value, func(t *testing.T) {
			storage := make([]byte, 1, 64)
			storage[0] = 0xaa
			builder := schottky.NewBuilder(storage)
			builder.Decimal(value, schottky.AscNullsFirst)
			key, err := builder.Key()
			if !errors.Is(err, schottky.ErrInvalidValue) {
				t.Fatalf("Key() error = %v, want ErrInvalidValue", err)
			}
			if len(key) != 1 || key[0] != 0xaa {
				t.Fatalf("key after invalid decimal = %x, want aa", key)
			}
		})
	}
}

func TestDecimalShortDecodeDoesNotWrite(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Decimal("123.45", schottky.AscNullsFirst)
	})
	decoder := schottky.NewDecoder(key)
	storage := make([]byte, 1, 4)
	storage[0] = 0xaa
	decoded, _ := decoder.Decimal(storage, schottky.AscNullsFirst)
	if !errors.Is(decoder.Err(), schottky.ErrShortBuffer) {
		t.Fatalf("Err() = %v, want ErrShortBuffer", decoder.Err())
	}
	if len(decoded) != 1 || decoded[0] != 0xaa {
		t.Fatalf("decoded after short buffer = %x, want aa", decoded)
	}
}

func TestDecimalDecoderRejectsNonCanonicalPayload(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{name: "unknown class", key: []byte{1, 6}},
		{name: "missing magnitude", key: []byte{1, 3}},
		{name: "leading zero digit", key: []byte{1, 3, 0x80, 0, 0, 0, 1, 0}},
		{name: "trailing zero digit", key: []byte{1, 3, 0x80, 0, 0, 0, 2, 1, 0}},
		{name: "invalid digit", key: []byte{1, 3, 0x80, 0, 0, 0, 11, 0}},
		{name: "missing terminator", key: []byte{1, 3, 0x80, 0, 0, 0, 2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := schottky.NewDecoder(test.key)
			var storage [32]byte
			decoder.Decimal(storage[:0], schottky.AscNullsFirst)
			if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
				t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
			}
		})
	}
}
