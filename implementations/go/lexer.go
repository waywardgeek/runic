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

// Lexer tokenizes input from a Filepath.
type Lexer struct {
	Filepath              *Filepath
	Keytab                *Keytab
	peg                   *Peg  // Parent Peg parser (OneToOne cascade)
	Pos                   uint32
	Len                   uint32
	Line                  uint32
	AllowIdentUnderscores bool
	UseWeakStrings        bool
	StartPos              uint32
	Tokens                []*Token       // ArrayList relation
	ParseResults          []*ParseResult // DoublyLinked relation
}

// NewLexer creates a new Lexer for a file.
// If readFile is true, the file is read from disk first.
func NewLexer(filepath *Filepath, keytab *Keytab, readFile bool) (*Lexer, error) {
	if readFile {
		if err := filepath.ReadFile(); err != nil {
			return nil, err
		}
	}

	lexer := &Lexer{
		Filepath:              filepath,
		Keytab:                keytab,
		Pos:                   0,
		Len:                   uint32(len(filepath.Text)),
		Line:                  1,
		AllowIdentUnderscores: false,
		UseWeakStrings:        false,
		StartPos:              0,
		Tokens:                make([]*Token, 0),
		ParseResults:          make([]*ParseResult, 0),
	}
	filepath.AppendLexer(lexer)
	return lexer, nil
}

// AppendToken adds a token to this lexer's token list (ArrayList relation).
func (l *Lexer) AppendToken(token *Token) {
	l.Tokens = append(l.Tokens, token)
}

// Close is a cleanup method (for now, just a placeholder).
func (l *Lexer) Close() {
	// Cleanup if needed
}

// AppendParseResult adds a parse result (DoublyLinked relation).
// ParseResult is fully defined in parseresult.go
func (l *Lexer) AppendParseResult(pr *ParseResult) {
	if pr == nil {
		return
	}
	pr.lexer = l
	if l.ParseResults == nil {
		l.ParseResults = make([]*ParseResult, 0)
	}

	if len(l.ParseResults) == 0 {
		pr.prevLexerParseResult = nil
	} else {
		lastPR := l.ParseResults[len(l.ParseResults)-1]
		lastPR.nextLexerParseResult = pr
		pr.prevLexerParseResult = lastPR
	}
	l.ParseResults = append(l.ParseResults, pr)
}

// RemoveParseResult removes a parse result from this lexer.
func (l *Lexer) RemoveParseResult(pr *ParseResult) {
	if pr == nil {
		return
	}
	for i, p := range l.ParseResults {
		if p == pr {
			l.ParseResults = append(l.ParseResults[:i], l.ParseResults[i+1:]...)
			if i > 0 {
				l.ParseResults[i-1].nextLexerParseResult = nil
				if i < len(l.ParseResults) {
					l.ParseResults[i-1].nextLexerParseResult = l.ParseResults[i]
					l.ParseResults[i].prevLexerParseResult = l.ParseResults[i-1]
				}
			} else if i < len(l.ParseResults) {
				l.ParseResults[i].prevLexerParseResult = nil
			}
			pr.prevLexerParseResult = nil
			pr.nextLexerParseResult = nil
			pr.lexer = nil
			return
		}
	}
}

// ============================================================================
// TOKENIZATION METHODS
// ============================================================================

// ParseToken reads and returns the next token from input.
func (l *Lexer) ParseToken() (*Token, error) {
	if l.Eof() {
		return l.EofToken(), nil
	}

	// No further checks for eof are needed because the file always ends in a newline
	// (we add one if we detect it is missing when we read the file).
	l.skipSpace()
	l.StartPos = l.Pos
	char := l.readChar()
	if err := l.checkCharValid(char); err != nil {
		return nil, err
	}

	c := l.Filepath.Text[char.Pos]

	if c == '"' || (l.UseWeakStrings && c == '\'') {
		return l.parseString(c)
	} else if c == '\'' {
		return l.parseAsciiChar()
	} else if IsDigit(c) {
		return l.parseNumber()
	} else if c == '\\' {
		return l.parseEscapedIdent()
	}

	token, err := l.tryToParseUintIntOrRandType()
	if err != nil {
		return nil, err
	}
	if token != nil {
		return token, nil
	}

	if l.isValidIdentChar(char) {
		return l.readIdentOrKeyword()
	}

	return l.parseNonAlphaKeyword(char)
}

