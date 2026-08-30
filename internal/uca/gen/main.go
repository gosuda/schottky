package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	allkeysURL                 = "https://www.unicode.org/Public/UCA/14.0.0/allkeys.txt"
	allkeysSHA256              = "c38aa26b5afe8564caf683120db2d165f8f67fec231614f3517b02996aa94804"
	propListURL                = "https://www.unicode.org/Public/14.0.0/ucd/PropList.txt"
	propListSHA256             = "6bddfdb850417a5bee6deff19290fd1b138589909afb50f5a049f343bf2c6722"
	unicodeDataURL             = "https://www.unicode.org/Public/14.0.0/ucd/UnicodeData.txt"
	unicodeDataSHA256          = "36018e68657fdcb3485f636630ffe8c8532e01c977703d2803f5b89d6c5feafb"
	mappingCountBits           = 5
	mappingCountMask           = 1<<mappingCountBits - 1
	maxScalarCollationElements = 8
)

type options struct {
	output      string
	allkeys     string
	propList    string
	unicodeData string
}

type entry struct {
	codePoints []uint32
	weights    []uint16
	mapping    uint32
}

type codePointRange struct {
	first uint32
	last  uint32
}

type implicitDirective struct {
	codePointRange
	lead   uint16
	origin uint32
}

type contraction struct {
	first   uint32
	second  uint32
	third   uint32
	mapping uint32
}

type caseMapping struct {
	codePoint uint32
	upper     uint32
}

type caseRange struct {
	first  uint32
	last   uint32
	delta  int32
	stride uint32
}

func main() {
	var config options
	flag.StringVar(&config.output, "output", "tables_generated.go", "generated Go output")
	flag.StringVar(&config.allkeys, "allkeys", "", "local allkeys.txt instead of downloading")
	flag.StringVar(&config.propList, "prop-list", "", "local PropList.txt instead of downloading")
	flag.StringVar(&config.unicodeData, "unicode-data", "", "local UnicodeData.txt instead of downloading")
	flag.Parse()

	if err := generate(config); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(config options) error {
	allkeys, err := loadPinned(config.allkeys, allkeysURL, allkeysSHA256)
	if err != nil {
		return fmt.Errorf("load allkeys: %w", err)
	}
	propList, err := loadPinned(config.propList, propListURL, propListSHA256)
	if err != nil {
		return fmt.Errorf("load PropList: %w", err)
	}
	unicodeData, err := loadPinned(config.unicodeData, unicodeDataURL, unicodeDataSHA256)
	if err != nil {
		return fmt.Errorf("load UnicodeData: %w", err)
	}

	entries, directives, err := parseAllkeys(allkeys)
	if err != nil {
		return err
	}
	unified, err := parsePropertyRanges(propList, "Unified_Ideograph")
	if err != nil {
		return fmt.Errorf("parse PropList: %w", err)
	}
	upperRanges, err := parseSimpleUpper(unicodeData)
	if err != nil {
		return fmt.Errorf("parse UnicodeData: %w", err)
	}

	generated, err := buildOutput(entries, directives, unified, upperRanges)
	if err != nil {
		return err
	}
	generated, err = format.Source(generated)
	if err != nil {
		return fmt.Errorf("format output: %w", err)
	}
	if err := os.WriteFile(config.output, generated, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", config.output, err)
	}
	return nil
}

func loadPinned(path, sourceURL, expectedHash string) ([]byte, error) {
	var data []byte
	var err error
	if path != "" {
		data, err = os.ReadFile(path)
	} else {
		data, err = download(sourceURL)
	}
	if err != nil {
		return nil, err
	}

	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if actual != expectedHash {
		return nil, fmt.Errorf("SHA-256 is %s, want %s", actual, expectedHash)
	}
	return data, nil
}

func download(sourceURL string) (data []byte, err error) {
	response, err := http.Get(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", sourceURL, err)
	}
	defer func() {
		err = errors.Join(err, response.Body.Close())
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", sourceURL, response.Status)
	}
	data, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", sourceURL, err)
	}
	return data, nil
}

