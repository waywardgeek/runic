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
	"io/ioutil"
)

// Filepath represents a source file path and its contents.
type Filepath struct {
	Name   string
	Parent *Filepath
	Text   string
	IsDir  bool
	Lexers []*Lexer // ArrayList relation
}

// NewFilepath creates a new Filepath.
// If readFile is true, the file contents are read from disk.
func NewFilepath(name string, parent *Filepath, isDir bool) *Filepath {
	fp := &Filepath{
		Name:   name,
		Parent: parent,
		IsDir:  isDir,
		Text:   "",
		Lexers: make([]*Lexer, 0),
	}
	return fp
}

// ReadFile reads the file contents from disk.
// If the file doesn't exist, this returns an error.
func (fp *Filepath) ReadFile() error {
	data, err := ioutil.ReadFile(fp.Name)
	if err != nil {
		return err
	}
	text := string(data)
	// Ensure the file ends with a newline
	if len(text) == 0 || text[len(text)-1] != '\n' {
		text += "\n"
	}
	fp.Text = text
	return nil
}

// AppendLexer adds a lexer to this file (ArrayList relation).
func (fp *Filepath) AppendLexer(lexer *Lexer) {
	fp.Lexers = append(fp.Lexers, lexer)
}

// Lexers returns all lexers for this file.
func (fp *Filepath) GetLexers() []*Lexer {
	return fp.Lexers
}

// Location represents a position in source code.
type Location struct {
	Filepath *Filepath
	Pos      uint32 // Character position in file
	Len      uint32 // Length in characters (for error messages)
	Line     uint32 // Line number (1-indexed)
}

// NewLocation creates a new Location.
func NewLocation(filepath *Filepath, pos, len, line uint32) Location {
	return Location{
		Filepath: filepath,
		Pos:      pos,
		Len:      len,
		Line:     line,
	}
}

// EmptyLocation returns a location with no source info.
func EmptyLocation() Location {
	return Location{
		Filepath: nil,
		Pos:      0,
		Len:      0,
		Line:     0,
	}
}

// Dump outputs debugging information about this location.
func (l Location) Dump() {
	if l.Filepath == nil {
		fmt.Println("Location: <empty>")
		return
	}
	fmt.Printf("%s:%d\n", l.Filepath.Name, l.Line)
}

// Error reports an error at this location and returns it.
func (l Location) Error(msg string) error {
	if l.Filepath != nil {
		return fmt.Errorf("%s:%d: %s", l.Filepath.Name, l.Line, msg)
	}
	return fmt.Errorf("error: %s", msg)
}
