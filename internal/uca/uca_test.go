package uca

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

type oracleArtifact struct {
	Counts struct {
		Mappings int `json:"mappings"`
		Profiles int `json:"profiles"`
	} `json:"counts"`
	Profiles []oracleProfile `json:"profiles"`
}

type oracleProfile struct {
	ID       string          `json:"id"`
	Mappings []oracleMapping `json:"mappings"`
}

type oracleMapping struct {
	Input   []string `json:"input"`
	Weights struct {
		Primary   []string `json:"level_1"`
		Secondary []string `json:"level_2"`
		Tertiary  []string `json:"level_3"`
	} `json:"weights"`
}

func TestTailoringOracleMappings(t *testing.T) {
	data, err := os.ReadFile("tailoring_oracle.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var artifact oracleArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if artifact.Counts.Profiles != 22 || len(artifact.Profiles) != 22 {
		t.Fatalf("profile count = %d/%d, want 22", artifact.Counts.Profiles, len(artifact.Profiles))
	}

	mappingCount := 0
	for _, profile := range artifact.Profiles {
		tailoring, ok := oracleTailoring(profile.ID)
		if !ok {
			t.Fatalf("unknown profile %q", profile.ID)
		}
		for _, mapping := range profile.Mappings {
			mappingCount++
			input := oracleInput(t, mapping.Input)
			expected := oracleKey(t, mapping)
			size, ok := KeySize(input, allLevels, tailoring)
			if !ok {
				t.Fatalf("KeySize(%q, %s) rejected", input, profile.ID)
			}
			if size != len(expected) {
				t.Fatalf("KeySize(%q, %s) = %d, want %d", input, profile.ID, size, len(expected))
			}
			got, ok := AppendKey(make([]byte, 0, size), input, allLevels, tailoring)
			if !ok {
				t.Fatalf("AppendKey(%q, %s) rejected", input, profile.ID)
			}
			if !bytes.Equal(got, expected) {
				t.Fatalf("AppendKey(%q, %s) = %X, want %X", input, profile.ID, got, expected)
			}
		}
	}
	if mappingCount != artifact.Counts.Mappings || mappingCount != 529 {
		t.Fatalf("mapping count = %d/%d, want 529", mappingCount, artifact.Counts.Mappings)
	}
}

func TestTailoringRootFallback(t *testing.T) {
	root := keyForTest(t, "x", Root)
	for tailoring := Croatian; tailoring <= Vietnamese; tailoring++ {
		if got := keyForTest(t, "x", tailoring); !bytes.Equal(got, root) {
			t.Fatalf("tailoring %d changed root fallback: %X, want %X", tailoring, got, root)
		}
	}
}

func TestRootVersionBoundaries(t *testing.T) {
	size, ok := KeySize("\uFDFA", Primary, Root)
	if !ok {
		t.Fatal("KeySize(U+FDFA) rejected")
	}
	if size != 18 {
		t.Fatalf("KeySize(U+FDFA, Primary) = %d, want 18", size)
	}

	tests := []struct {
		input string
		want  []byte
	}{
		{input: "\U00018D7F", want: []byte{0xFB, 0x00, 0x9D, 0x7F, 0, 0}},
		{input: "\U00018D80", want: []byte{0xFB, 0xC3, 0x8D, 0x80, 0, 0}},
	}
	for _, test := range tests {
		size, ok := KeySize(test.input, Primary, Root)
		if !ok {
			t.Fatalf("KeySize(%U) rejected", []rune(test.input)[0])
		}
		got, ok := AppendKey(make([]byte, 0, size), test.input, Primary, Root)
		if !ok {
			t.Fatalf("AppendKey(%U) rejected", []rune(test.input)[0])
		}
		if !bytes.Equal(got, test.want) {
			t.Fatalf("AppendKey(%U) = %X, want %X", []rune(test.input)[0], got, test.want)
		}
	}
}

func TestInvalidTailoringIsAtomic(t *testing.T) {
	dst := make([]byte, 1, 64)
	dst[0] = 0xA5
	got, ok := AppendKey(dst, "valid", allLevels, Tailoring(255))
	if ok {
		t.Fatal("AppendKey() accepted an invalid tailoring")
	}
	if len(got) != 1 || got[0] != 0xA5 {
		t.Fatalf("AppendKey() modified dst: %X", got)
	}
}

func keyForTest(t *testing.T, input string, tailoring Tailoring) []byte {
	t.Helper()
	size, ok := KeySize(input, allLevels, tailoring)
	if !ok {
		t.Fatalf("KeySize(%q, %d) rejected", input, tailoring)
	}
	key, ok := AppendKey(make([]byte, 0, size), input, allLevels, tailoring)
	if !ok {
		t.Fatalf("AppendKey(%q, %d) rejected", input, tailoring)
	}
	return key
}

func oracleInput(t *testing.T, values []string) string {
	t.Helper()
	runes := make([]rune, len(values))
	for index, text := range values {
		value, err := strconv.ParseUint(text, 16, 32)
		if err != nil {
			t.Fatalf("ParseUint(%q) error = %v", text, err)
		}
		runes[index] = rune(value)
	}
	return string(runes)
}

func oracleKey(t *testing.T, mapping oracleMapping) []byte {
	t.Helper()
	var key []byte
	for _, level := range [][]string{
		mapping.Weights.Primary,
		mapping.Weights.Secondary,
		mapping.Weights.Tertiary,
	} {
		for _, text := range level {
			value, err := strconv.ParseUint(text, 16, 16)
			if err != nil {
				t.Fatalf("ParseUint(%q) error = %v", text, err)
			}
			key = binary.BigEndian.AppendUint16(key, uint16(value))
		}
		key = binary.BigEndian.AppendUint16(key, 0)
	}
	return key
}

func oracleTailoring(id string) (Tailoring, bool) {
	switch id {
	case "croatian":
		return Croatian, true
	case "czech":
		return Czech, true
	case "danish":
		return Danish, true
	case "esperanto":
		return Esperanto, true
	case "estonian":
		return Estonian, true
	case "german-phonebook":
		return GermanPhonebook, true
	case "hungarian":
		return Hungarian, true
	case "icelandic":
		return Icelandic, true
	case "latvian":
		return Latvian, true
	case "lithuanian":
		return Lithuanian, true
	case "persian":
		return Persian, true
	case "polish":
		return Polish, true
	case "roman":
		return Roman, true
	case "romanian":
		return Romanian, true
	case "sinhala":
		return Sinhala, true
	case "slovak":
		return Slovak, true
	case "slovenian":
		return Slovenian, true
	case "spanish-modern":
		return Spanish, true
	case "spanish-traditional":
		return SpanishTraditional, true
	case "swedish":
		return Swedish, true
	case "turkish":
		return Turkish, true
	case "vietnamese":
		return Vietnamese, true
	default:
		return Root, false
	}
}
