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
	"math/big"
	"testing"
)

func newLexer(text string) *Lexer {
	filepath := NewFilepath("testdata/test", nil, false)
	filepath.Text = text + "\n"
	keytab := NewKeytab()
	createKeyword(keytab, "\n")
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		panic(err)
	}
	return lexer
}

// createKeyword is a helper to create a keyword in a keytab.
func createKeyword(keytab *Keytab, name string) *Keyword {
	return keytab.New(name)
}

func TestEmptyTest(t *testing.T) {
	filepath := NewFilepath("testdata/empty", nil, false)
	keytab := NewKeytab()
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	if lexer.Len != 0 {
		t.Errorf("Empty file should have len 0, got %d", lexer.Len)
	}
	if !lexer.Eof() {
		t.Errorf("Empty file should be at EOF")
	}
	token, err := lexer.ParseToken()
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}
	if token.Type != TokenTypeEof {
		t.Errorf("First token should be EOF, got %v", token.Type)
	}
	lexer.Close()
}

func TestParseEscapedCharsTest(t *testing.T) {
	// Test basic escape sequences
	inputStr := `"\a\b\e\f\n\r\t\v\\\"\0"`
	filepath := NewFilepath("testdata/test", nil, false)
	filepath.Text = inputStr + "\n"
	keytab := NewKeytab()
	createKeyword(keytab, "\n")
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	token, err := lexer.ParseToken()
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}
	if token.Type != TokenTypeString {
		t.Errorf("Expected TokenTypeString, got %v", token.Type)
	}
	value := token.Value.Val.(string)

	tests := []struct {
		index    int
		expected uint8
		name     string
	}{
		{0, '\x07', "Bell"},
		{1, '\x08', "Backspace"},
		{2, '\x1b', "Escape"},
		{3, '\x0c', "Formfeed"},
		{4, '\x0a', "Newline"},
		{5, '\x0d', "Return"},
		{6, '\x09', "Tab"},
		{7, '\x0b', "Vertical tab"},
		{8, '\\', "Backslash"},
		{9, '"', "Double quote"},
		{10, 0, "Null"},
	}

	for _, test := range tests {
		if test.index >= len(value) {
			t.Errorf("%s: index %d out of range (string length %d)", test.name, test.index, len(value))
			continue
		}
		actual := uint8(value[test.index])
		if actual != test.expected {
			t.Errorf("%s: expected 0x%02x, got 0x%02x", test.name, test.expected, actual)
		}
	}
}

func TestBadInputTest(t *testing.T) {
	// Test overlong encoding of '\0' - should return an error
	filepath := NewFilepath("testdata/test", nil, false)
	filepath.Text = "\xc0\x80\n"
	keytab := NewKeytab()
	createKeyword(keytab, "\n")
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	token, err := lexer.ParseToken()
	if err == nil {
		t.Errorf("Should have returned error for invalid UTF-8, got token: %v", token)
	}
	if token != nil {
		t.Errorf("Token should be nil when error is returned")
	}
}

func TestParseEscapedSingleQuotedCharsTest(t *testing.T) {
	lexer := newLexer("'\\a' '\\b' '\\e' '\\f' '\\n' '\\r' '\\t' '\\v' '\\\\' '\\x27' '\\0' '\\xde' '\\xad'")
	expRes := []uint8{
		'\x07', // Bell
		'\x08', // Backspace
		'\x1b', // Escape
		'\x0c', // Formfeed
		'\x0a', // Newline
		'\x0d', // Return
		'\x09', // Tab
		'\x0b', // Vertical tab
		'\\',
		'\'',   // Single quote (as \x27)
		0,      // Null
		'\xde',
		'\xad',
	}

	for i, expected := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.Type != TokenTypeInteger {
			t.Errorf("Token %d: expected TokenTypeInteger, got %v", i, token.Type)
			continue
		}
		val := token.Value.Val.(*big.Int)
		if val.Cmp(big.NewInt(int64(expected))) != 0 {
			t.Errorf("Token %d: expected %d, got %v", i, expected, val)
		}
	}
}

func TestParseIntegerTest(t *testing.T) {
	lexer := newLexer("0 1u2 3i3 57896044618658097711785492504343953926634992332820282019728792003956564819949u256")
	expRes := []string{
		"0",
		"1",
		"3",
		"57896044618658097711785492504343953926634992332820282019728792003956564819949",
	}

	i := 0
	for {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.IsKeyword("\n") {
			break
		}
		if i >= len(expRes) {
			t.Errorf("Too many tokens: expected %d, got more", len(expRes))
			break
		}
		if token.Type != TokenTypeInteger {
			t.Errorf("Token %d: expected TokenTypeInteger, got %v", i, token.Type)
		}
		val := token.Value.Val.(*big.Int)
		expected := new(big.Int)
		expected.SetString(expRes[i], 10)
		if val.Cmp(expected) != 0 {
			t.Errorf("Token %d: expected %s, got %v", i, expRes[i], val)
		}
		i++
	}
	if i != len(expRes) {
		t.Errorf("Expected %d tokens, got %d", len(expRes), i)
	}
}

func TestParseHexTest(t *testing.T) {
	lexer := newLexer("0x0 0xau4 0x3i3 0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffed 0xffffu256")
	expRes := []string{
		"0",
		"a",
		"3",
		"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffed",
		"ffff",
	}

	for i, expected := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.Type != TokenTypeInteger {
			t.Errorf("Token %d: expected TokenTypeInteger, got %v", i, token.Type)
			continue
		}
		val := token.Value.Val.(*big.Int)
		valStr := val.Text(16)
		if valStr != expected {
			t.Errorf("Token %d: expected %s, got %s", i, expected, valStr)
		}
	}
}

