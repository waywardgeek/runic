// Copyright 2023 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"testing"
)

func TestReadAscii(t *testing.T) {
	// Test control characters 0-31
	for val := uint8(0); val < 32; val++ {
		c := GetChar(string([]byte{val}), 0)
		isWhitespace := val == '\n' || val == '\r' || val == '\t'
		if isWhitespace && !c.Valid {
			t.Errorf("GetChar(0x%02x) should be valid (whitespace), got invalid", val)
		} else if !isWhitespace && c.Valid {
			t.Errorf("GetChar(0x%02x) should be invalid (control char), got valid", val)
		}
	}

	// Test printable ASCII 32-126
	for val := uint8(32); val < 127; val++ {
		c := GetChar(string([]byte{val}), 0)
		if !c.Valid {
			t.Errorf("GetChar(0x%02x) should be valid, got invalid", val)
		}
	}

	// Test DEL character (127)
	c := GetChar(string([]byte{127}), 0)
	if c.Valid {
		t.Errorf("GetChar(DEL, 0x7F) should be invalid, got valid")
	}
}

func TestReadUTF8(t *testing.T) {
	// Test Euro sign: € = U+20AC = E2 82 AC (3 bytes)
	char := GetChar("€", 0)
	if char.Pos != 0 || char.Len != 3 || !char.Valid {
		t.Errorf("Euro sign: expected Pos=0 Len=3 Valid=true, got Pos=%d Len=%d Valid=%v",
			char.Pos, char.Len, char.Valid)
	}

	// Test Greek text: Ἀφροδίτη
	text := "Ἀφροδίτη"
	pos := uint32(0)
	charCount := 0
	for pos < uint32(len(text)) {
		char := GetChar(text, pos)
		if !char.Valid {
			t.Errorf("Character at pos %d should be valid", pos)
		}
		pos += uint32(char.Len)
		charCount++
	}
	if charCount == 0 {
		t.Errorf("Should have parsed at least one character from Greek text")
	}
}

func TestOverlong(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"Overlong null 2-byte", "\xc0\x80"},
		{"Overlong null 3-byte", "\xe0\x80\x80"},
		{"Overlong euro", "\xf0\x82\x82\xac"},
	}

	for _, test := range tests {
		char := GetChar(test.text, 0)
		if char.Valid {
			t.Errorf("%s: should be invalid (overlong encoding), got valid", test.name)
		}
	}
}

func TestTrojanSource(t *testing.T) {
	// All Trojan source characters have first byte 0xE2
	table := [][]uint8{
		{0xE2, 0x80, 0xAA}, // LRE U+202A
		{0xE2, 0x80, 0xAB}, // RLE U+202B
		{0xE2, 0x80, 0xAC}, // PDF U+202C
		{0xE2, 0x80, 0xAD}, // LRO U+202D
		{0xE2, 0x80, 0xAE}, // RLO U+202E
		{0xE2, 0x81, 0xA6}, // LRI U+2066
		{0xE2, 0x81, 0xA7}, // RLI U+2067
		{0xE2, 0x81, 0xA8}, // FSI U+2068
		{0xE2, 0x81, 0xA9}, // PDI U+2069
	}

	for _, bytes := range table {
		s := string(bytes)
		char := GetChar(s, 0)
		if char.Valid {
			t.Errorf("Trojan source char %02X %02X %02X should be invalid, got valid",
				bytes[0], bytes[1], bytes[2])
		}
	}
}

func TestUpperLower(t *testing.T) {
	tests := []struct {
		input    uint8
		expected uint8
		fn       func(uint8) uint8
		name     string
	}{
		{'a', 'A', Upper, "Upper('a')"},
		{'n', 'N', Upper, "Upper('n')"},
		{'z', 'Z', Upper, "Upper('z')"},
		{' ', ' ', Upper, "Upper(' ')"},
		{'A', 'a', Lower, "Lower('A')"},
		{'N', 'n', Lower, "Lower('N')"},
		{'Z', 'z', Lower, "Lower('Z')"},
		{' ', ' ', Lower, "Lower(' ')"},
	}

	for _, test := range tests {
		result := test.fn(test.input)
		if result != test.expected {
			t.Errorf("%s: expected %c, got %c", test.name, test.expected, result)
		}
	}
}

func TestDigit(t *testing.T) {
	tests := []struct {
		char     uint8
		expected bool
		name     string
	}{
		{'a', false, "'a'"},
		{'A', false, "'A'"},
		{'g', false, "'g'"},
		{'G', false, "'G'"},
		{'0', true, "'0'"},
		{'9', true, "'9'"},
		{'0' - 1, false, "'0'-1"},
		{'9' + 1, false, "'9'+1"},
	}

	for _, test := range tests {
		result := IsDigit(test.char)
		if result != test.expected {
			t.Errorf("IsDigit(%s): expected %v, got %v", test.name, test.expected, result)
		}
	}
}

func TestHex(t *testing.T) {
	tests := []struct {
		char     uint8
		expected bool
		name     string
	}{
		{'a', true, "'a'"},
		{'A', true, "'A'"},
		{'g', false, "'g'"},
		{'G', false, "'G'"},
		{'0', true, "'0'"},
		{'9', true, "'9'"},
	}

	for _, test := range tests {
		result := IsHexDigit(test.char)
		if result != test.expected {
			t.Errorf("IsHexDigit(%s): expected %v, got %v", test.name, test.expected, result)
		}
	}

	// Test hex digit conversion
	if HexToChar('c', '5') != 0xc5 {
		t.Errorf("HexToChar('c', '5'): expected 0xc5, got 0x%02x", HexToChar('c', '5'))
	}

	if HexDigit('a') != 0xa {
		t.Errorf("HexDigit('a'): expected 0xa, got 0x%x", HexDigit('a'))
	}

	if HexDigit('A') != 0xa {
		t.Errorf("HexDigit('A'): expected 0xa, got 0x%x", HexDigit('A'))
	}
}

func TestWhitespace(t *testing.T) {
	tests := []struct {
		char     uint8
		expected bool
		name     string
	}{
		{'\x1b', false, "escape"},
		{0, false, "null"},
		{'_', false, "underscore"},
		{' ', true, "space"},
		{'\n', false, "newline"},
		{'\r', true, "carriage return"},
		{'\t', true, "tab"},
	}

	for _, test := range tests {
		result := IsWhitespace(test.char)
		if result != test.expected {
			t.Errorf("IsWhitespace(%s): expected %v, got %v", test.name, test.expected, result)
		}
	}
}
