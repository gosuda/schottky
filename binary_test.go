package schottky_test

import (
	"bytes"
	"errors"
	"testing"

	"gosuda.org/schottky"
)

func TestBytesOrdering(t *testing.T) {
	values := [][]byte{nil, {0}, {0, 0}, {0, 1}, {1}, {1, 0}, {0xff}}
	ascending := make([][]byte, 0, len(values))
	descending := make([][]byte, 0, len(values))
	for _, value := range values {
		ascending = append(ascending, buildKey(t, func(builder *schottky.Builder) {
			builder.Bytes(value, schottky.AscNullsFirst)
		}))
		descending = append(descending, buildKey(t, func(builder *schottky.Builder) {
			builder.Bytes(value, schottky.DescNullsFirst)
		}))
	}
	assertIncreasing(t, ascending)
	assertDecreasingValues(t, descending)
}

func TestVariableFieldsRoundTrip(t *testing.T) {
	bytesValue := []byte{0, 1, 0, 0xff}
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Bytes(bytesValue, schottky.DescNullsLast)
		builder.String("A\x00Z", schottky.AscNullsFirst)
		builder.CollationKey([]byte{3, 0, 2}, schottky.DescNullsFirst)
		builder.Tuple([]byte{1, 0, 0}, schottky.AscNullsLast)
	})
	decoder := schottky.NewDecoder(key)
	var storage [64]byte
	decoded := storage[:0]
	decoded, presence := decoder.Bytes(decoded, schottky.DescNullsLast)
	if presence != schottky.Present || string(decoded) != string(bytesValue) {
		t.Fatalf("Bytes() = (%x, %v), want (%x, Present)", decoded, presence, bytesValue)
	}
	start := len(decoded)
	decoded, _ = decoder.String(decoded, schottky.AscNullsFirst)
	if string(decoded[start:]) != "A\x00Z" {
		t.Fatalf("String() = %x, want 41005a", decoded[start:])
	}
	start = len(decoded)
	decoded, _ = decoder.CollationKey(decoded, schottky.DescNullsFirst)
	if string(decoded[start:]) != string([]byte{3, 0, 2}) {
		t.Fatalf("CollationKey() = %x, want 030002", decoded[start:])
	}
	start = len(decoded)
	decoded, _ = decoder.Tuple(decoded, schottky.AscNullsLast)
	if string(decoded[start:]) != string([]byte{1, 0, 0}) {
		t.Fatalf("Tuple() = %x, want 010000", decoded[start:])
	}
	if decoder.Err() != nil || decoder.Remaining() != 0 {
		t.Fatalf("decoder state: error=%v remaining=%d", decoder.Err(), decoder.Remaining())
	}
}

func TestBytesShortDecodeDoesNotWrite(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Bytes([]byte("abcd"), schottky.AscNullsFirst)
	})
	decoder := schottky.NewDecoder(key)
	storage := make([]byte, 1, 4)
	storage[0] = 0xaa
	decoded, _ := decoder.Bytes(storage, schottky.AscNullsFirst)
	if !errors.Is(decoder.Err(), schottky.ErrShortBuffer) {
		t.Fatalf("Err() = %v, want ErrShortBuffer", decoder.Err())
	}
	if len(decoded) != 1 || decoded[0] != 0xaa {
		t.Fatalf("decoded after short buffer = %x, want aa", decoded)
	}
}

func TestDecoderRejectsMalformedBinary(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{name: "missing terminator", key: []byte{1, 'a'}},
		{name: "truncated escape", key: []byte{1, 0}},
		{name: "invalid escape", key: []byte{1, 0, 1}},
		{name: "invalid presence", key: []byte{2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := schottky.NewDecoder(test.key)
			var storage [8]byte
			decoder.Bytes(storage[:0], schottky.AscNullsFirst)
			if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
				t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
			}
		})
	}
}

func TestStringEncodingMatchesBytes(t *testing.T) {
	value := "\x00A\xff"
	stringKey := buildKey(t, func(builder *schottky.Builder) {
		builder.String(value, schottky.DescNullsLast)
	})
	bytesKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Bytes([]byte(value), schottky.DescNullsLast)
	})
	if string(stringKey) != string(bytesKey) {
		t.Fatalf("String key = %x, Bytes key = %x", stringKey, bytesKey)
	}
}

func TestDescendingBytesFormat(t *testing.T) {
	value := bytes.Repeat([]byte{0x55}, 64)
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Bytes(value, schottky.DescNullsFirst)
	})
	want := append([]byte{1}, bytes.Repeat([]byte{0xaa}, 64)...)
	want = append(want, 0xff, 0xff)
	if !bytes.Equal(key, want) {
		t.Fatalf("descending key = %x, want %x", key, want)
	}
}

func TestEscapedBytesFormat(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Bytes([]byte{0, 1}, schottky.AscNullsFirst)
	})
	want := []byte{1, 0, 0xff, 1, 0, 0}
	if !bytes.Equal(key, want) {
		t.Fatalf("ascending key = %x, want %x", key, want)
	}
}