// Eof returns true if we've reached the end of input.
func (l *Lexer) Eof() bool {
	return l.Pos >= l.Len
}

// readChar reads one UTF-8 character and advances Pos.
func (l *Lexer) readChar() Char {
	char := GetChar(l.Filepath.Text, l.Pos)
	l.Pos += uint32(char.Len)
	return char
}

// checkCharValid returns an error if the character is invalid UTF-8.
func (l *Lexer) checkCharValid(char Char) error {
	if !char.Valid {
		return l.errorMsg("Invalid character")
	}
	return nil
}

// EofToken creates and returns an EOF token.
func (l *Lexer) EofToken() *Token {
	return NewToken(l, TokenTypeEof, NewLocation(l.Filepath, l.Len, 0, l.Line), nil, NewValue(nil))
}

// location returns a Location from StartPos to current Pos.
func (l *Lexer) location() Location {
	return NewLocation(l.Filepath, l.StartPos, l.Pos-l.StartPos, l.Line)
}

// errorMsg creates an error with current location.
func (l *Lexer) errorMsg(msg string) error {
	return l.location().Error(msg)
}

// ============================================================================
// WHITESPACE AND COMMENT HANDLING
// ============================================================================

// skipSpace skips whitespace and comments, but not newlines.
func (l *Lexer) skipSpace() {
	l.rawSkipSpace()
	for {
		skippedComment := false
		if l.inputHas("//") {
			l.skipSingleLineComment()
			l.rawSkipSpace()
			skippedComment = true
		} else if l.inputHas("/*") {
			l.skipBlockComment()
			l.rawSkipSpace()
			skippedComment = true
		}
		if !skippedComment {
			break
		}
	}
}

// rawSkipSpace skips just whitespace, not comments or newlines.
func (l *Lexer) rawSkipSpace() {
	for l.Pos < l.Len {
		c := l.Filepath.Text[l.Pos]
		if c == ' ' || c == '\r' || c == '\t' {
			l.Pos++
		} else {
			break
		}
	}
}

// skipSingleLineComment skips everything up to (but not including) the newline.
func (l *Lexer) skipSingleLineComment() {
	for l.Pos < l.Len {
		c := l.Filepath.Text[l.Pos]
		if c != '\n' {
			l.Pos++
		} else {
			break
		}
	}
}

// skipBlockComment skips nested block comments.
// They can be nested, so we maintain a depth counter.
func (l *Lexer) skipBlockComment() {
	depth := 1
	l.Pos += 2 // Skip the "/*"

	for l.Pos < l.Len && depth != 0 {
		if l.inputHas("/*") {
			depth++
			l.Pos += 2
		} else if l.inputHas("*/") {
			depth--
			l.Pos += 2
		} else {
			l.Pos++
		}
	}
}

// inputHas returns true if the input at current Pos starts with text.
func (l *Lexer) inputHas(text string) bool {
	if l.Pos+uint32(len(text)) > l.Len {
		return false
	}
	return text == l.Filepath.Text[l.Pos:l.Pos+uint32(len(text))]
}

// ============================================================================
// STRING AND CHARACTER PARSING
// ============================================================================

// isValidIdentChar returns true if char could start an identifier.
func (l *Lexer) isValidIdentChar(char Char) bool {
	c := l.Filepath.Text[char.Pos]
	return IsAsciiAlpha(l.Filepath.Text, char) || char.Len > 1 ||
		(l.AllowIdentUnderscores && (c == '_' || c == '$'))
}

// parseString parses a quoted string, handling escape sequences.
// target is the quote character (' or ")
func (l *Lexer) parseString(target uint8) (*Token, error) {
	s := ""

	for {
		if l.Eof() {
			return nil, l.errorMsg("End of file while reading string")
		}
		char := l.readChar()
		c := l.Filepath.Text[char.Pos]

		if c == target {
			break
		}
		if c == '\\' {
			escapedChar, err := l.readEscapedChar(target == '\'')
			if err != nil {
				return nil, err
			}
			s += string(escapedChar)
		} else {
			for i := char.Pos; i < char.Pos+uint32(char.Len); i++ {
				s += string(l.Filepath.Text[i])
			}
		}
	}

	token := NewValueToken(l, s, l.location())
	if l.UseWeakStrings && target == '\'' {
		token.Type = TokenTypeWeakString
	}
	return token, nil
}