func TestParseFloatTest(t *testing.T) {
	lexer := newLexer("0. 3.14 0.999e3 2.4e-24 123456789.123456789")
	expRes := []float64{
		0.0,
		3.14,
		999.0, // 0.999e3 = 0.999 * 1000
		2.4e-24,
		123456789.123456789,
	}

	for i, expected := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.Type != TokenTypeFloat {
			t.Errorf("Token %d: expected TokenTypeFloat, got %v", i, token.Type)
			continue
		}
		val := token.Value.Val.(float64)

		// Use approximate comparison for floats
		diff := val - expected
		if diff < 0 {
			diff = -diff
		}
		if diff > expected*1e-10 && diff > 1e-10 {
			t.Errorf("Token %d: expected ~%g, got %g", i, expected, val)
		}
	}
}

func TestParseEscapedIdentTest(t *testing.T) {
	lexer := newLexer("\\if \\+ \\test")
	expRes := []string{"if", "+", "test"}

	for i, expected := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.Type != TokenTypeIdent {
			t.Errorf("Token %d: expected TokenTypeIdent, got %v", i, token.Type)
			continue
		}
		val := token.Value.Val.(*Sym)
		if val.Name != expected {
			t.Errorf("Token %d: expected %s, got %s", i, expected, val.Name)
		}
	}
}

func TestParseIdentTest(t *testing.T) {
	lexer := newLexer("schön a123 test")
	expRes := []string{"schön", "a123", "test"}

	for i, expected := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.Type != TokenTypeIdent {
			t.Errorf("Token %d: expected TokenTypeIdent, got %v", i, token.Type)
			continue
		}
		val := token.Value.Val.(*Sym)
		if val.Name != expected {
			t.Errorf("Token %d: expected %s, got %s", i, expected, val.Name)
		}
	}
}

func TestEnableUnderscoresTest(t *testing.T) {
	lexer := newLexer("$sch_ön $a1_23 _test")
	lexer.EnableIdentUnderscores(true)
	expRes := []string{"$sch_ön", "$a1_23", "_test"}

	for i, expected := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.Type != TokenTypeIdent {
			t.Errorf("Token %d: expected TokenTypeIdent, got %v", i, token.Type)
			continue
		}
		val := token.Value.Val.(*Sym)
		if val.Name != expected {
			t.Errorf("Token %d: expected %s, got %s", i, expected, val.Name)
		}
	}
}

func TestUintIntOrRandTest(t *testing.T) {
	lexer := newLexer("u32 i6 rand1_024")
	lexer.EnableIdentUnderscores(true)
	expRes := []string{"u32", "i6", "rand1024"}
	expTypes := []TokenType{TokenTypeUintType, TokenTypeIntType, TokenTypeRandUint}

	for i := 0; i < len(expRes); i++ {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if token.Type != expTypes[i] {
			t.Errorf("Token %d: expected type %v, got %v", i, expTypes[i], token.Type)
		}
		width := token.Value.Val.(*big.Int).Int64()
		expectedWidth := extractWidth(expRes[i])
		if width != expectedWidth {
			t.Errorf("Token %d: expected width %d, got %d", i, expectedWidth, width)
		}
	}
}

func extractWidth(s string) int64 {
	var num string
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			num += string(ch)
		}
	}
	result := int64(0)
	for _, ch := range num {
		result = result*10 + int64(ch-'0')
	}
	return result
}

func TestSingleLineCommentTest(t *testing.T) {
	lexer := newLexer("// Empty line\n1 2 3 // No more on this line\n// Comment above line.\n4 5")
	lexer.EnableIdentUnderscores(true)
	expRes := []struct {
		isNewline bool
		value     int64
	}{
		{true, 0},  // newline
		{false, 1},
		{false, 2},
		{false, 3},
		{true, 0},  // newline
		{true, 0},  // newline
		{false, 4},
		{false, 5},
	}

	for i, exp := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if exp.isNewline {
			if !token.IsKeyword("\n") {
				t.Errorf("Token %d: expected newline, got %v", i, token.Type)
			}
		} else {
			if token.Type != TokenTypeInteger {
				t.Errorf("Token %d: expected TokenTypeInteger, got %v", i, token.Type)
				continue
			}
			val := token.Value.Val.(*big.Int)
			if val.Int64() != exp.value {
				t.Errorf("Token %d: expected %d, got %v", i, exp.value, val)
			}
		}
	}
}

func TestBlockCommentTest(t *testing.T) {
	lexer := newLexer("/* Empty /* line\n */1 */2 3 /* No more on this line*/\n/* Comment above line.\n4*/ 5")
	lexer.EnableIdentUnderscores(true)
	expRes := []struct {
		isNewline bool
		value     int64
	}{
		{false, 2},
		{false, 3},
		{true, 0},  // newline
		{false, 5},
	}

	for i, exp := range expRes {
		token, err := lexer.ParseToken()
		if err != nil {
			t.Fatalf("Token %d: failed to parse: %v", i, err)
		}
		if exp.isNewline {
			if !token.IsKeyword("\n") {
				t.Errorf("Token %d: expected newline, got %v", i, token.Type)
			}
		} else {
			if token.Type != TokenTypeInteger {
				t.Errorf("Token %d: expected TokenTypeInteger, got %v", i, token.Type)
				continue
			}
			val := token.Value.Val.(*big.Int)
			if val.Int64() != exp.value {
				t.Errorf("Token %d: expected %d, got %v", i, exp.value, val)
			}
		}
	}
}
