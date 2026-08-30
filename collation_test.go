package schottky_test

import (
	"bytes"
	"errors"
	"testing"

	"gosuda.org/schottky"
)

func TestUnicodeCollationLevels(t *testing.T) {
	tests := []struct {
		name    string
		profile schottky.CollationProfile
		left    string
		right   string
		want    int
	}{
		{name: "primary case", left: "e", right: "E", want: 0},
		{name: "primary accent", left: "e", right: "é", want: 0},
		{name: "primary decomposition", left: "é", right: "e\u0301", want: 0},
		{name: "primary expansion", left: "ß", right: "ss", want: 0},
		{name: "secondary accent", profile: schottky.CollationProfile{AccentSensitive: true}, left: "e", right: "é", want: -1},
		{name: "tertiary case", profile: schottky.CollationProfile{CaseSensitive: true}, left: "e", right: "E", want: -1},
		{name: "supplementary", left: "😀", right: "😁", want: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collator, err := schottky.NewCollator(test.profile)
			if err != nil {
				t.Fatalf("NewCollator() error = %v", err)
			}
			left := collationKey(t, collator, test.left, false)
			right := collationKey(t, collator, test.right, false)
			got := bytes.Compare(left, right)
			if got < 0 {
				got = -1
			} else if got > 0 {
				got = 1
			}
			if got != test.want {
				t.Fatalf("compare(%q, %q) = %d, want %d; keys=(%x, %x)", test.left, test.right, got, test.want, left, right)
			}
		})
	}
}

func TestUnicodeCollationDerivedMappings(t *testing.T) {
	collator, err := schottky.NewCollator(schottky.CollationProfile{
		AccentSensitive: true,
		CaseSensitive:   true,
	})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	if !bytes.Equal(
		collationKey(t, collator, "l·", false),
		collationKey(t, collator, "ŀ", false),
	) {
		t.Fatal("explicit contraction and scalar keys differ")
	}
	if bytes.Compare(
		collationKey(t, collator, "\u0378", false),
		collationKey(t, collator, "\u0379", false),
	) >= 0 {
		t.Fatal("implicit key for U+0378 must precede U+0379")
	}
	if bytes.Equal(
		collationKey(t, collator, "가", false),
		collationKey(t, collator, "가", false),
	) {
		t.Fatal("profile must not decompose Hangul")
	}
}

func TestUnicodeCollationProfileMatrix(t *testing.T) {
	count := 0
	for tailoring := schottky.TailoringRoot; tailoring <= schottky.TailoringVietnamese; tailoring++ {
		for padding := schottky.NoPadding; padding <= schottky.SpacePadding; padding++ {
			for _, accentSensitive := range []bool{false, true} {
				for _, caseSensitive := range []bool{false, true} {
					_, err := schottky.NewCollator(schottky.CollationProfile{
						Tailoring:       tailoring,
						Padding:         padding,
						AccentSensitive: accentSensitive,
						CaseSensitive:   caseSensitive,
					})
					if err != nil {
						t.Fatalf("profile %d/%d/%t/%t rejected: %v", tailoring, padding, accentSensitive, caseSensitive, err)
					}
					count++
				}
			}
		}
	}
	if count != 184 {
		t.Fatalf("profile count = %d, want 184", count)
	}
}

func TestUnicodeCollationTailoredOrdering(t *testing.T) {
	tests := []struct {
		name      string
		tailoring schottky.CollationTailoring
		values    []string
		equal     bool
	}{
		{name: "Czech", tailoring: schottky.TailoringCzech, values: []string{"h", "ch", "i"}},
		{name: "Danish", tailoring: schottky.TailoringDanish, values: []string{"z", "æ", "ø", "å"}},
		{name: "German phonebook", tailoring: schottky.TailoringGermanPhonebook, values: []string{"ä", "ae"}, equal: true},
		{name: "traditional Spanish", tailoring: schottky.TailoringSpanishTraditional, values: []string{"c", "ch", "d"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collator, err := schottky.NewCollator(schottky.CollationProfile{Tailoring: test.tailoring})
			if err != nil {
				t.Fatalf("NewCollator() error = %v", err)
			}
			for index := 1; index < len(test.values); index++ {
				left := collationKey(t, collator, test.values[index-1], false)
				right := collationKey(t, collator, test.values[index], false)
				comparison := bytes.Compare(left, right)
				if test.equal {
					if comparison != 0 {
						t.Fatalf("compare(%q, %q) = %d, want equality", test.values[index-1], test.values[index], comparison)
					}
				} else if comparison >= 0 {
					t.Fatalf("compare(%q, %q) = %d, want ascending", test.values[index-1], test.values[index], comparison)
				}
			}
		})
	}
}

func TestCollationPadding(t *testing.T) {
	spacePadded, err := schottky.NewCollator(schottky.CollationProfile{Padding: schottky.SpacePadding})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	noPadding, err := schottky.NewCollator(schottky.CollationProfile{})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	if !bytes.Equal(
		collationKey(t, spacePadded, "a", false),
		collationKey(t, spacePadded, "a ", false),
	) {
		t.Fatal("space-padded keys differ")
	}
	if bytes.Compare(
		collationKey(t, noPadding, "a", false),
		collationKey(t, noPadding, "a ", false),
	) >= 0 {
		t.Fatal("no-padding key for a must precede a-space")
	}
}