// readEscapedChar reads the character after a backslash.
// singleQuotes indicates if we're in single quotes (for escape validation).
func (l *Lexer) readEscapedChar(singleQuotes bool) (uint8, error) {
	char := l.readChar()
	c := l.Filepath.Text[char.Pos]

	switch c {
	case 'a':
		return '\a', nil
	case 'b':
		return '\b', nil
	case 'e':
		return 27, nil // Escape character
	case 'f':
		return '\f', nil
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 't':
		return '\t', nil
	case 'v':
		return '\v', nil
	case '\\':
		return '\\', nil
	case '"':
		if !singleQuotes {
			return '"', nil
		}
	case '\'':
		if singleQuotes {
			return '\'', nil
		}
	case '0':
		return 0, nil
	case 'x':
		hi := l.readChar()
		lo := l.readChar()
		if !IsHexDigit(l.Filepath.Text[hi.Pos]) || !IsHexDigit(l.Filepath.Text[lo.Pos]) {
			return 0, l.errorMsg("Non-hex digit in hexadecimal escape sequence")
		}
		return HexToChar(l.Filepath.Text[hi.Pos], l.Filepath.Text[lo.Pos]), nil
	}

	return 0, l.errorMsg("Invalid escape sequence")
}

// parseAsciiChar returns a single-quoted character as a u8 integer token.
func (l *Lexer) parseAsciiChar() (*Token, error) {
	char := l.readChar()
	if err := l.checkCharValid(char); err != nil {
		return nil, err
	}

	if char.Len != 1 {
		return nil, l.errorMsg("Only single-byte characters can be used in single quotes")
	}

	c := l.Filepath.Text[char.Pos]
	if c == '\\' {
		escapedChar, err := l.readEscapedChar(true)
		if err != nil {
			return nil, err
		}
		c = escapedChar
	}

	if err := l.expectChar('\''); err != nil {
		return nil, err
	}
	return NewValueToken(l, uint8(c), l.location()), nil
}

// expectChar reads a character and returns an error if it doesn't match expected.
func (l *Lexer) expectChar(expectedChar uint8) error {
	char := l.readChar()
	c := l.Filepath.Text[char.Pos]
	if c != expectedChar {
		return l.errorMsg(fmt.Sprintf("Expected %s, got %s", string(expectedChar), string(c)))
	}
	return nil
}

// ============================================================================
// NUMBER PARSING
// ============================================================================

// parseNumber parses numeric literals (integers or floats).
func (l *Lexer) parseNumber() (*Token, error) {
	l.Pos-- // Rewind to start

	intVal := l.parseRawInteger()
	
	// Check if we're at end of file
	if l.Pos >= l.Len {
		return l.parseIntegerSuffix(intVal)
	}
	
	c := l.Filepath.Text[l.Pos]

	if c == '.' || c == 'f' || c == 'e' || c == 'E' {
		return l.parseFloat(intVal)
	}

	if c == 'x' && l.Pos == l.StartPos+1 && l.Filepath.Text[l.StartPos] == '0' {
		l.Pos++
		intVal = l.parseHexInteger()
	}

	return l.parseIntegerSuffix(intVal)
}

// parseRawInteger parses an integer without width spec.
// Returns a Bigint with minimum width to fit the value.
func (l *Lexer) parseRawInteger() *big.Int {
	intVal := big.NewInt(0)

	for l.Pos < l.Len {
		c := l.Filepath.Text[l.Pos]
		if IsDigit(c) || c == '_' {
			l.Pos++
			if c != '_' {
				intVal.Mul(intVal, big.NewInt(10))
				intVal.Add(intVal, big.NewInt(int64(c-'0')))
			}
		} else {
			break
		}
	}

	return intVal
}

// parseHexInteger parses hexadecimal digits into a Bigint.
func (l *Lexer) parseHexInteger() *big.Int {
	intVal := big.NewInt(0)

	for l.Pos < l.Len {
		c := l.Filepath.Text[l.Pos]
		if IsHexDigit(c) || c == '_' {
			if c != '_' {
				intVal.Lsh(intVal, 4)
				intVal.Or(intVal, big.NewInt(int64(HexDigit(c))))
			}
			l.Pos++
		} else {
			break
		}
	}

	return intVal
}