func parseAllkeys(data []byte) ([]entry, []implicitDirective, error) {
	var entries []entry
	var directives []implicitDirective
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" || strings.HasPrefix(line, "@version") {
			continue
		}
		if strings.HasPrefix(line, "@implicitweights") {
			directive, err := parseImplicitDirective(line)
			if err != nil {
				return nil, nil, fmt.Errorf("allkeys line %d: %w", lineNumber, err)
			}
			directives = append(directives, directive)
			continue
		}
		if strings.HasPrefix(line, "@") {
			return nil, nil, fmt.Errorf("allkeys line %d: unsupported directive %q", lineNumber, line)
		}
		parsed, err := parseEntry(line)
		if err != nil {
			return nil, nil, fmt.Errorf("allkeys line %d: %w", lineNumber, err)
		}
		entries = append(entries, parsed)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan allkeys: %w", err)
	}
	return entries, directives, nil
}

func parseImplicitDirective(line string) (implicitDirective, error) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "@implicitweights"))
	parts := strings.Split(line, ";")
	if len(parts) != 2 {
		return implicitDirective{}, fmt.Errorf("invalid implicit directive %q", line)
	}
	codePoints, err := parseRange(strings.TrimSpace(parts[0]))
	if err != nil {
		return implicitDirective{}, err
	}
	lead, err := parseHex(strings.TrimSpace(parts[1]), 0xFFFF)
	if err != nil {
		return implicitDirective{}, fmt.Errorf("invalid implicit lead: %w", err)
	}
	return implicitDirective{codePointRange: codePoints, lead: uint16(lead)}, nil
}

func parseEntry(line string) (entry, error) {
	parts := strings.Split(line, ";")
	if len(parts) != 2 {
		return entry{}, fmt.Errorf("invalid entry %q", line)
	}

	fields := strings.Fields(parts[0])
	if len(fields) == 0 || len(fields) > 3 {
		return entry{}, fmt.Errorf("invalid code point sequence %q", parts[0])
	}
	parsed := entry{codePoints: make([]uint32, len(fields))}
	for index, field := range fields {
		codePoint, err := parseHex(field, 0x10FFFF)
		if err != nil || codePoint >= 0xD800 && codePoint <= 0xDFFF {
			return entry{}, fmt.Errorf("invalid code point %q", field)
		}
		if len(fields) > 1 && codePoint == 0 {
			return entry{}, fmt.Errorf("zero code point cannot be encoded in a contraction")
		}
		parsed.codePoints[index] = codePoint
	}

	weights, err := parseElements(strings.TrimSpace(parts[1]))
	if err != nil {
		return entry{}, err
	}
	parsed.weights = weights
	return parsed, nil
}

func parseElements(text string) ([]uint16, error) {
	var weights []uint16
	for len(text) != 0 {
		if text[0] != '[' {
			return nil, fmt.Errorf("invalid collation elements %q", text)
		}
		end := strings.IndexByte(text, ']')
		if end < 0 {
			return nil, fmt.Errorf("unterminated collation element %q", text)
		}
		element := text[1:end]
		if len(element) < 2 || element[0] != '.' && element[0] != '*' {
			return nil, fmt.Errorf("invalid collation element %q", element)
		}
		fields := strings.Split(element[1:], ".")
		if len(fields) < 3 || len(fields) > 4 {
			return nil, fmt.Errorf("invalid collation element %q", element)
		}
		for _, field := range fields[:3] {
			weight, err := parseHex(field, 0xFFFF)
			if err != nil {
				return nil, fmt.Errorf("invalid weight %q: %w", field, err)
			}
			weights = append(weights, uint16(weight))
		}
		if len(fields) == 4 {
			if _, err := parseHex(fields[3], 0xFFFF); err != nil {
				return nil, fmt.Errorf("invalid quaternary weight %q: %w", fields[3], err)
			}
		}
		text = strings.TrimSpace(text[end+1:])
	}
	if len(weights) == 0 || len(weights)/3 > mappingCountMask {
		return nil, fmt.Errorf("invalid collation-element count %d", len(weights)/3)
	}
	return weights, nil
}

func parsePropertyRanges(data []byte, property string) ([]codePointRange, error) {
	var ranges []codePointRange
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: invalid property entry", lineNumber)
		}
		if strings.TrimSpace(parts[1]) != property {
			continue
		}
		parsed, err := parseRange(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		ranges = append(ranges, parsed)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mergeRanges(ranges), nil
}