func TestBinaryCollationProfiles(t *testing.T) {
	noPadding, err := schottky.NewCollator(schottky.CollationProfile{Algorithm: schottky.BinaryCollation})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	spacePadded, err := schottky.NewCollator(schottky.CollationProfile{
		Algorithm: schottky.BinaryCollation,
		Padding:   schottky.SpacePadding,
	})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	if bytes.Compare(
		collationKey(t, noPadding, "A", false),
		collationKey(t, noPadding, "a", false),
	) >= 0 {
		t.Fatal("binary key for A must precede a")
	}
	if !bytes.Equal(
		collationKey(t, spacePadded, "a", false),
		collationKey(t, spacePadded, "a ", false),
	) {
		t.Fatal("binary space-padded keys differ")
	}
}

func TestSimpleCaseCollation(t *testing.T) {
	collator, err := schottky.NewCollator(schottky.CollationProfile{
		Algorithm:       schottky.SimpleCaseCollation,
		Padding:         schottky.SpacePadding,
		AccentSensitive: true,
	})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	if !bytes.Equal(
		collationKey(t, collator, "e", false),
		collationKey(t, collator, "E", false),
	) {
		t.Fatal("simple-case keys must fold case")
	}
	if !bytes.Equal(
		collationKey(t, collator, "𐐨", false),
		collationKey(t, collator, "𐐀", false),
	) {
		t.Fatal("simple-case keys must fold supplementary case")
	}
	if bytes.Compare(
		collationKey(t, collator, "e", false),
		collationKey(t, collator, "é", false),
	) >= 0 {
		t.Fatal("simple-case key for e must precede e-acute")
	}
	if bytes.Equal(
		collationKey(t, collator, "é", false),
		collationKey(t, collator, "e\u0301", false),
	) {
		t.Fatal("simple-case profile must not expand or normalize")
	}
	if !bytes.Equal(
		collationKey(t, collator, "a", false),
		collationKey(t, collator, "a ", false),
	) {
		t.Fatal("simple-case profile must apply space padding")
	}
}

func TestCollationTotalKeyAddsRawTieBreak(t *testing.T) {
	collator, err := schottky.NewCollator(schottky.CollationProfile{})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	leftComparison := collationKey(t, collator, "e", false)
	rightComparison := collationKey(t, collator, "é", false)
	if !bytes.Equal(leftComparison, rightComparison) {
		t.Fatalf("comparison keys differ: %x != %x", leftComparison, rightComparison)
	}
	leftTotal := collationKey(t, collator, "e", true)
	rightTotal := collationKey(t, collator, "é", true)
	if bytes.Compare(leftTotal, rightTotal) >= 0 {
		t.Fatalf("total keys = (%x, %x), want left < right", leftTotal, rightTotal)
	}
}

func TestCollatorRejectsInvalidInputAtomically(t *testing.T) {
	invalidProfiles := []schottky.CollationProfile{
		{Algorithm: schottky.BinaryCollation, AccentSensitive: true},
		{Algorithm: schottky.BinaryCollation, Tailoring: schottky.TailoringCzech},
		{Tailoring: schottky.CollationTailoring(255)},
	}
	for _, profile := range invalidProfiles {
		if _, err := schottky.NewCollator(profile); !errors.Is(err, schottky.ErrInvalidValue) {
			t.Fatalf("NewCollator(%+v) error = %v, want ErrInvalidValue", profile, err)
		}
	}
	collator, err := schottky.NewCollator(schottky.CollationProfile{})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	initial := []byte{1, 2, 3}
	if key, err := collator.Key(initial, "\xff"); !errors.Is(err, schottky.ErrInvalidValue) || !bytes.Equal(key, initial) {
		t.Fatalf("Key(invalid UTF-8) = (%x, %v)", key, err)
	}
	short := make([]byte, 1, 1)
	if key, err := collator.Key(short, "valid"); !errors.Is(err, schottky.ErrShortBuffer) || !bytes.Equal(key, short) {
		t.Fatalf("Key(short) = (%x, %v)", key, err)
	}
}

func TestCollationKeyIntegratesWithBuilderDirection(t *testing.T) {
	collator, err := schottky.NewCollator(schottky.CollationProfile{AccentSensitive: true, CaseSensitive: true})
	if err != nil {
		t.Fatalf("NewCollator() error = %v", err)
	}
	first := collationKey(t, collator, "e", false)
	second := collationKey(t, collator, "é", false)
	firstKey := buildKey(t, func(builder *schottky.Builder) {
		builder.CollationKey(first, schottky.DescNullsFirst)
	})
	secondKey := buildKey(t, func(builder *schottky.Builder) {
		builder.CollationKey(second, schottky.DescNullsFirst)
	})
	if bytes.Compare(firstKey, secondKey) <= 0 {
		t.Fatalf("descending keys = (%x, %x), want first > second", firstKey, secondKey)
	}
}

func collationKey(t *testing.T, collator schottky.Collator, value string, total bool) []byte {
	t.Helper()
	size, err := collator.KeySize(value)
	if total {
		size, err = collator.TotalKeySize(value)
	}
	if err != nil {
		t.Fatalf("collation size error = %v", err)
	}
	storage := make([]byte, 0, size)
	var key []byte
	if total {
		key, err = collator.TotalKey(storage, value)
	} else {
		key, err = collator.Key(storage, value)
	}
	if err != nil {
		t.Fatalf("collation key error = %v", err)
	}
	if len(key) != size {
		t.Fatalf("collation key length = %d, want %d", len(key), size)
	}
	return key
}
