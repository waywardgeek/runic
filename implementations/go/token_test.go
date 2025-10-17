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

func TestTokenDumpTest(t *testing.T) {
	filepath := NewFilepath("test_filepath", nil, false)
	filepath.Text = "first line\nsecond line"
	keytab := NewKeytab()
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	loc1 := NewLocation(filepath, 0, 5, 1)
	loc2 := NewLocation(filepath, 6, 4, 1)
	locNL := NewLocation(filepath, 10, 1, 1)
	loc3 := NewLocation(filepath, 11, 6, 2)
	loc4 := NewLocation(filepath, 18, 4, 2)

	token1 := NewToken(lexer, TokenTypeIdent, loc1, nil, NewValue(nil))
	token2 := NewToken(lexer, TokenTypeIdent, loc2, nil, NewValue(nil))
	newline := NewToken(lexer, TokenTypeKeyword, locNL, nil, NewValue(nil))
	token3 := NewToken(lexer, TokenTypeIdent, loc3, nil, NewValue(nil))
	token4 := NewToken(lexer, TokenTypeIdent, loc4, nil, NewValue(nil))

	// Just verify they were created
	if token1 == nil || token2 == nil || newline == nil || token3 == nil || token4 == nil {
		t.Errorf("Failed to create tokens")
	}
	token1.Dump()
	token2.Dump()
	newline.Dump()
	token3.Dump()
	token4.Dump()
}

func TestTokenEofToken(t *testing.T) {
	filepath := NewFilepath("test_filepath", nil, false)
	keytab := NewKeytab()
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	location := NewLocation(filepath, 0, 0, 1)
	token := NewToken(lexer, TokenTypeEof, location, nil, NewValue(nil))

	if token.Type != TokenTypeEof {
		t.Errorf("Expected TokenTypeEof, got %v", token.Type)
	}
	if !token.IsEof() {
		t.Errorf("IsEof() should return true")
	}
}

func TestTokenNewValueToken(t *testing.T) {
	filepath := NewFilepath("test_filepath", nil, false)
	keytab := NewKeytab()
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	location := NewLocation(filepath, 0, 0, 1)

	tests := []struct {
		value     interface{}
		tokenType TokenType
		name      string
	}{
		{"string token", TokenTypeString, "string"},
		{true, TokenTypeBool, "bool true"},
		{false, TokenTypeBool, "bool false"},
		{int8(123), TokenTypeInteger, "int8"},
		{uint32(123456789), TokenTypeInteger, "uint32"},
		{big.NewInt(123), TokenTypeInteger, "Bigint"},
		{float64(3.14), TokenTypeFloat, "float64"},
		{NewSym("identifier"), TokenTypeIdent, "ident"},
		{keytab.New("keyword"), TokenTypeKeyword, "keyword"},
	}

	for _, test := range tests {
		token := NewValueToken(lexer, test.value, location)
		if token.Type != test.tokenType {
			t.Errorf("%s: expected type %v, got %v", test.name, test.tokenType, token.Type)
		}
	}
}

func TestTokenIsTokenValue(t *testing.T) {
	filepath := NewFilepath("test_filepath", nil, false)
	keytab := NewKeytab()
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	location := NewLocation(filepath, 0, 0, 1)

	// String tests
	tokenStr := NewValueToken(lexer, "string token", location)
	if !tokenStr.IsValue("string token") {
		t.Errorf("String token should match 'string token'")
	}
	if tokenStr.IsValue("bad token") {
		t.Errorf("String token should not match 'bad token'")
	}

	// Bool tests
	tokenTrue := NewValueToken(lexer, true, location)
	if !tokenTrue.IsValue(true) {
		t.Errorf("Bool token should match true")
	}
	if tokenTrue.IsValue(false) {
		t.Errorf("Bool token should not match false")
	}

	// Integer tests
	token8 := NewValueToken(lexer, int8(123), location)
	if !token8.IsValue(int8(123)) {
		t.Errorf("Int8 token should match 123")
	}

	tokenU32 := NewValueToken(lexer, uint32(123456789), location)
	if !tokenU32.IsValue(uint32(123456789)) {
		t.Errorf("Uint32 token should match 123456789")
	}

	tokenBig := NewValueToken(lexer, big.NewInt(123), location)
	if !tokenBig.IsValue(big.NewInt(123)) {
		t.Errorf("Bigint token should match 123")
	}

	// Float tests
	tokenFloat := NewValueToken(lexer, float64(3.14), location)
	if !tokenFloat.IsValue(3.14) {
		t.Errorf("Float token should match 3.14")
	}

	// Sym tests
	sym := NewSym("identifier")
	tokenSym := NewValueToken(lexer, sym, location)
	if !tokenSym.IsValue(sym) {
		t.Errorf("Sym token should match identifier")
	}
}

func TestTokenIsTokenKeyword(t *testing.T) {
	filepath := NewFilepath("test_filepath", nil, false)
	keytab := NewKeytab()
	lexer, err := NewLexer(filepath, keytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	location := NewLocation(filepath, 0, 0, 1)

	kw := keytab.New("test_kw")
	token := NewToken(lexer, TokenTypeKeyword, location, kw, NewValue(nil))

	if !token.IsKeyword("test_kw") {
		t.Errorf("IsKeyword should return true for 'test_kw'")
	}
	if token.IsKeyword("other_kw") {
		t.Errorf("IsKeyword should return false for 'other_kw'")
	}
}