// parseIntegerSuffix handles integer width specifiers (e.g., u32, i64).
func (l *Lexer) parseIntegerSuffix(intVal *big.Int) (*Token, error) {
	// Check if we're at end of file
	if l.Pos >= l.Len {
		// No suffix, just return the integer
		return NewValueToken(l, intVal, l.location()), nil
	}

	c := l.Filepath.Text[l.Pos]

	if c == 'i' || c == 'u' {
		savedPos := l.Pos + 1 // Save position right after 'u' or 'i'
		l.Pos++
		width, err := l.parseWidthSpec()
		if err != nil {
			return nil, err
		}
		if width == 0 {
			// Width spec parsing failed, restore position and continue
			l.Pos = savedPos - 1 // Go back to the 'u' or 'i'
		}
	}

	// For now, just store the Bigint as-is; type checking happens later
	return NewValueToken(l, intVal, l.location()), nil
}

// parseWidthSpec parses a width specifier (e.g., the "32" in "u32").
// Returns the width, or 0 if not a valid width spec.
func (l *Lexer) parseWidthSpec() (uint32, error) {
	if l.Pos >= l.Len {
		return 0, nil
	}
	c := l.Filepath.Text[l.Pos]
	if c < '1' || c > '9' {
		return 0, nil
	}

	newWidth := l.parseRawInteger()
	if newWidth.Cmp(big.NewInt(0xffff)) > 0 {
		return 0, nil
	}

	if l.Pos >= l.Len {
		return uint32(newWidth.Int64()), nil
	}

	// Check if next character is alphanumeric (would indicate invalid width spec)
	char := GetChar(l.Filepath.Text, l.Pos)
	if l.isValidIdentChar(char) {
		// Next char is alphanumeric, width spec is invalid
		return 0, nil
	}
	// Successfully parsed width - Pos is already at the right place after parseRawInteger()
	// Don't consume the non-alphanumeric character

	return uint32(newWidth.Int64()), nil
}

// parseFloat parses floating point numbers.
func (l *Lexer) parseFloat(intVal *big.Int) (*Token, error) {
	fracVal := big.NewInt(0)
	width := uint32(64)
	exp := int32(0)
	fracDigits := uint32(0)

	c := l.Filepath.Text[l.Pos]

	if c == '.' {
		l.Pos++
		fracDigits = l.countDigits()
		fracVal = l.parseRawInteger()
		c = l.Filepath.Text[l.Pos]
	}

	if c == 'e' || c == 'E' {
		l.Pos++
		negateExp := false
		if l.Pos < l.Len && l.Filepath.Text[l.Pos] == '-' {
			l.Pos++
			negateExp = true
		}
		if l.Pos >= l.Len || !IsDigit(l.Filepath.Text[l.Pos]) {
			return nil, l.errorMsg("Missing exponent after 'e' in floating point number")
		}
		expVal := l.parseRawInteger()
		exp = int32(expVal.Int64())
		if negateExp {
			exp = -exp
		}
		c = l.Filepath.Text[l.Pos]
	}

	if c == 'f' {
		l.Pos++
		widthVal := l.parseRawInteger()
		width = uint32(widthVal.Int64())
		if width != 32 && width != 64 {
			return nil, l.errorMsg("Only 32 and 64 bit floating point numbers are currently supported.")
		}
	}

	return l.buildFloatToken(intVal, fracVal, fracDigits, exp, width), nil
}

// buildFloatToken constructs a float token from components.
func (l *Lexer) buildFloatToken(intVal, fracVal *big.Int, fracDigits uint32, exp int32, width uint32) *Token {
	intFloat := float64(intVal.Int64())
	fracFloat := float64(fracVal.Int64())

	val := intFloat + fracFloat/pow(10.0, float64(fracDigits))
	val *= pow(10.0, float64(exp))

	return NewValueToken(l, val, l.location())
}

// pow computes base^exp for floats.
func pow(base, expVal float64) float64 {
	result := 1.0
	invert := expVal < 0
	exp := int64(expVal)
	if invert {
		exp = -exp
	}
	for i := int64(0); i < exp; i++ {
		result *= base
	}
	if invert {
		return 1.0 / result
	}
	return result
}