func parseSimpleUpper(data []byte) ([]caseRange, error) {
	var mappings []caseMapping
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		fields := strings.Split(scanner.Text(), ";")
		if len(fields) != 15 {
			return nil, fmt.Errorf("line %d: got %d UnicodeData fields, want 15", lineNumber, len(fields))
		}
		if fields[12] == "" {
			continue
		}
		codePoint, err := parseHex(fields[0], 0x10FFFF)
		if err != nil {
			return nil, fmt.Errorf("line %d code point: %w", lineNumber, err)
		}
		upper, err := parseHex(fields[12], 0x10FFFF)
		if err != nil {
			return nil, fmt.Errorf("line %d uppercase mapping: %w", lineNumber, err)
		}
		if len(mappings) != 0 && codePoint <= mappings[len(mappings)-1].codePoint {
			return nil, fmt.Errorf("line %d: uppercase mappings are not strictly ordered", lineNumber)
		}
		mappings = append(mappings, caseMapping{codePoint: codePoint, upper: upper})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return compressCaseMappings(mappings), nil
}

func compressCaseMappings(mappings []caseMapping) []caseRange {
	ranges := make([]caseRange, 0, len(mappings))
	for start := 0; start < len(mappings); {
		delta := int32(int64(mappings[start].upper) - int64(mappings[start].codePoint))
		bestEnd := start
		bestStride := uint32(1)
		for _, stride := range [...]uint32{1, 2} {
			end := start
			for end+1 < len(mappings) &&
				mappings[end+1].codePoint-mappings[end].codePoint == stride &&
				int32(int64(mappings[end+1].upper)-int64(mappings[end+1].codePoint)) == delta {
				end++
			}
			if end > bestEnd {
				bestEnd = end
				bestStride = stride
			}
		}
		ranges = append(ranges, caseRange{
			first:  mappings[start].codePoint,
			last:   mappings[bestEnd].codePoint,
			delta:  delta,
			stride: bestStride,
		})
		start = bestEnd + 1
	}
	return ranges
}

func parseRange(text string) (codePointRange, error) {
	parts := strings.Split(text, "..")
	if len(parts) < 1 || len(parts) > 2 {
		return codePointRange{}, fmt.Errorf("invalid range %q", text)
	}
	first, err := parseHex(parts[0], 0x10FFFF)
	if err != nil {
		return codePointRange{}, err
	}
	last := first
	if len(parts) == 2 {
		last, err = parseHex(parts[1], 0x10FFFF)
		if err != nil {
			return codePointRange{}, err
		}
	}
	if first > last {
		return codePointRange{}, fmt.Errorf("reversed range %q", text)
	}
	return codePointRange{first: first, last: last}, nil
}

func parseHex(text string, maximum uint32) (uint32, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(text), 16, 32)
	if err != nil || value > uint64(maximum) {
		return 0, fmt.Errorf("invalid hexadecimal value %q", text)
	}
	return uint32(value), nil
}

func mergeRanges(ranges []codePointRange) []codePointRange {
	sort.Slice(ranges, func(first, second int) bool {
		return ranges[first].first < ranges[second].first || ranges[first].first == ranges[second].first && ranges[first].last < ranges[second].last
	})
	merged := ranges[:0]
	for _, candidate := range ranges {
		if len(merged) == 0 || candidate.first > merged[len(merged)-1].last+1 {
			merged = append(merged, candidate)
			continue
		}
		if candidate.last > merged[len(merged)-1].last {
			merged[len(merged)-1].last = candidate.last
		}
	}
	return merged
}

