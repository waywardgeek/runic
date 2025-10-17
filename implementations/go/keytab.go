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

// Sym represents a symbol (interned string).
type Sym struct {
	Name string
}

// NewSym creates or returns a cached Sym for the given name.
var symCache = make(map[string]*Sym)

// NewSym creates a new Sym with the given name.
// Symbols are interned, so multiple calls with the same name return the same *Sym.
func NewSym(name string) *Sym {
	if s, exists := symCache[name]; exists {
		return s
	}
	s := &Sym{Name: name}
	symCache[name] = s
	return s
}

// Keyword represents a keyword token with an optional numeric ID.
type Keyword struct {
	Sym           *Sym
	Num           uint32
	Tokens        []*Token  // DoublyLinked Keyword Token (not used in PEG)
	firstPexpr    *Pexpr    // TailLinked Keyword Pexpr cascade
	lastPexpr     *Pexpr
}

// Keytab is a symbol-based hash table for keywords.
type Keytab struct {
	Keywords map[string]*Keyword // Hashed by Sym.Name
}

// NewKeytab creates a new empty keyword table.
func NewKeytab() *Keytab {
	return &Keytab{
		Keywords: make(map[string]*Keyword),
	}
}

// Lookup returns the keyword with the given name, or nil if not found.
func (kt *Keytab) Lookup(name string) *Keyword {
	return kt.Keywords[name]
}

// New gets or creates a keyword with the given name.
// If the keyword already exists, it is returned. Otherwise a new one is created.
func (kt *Keytab) New(name string) *Keyword {
	if kw, exists := kt.Keywords[name]; exists {
		return kw
	}
	kw := &Keyword{
		Sym:    NewSym(name),
		Num:    0,
		Tokens: make([]*Token, 0),
	}
	kt.Keywords[name] = kw
	return kw
}

// InsertKeyword adds a keyword to this keytab.
func (kt *Keytab) InsertKeyword(kw *Keyword) {
	kt.Keywords[kw.Sym.Name] = kw
}

// FindKeyword finds a keyword by Sym.
func (kt *Keytab) FindKeyword(sym *Sym) *Keyword {
	return kt.Keywords[sym.Name]
}

// SetKeywordNums assigns numeric IDs to all keywords in order.
// Returns the total number of keywords.
func (kt *Keytab) SetKeywordNums() uint32 {
	num := uint32(0)
	for _, kw := range kt.Keywords {
		kw.Num = num
		num++
	}
	return num
}

// NewKeyword creates a new keyword in the given keytab.
func NewKeyword(kt *Keytab, name string) *Keyword {
	return kt.New(name)
}

// ============================================================================
// TailLinked Keyword Pexpr cascade
// ============================================================================

// AppendPexpr adds a Pexpr to this keyword's list.
func (kw *Keyword) AppendPexpr(pexpr *Pexpr) {
	if pexpr == nil {
		return
	}

	if kw.lastPexpr == nil {
		kw.firstPexpr = pexpr
	} else {
		kw.lastPexpr.nextKeywordPexpr = pexpr
	}
	kw.lastPexpr = pexpr
	pexpr.Keyword = kw
	pexpr.nextKeywordPexpr = nil
}

// Pexprs returns a slice of all Pexprs for this keyword.
func (kw *Keyword) Pexprs() []*Pexpr {
	var pexprs []*Pexpr
	for p := kw.firstPexpr; p != nil; p = p.nextKeywordPexpr {
		pexprs = append(pexprs, p)
	}
	return pexprs
}