// countDigits counts consecutive digits starting at current Pos.
func (l *Lexer) countDigits() uint32 {
	numDigits := uint32(0)
	for l.Pos+numDigits < l.Len && IsDigit(l.Filepath.Text[l.Pos+numDigits]) {
		numDigits++
	}
	return numDigits
}

// ============================================================================
// IDENTIFIER PARSING
// ============================================================================

// parseEscapedIdent parses an identifier starting with backslash.
func (l *Lexer) parseEscapedIdent() (*Token, error) {
	l.StartPos = l.Pos // Don't include the backslash in the name
	for l.Pos < l.Len {
		char := l.readChar()
		c := l.Filepath.Text[char.Pos]
		if IsWhitespace(c) || c == '\n' {
			l.Pos = char.Pos // Position of the whitespace
			break
		}
	}
	name := l.Filepath.Text[l.StartPos:l.Pos]
	return NewValueToken(l, NewSym(name), l.location()), nil
}

// tryToParseUintIntOrRandType tries to parse tokens like u32, i64, rand256.
func (l *Lexer) tryToParseUintIntOrRandType() (*Token, error) {
	pos := l.Pos
	var tokenType TokenType

	if l.tokenStartsWith("rand") {
		l.Pos += 3
		tokenType = TokenTypeRandUint
	} else if l.tokenStartsWith("i") {
		tokenType = TokenTypeIntType
	} else if l.tokenStartsWith("u") {
		tokenType = TokenTypeUintType
	} else {
		return nil, nil
	}

	width, err := l.parseWidthSpec()
	if err != nil {
		return nil, err
	}
	if width == 0 {
		l.Pos = pos
		return nil, nil
	}

	token := NewToken(l, tokenType, l.location(), nil, NewValue(big.NewInt(int64(width))))
	return token, nil
}

// tokenStartsWith returns true if text starting at StartPos matches text.
func (l *Lexer) tokenStartsWith(text string) bool {
	if l.StartPos+uint32(len(text)) > l.Len {
		return false
	}
	return text == l.Filepath.Text[l.StartPos:l.StartPos+uint32(len(text))]
}

// readIdentOrKeyword parses an identifier or keyword.
func (l *Lexer) readIdentOrKeyword() (*Token, error) {
	for l.Pos < l.Len {
		char := l.readChar()
		c := l.Filepath.Text[char.Pos]
		if !(IsAsciiAlpha(l.Filepath.Text, char) || char.Len > 1 || IsDigit(c) ||
			(l.AllowIdentUnderscores && (c == '_' || c == '$'))) {
			l.Pos = char.Pos // Push back the next character
			break
		}
	}

	name := l.Filepath.Text[l.StartPos:l.Pos]
	keyword := l.Keytab.Lookup(name)

	if keyword != nil {
		return NewToken(l, TokenTypeKeyword, l.location(), keyword, NewValue(nil)), nil
	}

	return NewValueToken(l, NewSym(name), l.location()), nil
}

// parseNonAlphaKeyword tries to parse operators and punctuation (up to 4 characters).
func (l *Lexer) parseNonAlphaKeyword(char Char) (*Token, error) {
	for _, i := range []int{4, 3, 2, 1} {
		l.Pos = l.StartPos
		keyword := l.tryNonAlphaKeyword(uint64(i))
		if keyword != nil {
			// Check if it's a newline
			if i == 1 && keyword.Sym.Name == "\n" {
				l.Line++
			}
			return NewToken(l, TokenTypeKeyword, l.location(), keyword, NewValue(nil)), nil
		}
	}

	return nil, l.errorMsg("Parser error: keyword not found")
}

// tryNonAlphaKeyword tries to parse a keyword of exactly len characters.
func (l *Lexer) tryNonAlphaKeyword(len uint64) *Keyword {
	for i := uint64(0); i < len; i++ {
		l.readChar()
	}
	text := l.Filepath.Text[l.StartPos:l.Pos]
	return l.Keytab.Lookup(text)
}

// ============================================================================
// CONFIGURATION
// ============================================================================

// EnableIdentUnderscores enables underscores and dollar signs in identifiers.
func (l *Lexer) EnableIdentUnderscores(value bool) {
	l.AllowIdentUnderscores = value
}

// EnableWeakStrings enables single-quoted strings as weak strings.
func (l *Lexer) EnableWeakStrings(value bool) {
	l.UseWeakStrings = value
}