func buildOutput(entries []entry, directives []implicitDirective, unified []codePointRange, upperRanges []caseRange) ([]byte, error) {
	if err := constrainScalarMappings(entries); err != nil {
		return nil, err
	}
	pool, err := assignMappings(entries)
	if err != nil {
		return nil, err
	}

	var scalars []entry
	var contractionEntries []contraction
	for _, item := range entries {
		if len(item.codePoints) == 1 {
			scalars = append(scalars, item)
			continue
		}
		candidate := contraction{first: item.codePoints[0], second: item.codePoints[1], mapping: item.mapping}
		if len(item.codePoints) == 3 {
			candidate.third = item.codePoints[2]
		}
		contractionEntries = append(contractionEntries, candidate)
	}
	sort.Slice(scalars, func(first, second int) bool { return scalars[first].codePoints[0] < scalars[second].codePoints[0] })
	for index := 1; index < len(scalars); index++ {
		if scalars[index-1].codePoints[0] == scalars[index].codePoints[0] {
			return nil, fmt.Errorf("duplicate scalar mapping U+%04X", scalars[index].codePoints[0])
		}
	}
	sort.Slice(contractionEntries, func(first, second int) bool {
		a, b := contractionEntries[first], contractionEntries[second]
		return a.first < b.first || a.first == b.first && (a.second < b.second || a.second == b.second && a.third < b.third)
	})
	for index := 1; index < len(contractionEntries); index++ {
		if contractionEntries[index-1].first == contractionEntries[index].first && contractionEntries[index-1].second == contractionEntries[index].second && contractionEntries[index-1].third == contractionEntries[index].third {
			return nil, fmt.Errorf("duplicate contraction mapping")
		}
	}

	special, err := implicitRanges(directives)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	writeHeader(&output)
	writeUint16Array(&output, "collationWeights", pool, 12)
	writeScalarArrays(&output, scalars)
	writeContractions(&output, contractionEntries)
	writeUnifiedRanges(&output, unified)
	writeImplicitRanges(&output, special)
	writeCaseRanges(&output, upperRanges)
	return output.Bytes(), nil
}

func constrainScalarMappings(entries []entry) error {
	truncated := 0
	for index := range entries {
		if len(entries[index].codePoints) != 1 || len(entries[index].weights)/3 <= maxScalarCollationElements {
			continue
		}
		entries[index].weights = entries[index].weights[:maxScalarCollationElements*3]
		truncated++
	}
	if truncated != 1 {
		return fmt.Errorf("truncated scalar mapping count is %d, want 1", truncated)
	}
	return nil
}

func assignMappings(entries []entry) ([]uint16, error) {
	mappingByWeights := make(map[string]uint32)
	var pool []uint16
	for index := range entries {
		keyBytes := make([]byte, len(entries[index].weights)*2)
		for at, weight := range entries[index].weights {
			binary.BigEndian.PutUint16(keyBytes[at*2:], weight)
		}
		key := string(keyBytes)
		if mapping, found := mappingByWeights[key]; found {
			entries[index].mapping = mapping
			continue
		}
		count := len(entries[index].weights) / 3
		if len(pool) > int(^uint32(0)>>mappingCountBits) {
			return nil, fmt.Errorf("collation-weight pool is too large")
		}
		mapping := uint32(len(pool))<<mappingCountBits | uint32(count)
		entries[index].mapping = mapping
		mappingByWeights[key] = mapping
		pool = append(pool, entries[index].weights...)
	}
	return pool, nil
}

func implicitRanges(directives []implicitDirective) ([]implicitDirective, error) {
	origins := make(map[uint16]uint32)
	for _, directive := range directives {
		origin, found := origins[directive.lead]
		if !found || directive.first < origin {
			origins[directive.lead] = directive.first
		}
	}

	clamped := 0
	ranges := make([]implicitDirective, 0, len(directives))
	for _, directive := range directives {
		if directive.first == 0x18D00 && directive.last > 0x18D7F {
			directive.last = 0x18D7F
			clamped++
		}
		directive.origin = origins[directive.lead]
		ranges = append(ranges, directive)
	}
	if clamped != 1 {
		return nil, fmt.Errorf("clamped implicit range count is %d, want 1", clamped)
	}
	sort.Slice(ranges, func(first, second int) bool { return ranges[first].first < ranges[second].first })
	return ranges, nil
}

