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
	"fmt"
	"math/big"
)

// TokenType enumerates all token types in the Rune language.
type TokenType uint32

const (
	TokenTypeKeyword TokenType = iota
	TokenTypeIdent
	TokenTypeInteger
	TokenTypeFloat
	TokenTypeBool
	TokenTypeString
	TokenTypeWeakString // Only used in parsing PEG rules
	TokenTypeEof
	TokenTypeRandUint
	TokenTypeIntType
	TokenTypeUintType // If this is not the last anymore, fix code that assumes this.
)

// Value represents a token's value as an interface{}.
// It can hold: bool, string, *Sym, *Keyword, *big.Int, float64, etc.
type Value struct {
	Val interface{}
}

// NewValue creates a Value from various types.
func NewValue(v interface{}) Value {
	return Value{Val: v}
}

// Token represents a lexical token.
type Token struct {
	Type     TokenType
	Location Location
	Keyword  *Keyword  // For TokenTypeKeyword
	Value    Value     // For other token types
	Lexer    *Lexer
	Pexpr    interface{} // For PEG parser use (will be *Pexpr during parsing)
	
	// Previous/Next for DoublyLinked Keyword Token relation
	PrevKeywordToken *Token
	NextKeywordToken *Token
}

// NewToken creates a new token for a Lexer.
func NewToken(lexer *Lexer, tokenType TokenType, location Location, keyword *Keyword, value Value) *Token {
	token := &Token{
		Type:     tokenType,
		Location: location,
		Keyword:  keyword,
		Value:    value,
		Lexer:    lexer,
		Pexpr:    nil,
	}
	if keyword != nil {
		keyword.AppendToken(token)
	}
	lexer.AppendToken(token)
	return token
}

// AppendToken adds a token to this keyword's list (DoublyLinked relation helper).
// This is called from the Keyword side when a token is created.
func (kw *Keyword) AppendToken(token *Token) {
	// Link into doubly-linked list
	if len(kw.Tokens) > 0 {
		last := kw.Tokens[len(kw.Tokens)-1]
		last.NextKeywordToken = token
		token.PrevKeywordToken = last
	}
	kw.Tokens = append(kw.Tokens, token)
}

// NewValueToken creates a token from a value of various types.
func NewValueToken(lexer *Lexer, value interface{}, location Location) *Token {
	switch v := value.(type) {
	case string:
		return NewToken(lexer, TokenTypeString, location, nil, NewValue(v))
	case bool:
		return NewToken(lexer, TokenTypeBool, location, nil, NewValue(v))
	case int8:
		return NewToken(lexer, TokenTypeInteger, location, nil, NewValue(big.NewInt(int64(v))))
	case uint8:
		return NewToken(lexer, TokenTypeInteger, location, nil, NewValue(big.NewInt(int64(v))))
	case int32:
		return NewToken(lexer, TokenTypeInteger, location, nil, NewValue(big.NewInt(int64(v))))
	case uint32:
		return NewToken(lexer, TokenTypeInteger, location, nil, NewValue(big.NewInt(int64(v))))
	case int64:
		return NewToken(lexer, TokenTypeInteger, location, nil, NewValue(big.NewInt(v)))
	case uint64:
		return NewToken(lexer, TokenTypeInteger, location, nil, NewValue(big.NewInt(int64(v))))
	case *big.Int:
		return NewToken(lexer, TokenTypeInteger, location, nil, NewValue(v))
	case float32:
		return NewToken(lexer, TokenTypeFloat, location, nil, NewValue(float64(v)))
	case float64:
		return NewToken(lexer, TokenTypeFloat, location, nil, NewValue(v))
	case *Sym:
		return NewToken(lexer, TokenTypeIdent, location, nil, NewValue(v))
	case *Keyword:
		return NewToken(lexer, TokenTypeKeyword, location, v, NewValue(nil))
	default:
		panic(fmt.Sprintf("Unknown value type for token: %T", value))
	}
}

// IsValue checks if this token's value matches the given value.
func (t *Token) IsValue(value interface{}) bool {
	if t.Value.Val == nil {
		return false
	}

	switch v := value.(type) {
	case string:
		val, ok := t.Value.Val.(string)
		return ok && val == v
	case bool:
		val, ok := t.Value.Val.(bool)
		return ok && val == v
	case int8, int32, int64, uint8, uint32, uint64:
		bigInt := big.NewInt(0)
		switch num := v.(type) {
		case int8:
			bigInt.SetInt64(int64(num))
		case uint8:
			bigInt.SetInt64(int64(num))
		case int32:
			bigInt.SetInt64(int64(num))
		case uint32:
			bigInt.SetInt64(int64(num))
		case int64:
			bigInt.SetInt64(num)
		case uint64:
			bigInt.SetInt64(int64(num))
		}
		if tval, ok := t.Value.Val.(*big.Int); ok {
			return tval.Cmp(bigInt) == 0
		}
		return false
	case *big.Int:
		if tval, ok := t.Value.Val.(*big.Int); ok {
			return tval.Cmp(v) == 0
		}
		return false
	case float32, float64:
		fval, ok := t.Value.Val.(float64)
		if !ok {
			return false
		}
		var targetVal float64
		if fv, isFloat32 := v.(float32); isFloat32 {
			targetVal = float64(fv)
		} else if fv, isFloat64 := v.(float64); isFloat64 {
			targetVal = fv
		}
		// Compare floats with small epsilon for precision
		return fval == targetVal
	case *Sym:
		val, ok := t.Value.Val.(*Sym)
		return ok && val.Name == v.Name
	}
	return false
}

// IsKeyword checks if this token is a specific keyword by name.
func (t *Token) IsKeyword(name string) bool {
	if t.Keyword == nil {
		return false
	}
	return t.Keyword.Sym.Name == name
}

// IsEof returns true if this is an EOF token.
func (t *Token) IsEof() bool {
	return t.Type == TokenTypeEof
}

// GetName returns the text representation of this token from the lexer's file.
func (t *Token) GetName() string {
	if t.Type == TokenTypeEof {
		return "EOF"
	}
	if t.Location.Len == 0 {
		return ""
	}
	endPos := t.Location.Pos + uint32(t.Location.Len)
	if endPos > uint32(len(t.Lexer.Filepath.Text)) {
		endPos = uint32(len(t.Lexer.Filepath.Text))
	}
	return t.Lexer.Filepath.Text[t.Location.Pos:endPos]
}

// Dump outputs debugging information about this token.
func (t *Token) Dump() {
	t.Location.Dump()
	fmt.Printf("Token: type=%v, name=%s\n", t.Type, t.GetName())
}
