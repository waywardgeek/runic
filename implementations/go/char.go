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

// Char describes the position and validity of one UTF-8 character.
type Char struct {
	Pos   uint32
	Len   uint8
	Valid bool // Set to false when we have an invalid UTF-8 encoding.
}

// GetChar returns a Char describing the UTF-8 character at pos in text.
func GetChar(text string, pos uint32) Char {
	if pos >= uint32(len(text)) {
		return Char{pos, 0, false}
	}

	if IsAscii(text, pos) {
		if IsValidAsciiInRuneFile(text, pos) {
			return Char{pos, 1, true}
		}
		return Char{pos, 1, false}
	}

	return readUTF8Char(text, pos)
}

// readUTF8Char reads a non-ASCII UTF-8 character at pos.
// A non-ASCII UTF-8 character will match [\xc0-\xf7][\x80-\xbf]*.
// See https://en.wikipedia.org/wiki/UTF-8 for format details.
func readUTF8Char(text string, pos uint32) Char {
	textLen := uint32(len(text))
	c := text[pos]

	// Determine expected length from first byte
	var expectedLen uint8
	if c&0x20 == 0 {
		expectedLen = 2
	} else if c&0x10 == 0 {
		expectedLen = 3
	} else if c&0x08 == 0 {
		expectedLen = 4
	} else {
		return Char{pos, 1, false}
	}

	// Check if we have enough bytes
	if pos+uint32(expectedLen) > textLen {
		return Char{pos, uint8(textLen - pos), false}
	}

	// Verify continuation bytes (should be 10xxxxxx = 0x80-0xBF)
	for i := uint8(1); i < expectedLen; i++ {
		if text[pos+uint32(i)]&0xC0 != 0x80 {
			return Char{pos, i + 1, false}
		}
	}

	// Check for overlong encoding or Trojan source characters
	if encodingIsOverlong(text, pos, expectedLen) || isTrojanSourceChar(text, pos, expectedLen) {
		return Char{pos, expectedLen, false}
	}

	return Char{pos, expectedLen, true}
}

// IsAscii returns true if the byte at pos is ASCII (< 128).
func IsAscii(text string, pos uint32) bool {
	return text[pos] < 128
}

// IsValidAsciiInRuneFile returns true if the ASCII character at pos is valid in a Rune file.
func IsValidAsciiInRuneFile(text string, pos uint32) bool {
	c := text[pos]
	if c >= ' ' && c <= '~' {
		return true
	}
	return c == '\n' || c == '\r' || c == '\t'
}

// IsAsciiAlpha returns true if the Char represents an ASCII letter.
func IsAsciiAlpha(text string, char Char) bool {
	if char.Len != 1 {
		return false
	}
	c := text[char.Pos]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// Lower converts an uppercase ASCII letter to lowercase.
func Lower(c uint8) uint8 {
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 'a'
	}
	return c
}

// Upper converts a lowercase ASCII letter to uppercase.
func Upper(c uint8) uint8 {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// IsWhitespace returns true if c is a legal whitespace character in a Rune file.
func IsWhitespace(c uint8) bool {
	return c == ' ' || c == '\t' || c == '\r'
}

// IsDigit returns true if c is a decimal digit.
func IsDigit(c uint8) bool {
	return c >= '0' && c <= '9'
}

// IsHexDigit returns true if c is a hexadecimal digit.
func IsHexDigit(c uint8) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// HexDigit converts a hex digit character to its numeric value (0-15).
func HexDigit(c uint8) uint8 {
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 10
	}
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 10
	}
	panic("Invalid hex digit: " + string(c))
}

// HexToChar converts two hex digit characters to their combined byte value.
func HexToChar(hi, lo uint8) uint8 {
	return (HexDigit(hi) << 4) | HexDigit(lo)
}

// encodingIsOverlong returns true if the UTF-8 encoding is overly long.
// All valid encodings are the shortest possible.
// E.g. 0xC041 encodes 'A', but is 2 bytes.
func encodingIsOverlong(text string, pos uint32, len uint8) bool {
	switch len {
	case 2:
		// See if the leading 4 bits post-decode would be zero
		return text[pos]&0x1E == 0
	case 3:
		// See if the leading 5 bits post-decode would be zero
		return text[pos]&0x0F == 0 && text[pos+1]&0x20 == 0
	case 4:
		// See if the leading 5 bits post-decode would be zero
		return text[pos]&0x07 == 0 && text[pos+1]&0x30 == 0
	}
	return false
}

// isTrojanSourceChar defends against Trojan source reordering attacks.
// See: https://trojansource.codes/trojan-source.pdf
//
// The characters which can be used in reordering attacks are all 14-bit characters
// requiring 3 bytes, with first byte 0xE2.
//
// - LRE U+202A Left-to-Right Embedding         => E2 80 AA
// - RLE U+202B Right-to-Left Embedding         => E2 80 AB
// - PDF U+202C Pop Directional Formatting      => E2 80 AC
// - LRO U+202D Left-to-Right Override          => E2 80 AD
// - RLO U+202E Right-to-Left Override          => E2 80 AE
// - LRI U+2066 Left-to-Right Isolate           => E2 81 A6
// - RLI U+2067 Right-to-Left Isolate           => E2 81 A7
// - FSI U+2068 First Strong Isolate            => E2 81 A8
// - PDI U+2069 Pop Directional Isolate         => E2 81 A9
func isTrojanSourceChar(text string, pos uint32, len uint8) bool {
	if len < 3 {
		return false
	}
	c1 := text[pos]
	if c1 == 0xE2 {
		c2 := text[pos+1]
		c3 := text[pos+2]
		if c2 == 0x80 {
			if c3 >= 0xAA && c3 <= 0xAE {
				return true
			}
		} else if c2 == 0x81 {
			if c3 >= 0xA6 && c3 <= 0xA9 {
				return true
			}
		}
	}
	return false
}