func writeHeader(output *bytes.Buffer) {
	fmt.Fprintf(output, "// Code generated by internal/uca/gen; DO NOT EDIT.\n")
	fmt.Fprintf(output, "//\n")
	fmt.Fprintf(output, "// Unicode 14.0.0 data sources:\n")
	fmt.Fprintf(output, "//   %s (SHA-256 %s)\n", allkeysURL, allkeysSHA256)
	fmt.Fprintf(output, "//   %s (SHA-256 %s)\n", propListURL, propListSHA256)
	fmt.Fprintf(output, "//   %s (SHA-256 %s)\n", unicodeDataURL, unicodeDataSHA256)
	fmt.Fprintf(output, "// Copyright 2021 Unicode, Inc. Distributed under the Unicode License v3;\n")
	fmt.Fprintf(output, "// see LICENSE-UNICODE.txt.\n\n")
	fmt.Fprintf(output, "package uca\n\n")
	fmt.Fprintf(output, "const (\n\tmappingCountBits = %d\n\tmappingCountMask = %d\n)\n\n", mappingCountBits, mappingCountMask)
	fmt.Fprintf(output, "type contractionRecord struct {\n\tfirst uint32\n\tsecond uint32\n\tthird uint32\n\tmapping uint32\n}\n\n")
	fmt.Fprintf(output, "type codePointRange struct {\n\tfirst uint32\n\tlast uint32\n}\n\n")
	fmt.Fprintf(output, "type implicitRange struct {\n\tfirst uint32\n\tlast uint32\n\torigin uint32\n\tlead uint16\n}\n\n")
	fmt.Fprintf(output, "type caseRange struct {\n\tfirst uint32\n\tlast uint32\n\tdelta int32\n\tstride uint32\n}\n\n")
}

func writeUint16Array(output *bytes.Buffer, name string, values []uint16, columns int) {
	fmt.Fprintf(output, "var %s = [...]uint16{\n", name)
	for index, value := range values {
		if index%columns == 0 {
			output.WriteByte('\t')
		}
		fmt.Fprintf(output, "0x%04X,", value)
		if index%columns == columns-1 || index == len(values)-1 {
			output.WriteByte('\n')
		} else {
			output.WriteByte(' ')
		}
	}
	fmt.Fprintf(output, "}\n\n")
}

func writeScalarArrays(output *bytes.Buffer, scalars []entry) {
	fmt.Fprintf(output, "var scalarCodePoints = [...]uint32{\n")
	for index, item := range scalars {
		if index%8 == 0 {
			output.WriteByte('\t')
		}
		fmt.Fprintf(output, "0x%X,", item.codePoints[0])
		if index%8 == 7 || index == len(scalars)-1 {
			output.WriteByte('\n')
		} else {
			output.WriteByte(' ')
		}
	}
	fmt.Fprintf(output, "}\n\nvar scalarMappings = [...]uint32{\n")
	for index, item := range scalars {
		if index%8 == 0 {
			output.WriteByte('\t')
		}
		fmt.Fprintf(output, "0x%X,", item.mapping)
		if index%8 == 7 || index == len(scalars)-1 {
			output.WriteByte('\n')
		} else {
			output.WriteByte(' ')
		}
	}
	fmt.Fprintf(output, "}\n\n")
}

func writeContractions(output *bytes.Buffer, entries []contraction) {
	fmt.Fprintf(output, "var contractions = [...]contractionRecord{\n")
	for _, item := range entries {
		fmt.Fprintf(output, "\t{first: 0x%X, second: 0x%X, third: 0x%X, mapping: 0x%X},\n", item.first, item.second, item.third, item.mapping)
	}
	fmt.Fprintf(output, "}\n\n")
}

func writeUnifiedRanges(output *bytes.Buffer, ranges []codePointRange) {
	fmt.Fprintf(output, "var unifiedIdeographRanges = [...]codePointRange{\n")
	for _, item := range ranges {
		fmt.Fprintf(output, "\t{first: 0x%X, last: 0x%X},\n", item.first, item.last)
	}
	fmt.Fprintf(output, "}\n\n")
}

func writeImplicitRanges(output *bytes.Buffer, ranges []implicitDirective) {
	fmt.Fprintf(output, "var specialImplicitRanges = [...]implicitRange{\n")
	for _, item := range ranges {
		fmt.Fprintf(output, "\t{first: 0x%X, last: 0x%X, origin: 0x%X, lead: 0x%04X},\n", item.first, item.last, item.origin, item.lead)
	}
	fmt.Fprintf(output, "}\n")
}

func writeCaseRanges(output *bytes.Buffer, ranges []caseRange) {
	fmt.Fprintf(output, "\nvar simpleUpperRanges = [...]caseRange{\n")
	for _, item := range ranges {
		fmt.Fprintf(output, "\t{first: 0x%X, last: 0x%X, delta: %d, stride: %d},\n", item.first, item.last, item.delta, item.stride)
	}
	fmt.Fprintf(output, "}\n")
}
