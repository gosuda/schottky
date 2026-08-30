package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	expectedInputSHA256 = "59b1597578c8098a1791c288f743d083ce2a94b989918dee8a69780a3cd8dd1b"
	mappingCountBits    = 5
)

var profileOrder = [...]string{
	"croatian",
	"czech",
	"danish",
	"esperanto",
	"estonian",
	"german-phonebook",
	"hungarian",
	"icelandic",
	"latvian",
	"lithuanian",
	"persian",
	"polish",
	"roman",
	"romanian",
	"sinhala",
	"slovak",
	"slovenian",
	"spanish-modern",
	"spanish-traditional",
	"swedish",
	"turkish",
	"vietnamese",
}

type artifact struct {
	Profiles []profile `json:"profiles"`
}

type profile struct {
	ID       string    `json:"id"`
	Mappings []mapping `json:"mappings"`
}

type mapping struct {
	Input   []string `json:"input"`
	Weights struct {
		Primary   []string `json:"level_1"`
		Secondary []string `json:"level_2"`
		Tertiary  []string `json:"level_3"`
	} `json:"weights"`
}

type record struct {
	first   uint32
	second  uint32
	mapping uint32
}

func main() {
	input := flag.String("input", "tailoring_oracle.json", "clean-room oracle JSON")
	output := flag.String("output", "tailoring_generated.go", "generated Go output")
	flag.Parse()
	if err := generate(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(input, output string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	actualHash := fmt.Sprintf("%x", sha256.Sum256(data))
	if actualHash != expectedInputSHA256 {
		return fmt.Errorf("input SHA-256 is %s, want %s", actualHash, expectedInputSHA256)
	}
	var source artifact
	if err := json.Unmarshal(data, &source); err != nil {
		return fmt.Errorf("decode input: %w", err)
	}
	if len(source.Profiles) != len(profileOrder) {
		return fmt.Errorf("profile count is %d, want %d", len(source.Profiles), len(profileOrder))
	}

	profiles := make(map[string]profile, len(source.Profiles))
	for _, candidate := range source.Profiles {
		if _, exists := profiles[candidate.ID]; exists {
			return fmt.Errorf("duplicate profile %q", candidate.ID)
		}
		profiles[candidate.ID] = candidate
	}

	bounds := make([]uint16, len(profileOrder)+2)
	var records []record
	weights := []uint16{0, 0, 0}
	weightOffsets := map[string]uint32{"\x00\x00\x00\x00\x00\x00": 0}
	for profileIndex, profileID := range profileOrder {
		candidate, exists := profiles[profileID]
		if !exists {
			return fmt.Errorf("missing profile %q", profileID)
		}
		bounds[profileIndex+1] = uint16(len(records))
		for _, sourceMapping := range candidate.Mappings {
			parsed, err := buildRecord(sourceMapping, &weights, weightOffsets)
			if err != nil {
				return fmt.Errorf("profile %s: %w", profileID, err)
			}
			records = append(records, parsed)
		}
		sort.Slice(records[bounds[profileIndex+1]:], func(i, j int) bool {
			left := records[int(bounds[profileIndex+1])+i]
			right := records[int(bounds[profileIndex+1])+j]
			return left.first < right.first || left.first == right.first && left.second < right.second
		})
		bounds[profileIndex+2] = uint16(len(records))
	}
	if len(records) >= 1<<16 || len(weights) >= 1<<(32-mappingCountBits) {
		return fmt.Errorf("generated table exceeds packed offsets")
	}

	generated, err := format.Source(render(bounds, records, weights))
	if err != nil {
		return fmt.Errorf("format output: %w", err)
	}
	if err := os.WriteFile(output, generated, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func buildRecord(source mapping, weights *[]uint16, offsets map[string]uint32) (record, error) {
	if len(source.Input) < 1 || len(source.Input) > 2 {
		return record{}, fmt.Errorf("input length is %d, want 1 or 2", len(source.Input))
	}
	first, err := parseHex(source.Input[0], 0x10FFFF)
	if err != nil {
		return record{}, err
	}
	second := uint32(0)
	if len(source.Input) == 2 {
		second, err = parseHex(source.Input[1], 0x10FFFF)
		if err != nil {
			return record{}, err
		}
		if second == 0 {
			return record{}, fmt.Errorf("two-scalar input uses U+0000 sentinel")
		}
	}

	count := max(len(source.Weights.Primary), len(source.Weights.Secondary), len(source.Weights.Tertiary))
	if count == 0 || count >= 1<<mappingCountBits {
		return record{}, fmt.Errorf("weight count is %d", count)
	}
	triples := make([]uint16, count*3)
	for level, sourceWeights := range [][]string{
		source.Weights.Primary,
		source.Weights.Secondary,
		source.Weights.Tertiary,
	} {
		for index, text := range sourceWeights {
			value, err := parseHex(text, 0xFFFF)
			if err != nil {
				return record{}, err
			}
			triples[index*3+level] = uint16(value)
		}
	}
	key := weightKey(triples)
	offset, exists := offsets[key]
	if !exists {
		offset = uint32(len(*weights))
		*weights = append(*weights, triples...)
		offsets[key] = offset
	}
	return record{
		first:   first,
		second:  second,
		mapping: offset<<mappingCountBits | uint32(count),
	}, nil
}

func parseHex(text string, maximum uint32) (uint32, error) {
	value, err := strconv.ParseUint(text, 16, 32)
	if err != nil || value > uint64(maximum) || value >= 0xD800 && value <= 0xDFFF {
		return 0, fmt.Errorf("invalid hexadecimal value %q", text)
	}
	return uint32(value), nil
}

func weightKey(values []uint16) string {
	var builder strings.Builder
	builder.Grow(len(values) * 2)
	for _, value := range values {
		builder.WriteByte(byte(value >> 8))
		builder.WriteByte(byte(value))
	}
	return builder.String()
}

func render(bounds []uint16, records []record, weights []uint16) []byte {
	var output strings.Builder
	output.WriteString("// Code generated by internal/uca/tailorgen; DO NOT EDIT.\n")
	output.WriteString("// Clean-room SQL oracle artifact SHA-256 ")
	output.WriteString(strings.ToUpper(expectedInputSHA256))
	output.WriteString(".\n\npackage uca\n\n")
	output.WriteString("type tailoringRecord struct {\n\tfirst uint32\n\tsecond uint32\n\tmapping uint32\n}\n\n")
	output.WriteString("var tailoringProfileBounds = [...]uint16{\n")
	for index, value := range bounds {
		if index%12 == 0 {
			output.WriteString("\t")
		}
		fmt.Fprintf(&output, "%d, ", value)
		if index%12 == 11 || index == len(bounds)-1 {
			output.WriteString("\n")
		}
	}
	output.WriteString("}\n\nvar tailoringRecords = [...]tailoringRecord{\n")
	for _, value := range records {
		fmt.Fprintf(&output, "\t{first: 0x%04X, second: 0x%04X, mapping: 0x%08X},\n", value.first, value.second, value.mapping)
	}
	output.WriteString("}\n\nvar tailoringWeights = [...]uint16{\n")
	for index, value := range weights {
		if index%12 == 0 {
			output.WriteString("\t")
		}
		fmt.Fprintf(&output, "0x%04X, ", value)
		if index%12 == 11 || index == len(weights)-1 {
			output.WriteString("\n")
		}
	}
	output.WriteString("}\n")
	return []byte(output.String())
}

func max(values ...int) int {
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}
