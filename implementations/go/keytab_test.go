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

func TestKeytabConstructor(t *testing.T) {
	keytab := NewKeytab()
	
	// Create keywords
	kw1 := keytab.New(":")
	kw2 := keytab.New("|")
	kw3 := keytab.New(";")
	
	// Test that keywords were created
	if kw1 == nil {
		t.Errorf("Failed to create keyword ':'")
	}
	if kw2 == nil {
		t.Errorf("Failed to create keyword '|'")
	}
	if kw3 == nil {
		t.Errorf("Failed to create keyword ';'")
	}
	
	// Test lookup of non-existent keyword
	if keytab.Lookup("\n") != nil {
		t.Errorf("Lookup of newline should return nil, got non-nil")
	}
	
	// Test lookup of existing keywords
	if keytab.Lookup(":") == nil {
		t.Errorf("Lookup of ':' should not return nil")
	}
	if keytab.Lookup("|") == nil {
		t.Errorf("Lookup of '|' should not return nil")
	}
	if keytab.Lookup(";") == nil {
		t.Errorf("Lookup of ';' should not return nil")
	}
	
	// Verify the returned keywords are the same ones we created
	if keytab.Lookup(":") != kw1 {
		t.Errorf("Lookup(':') should return the same keyword object")
	}
	if keytab.Lookup("|") != kw2 {
		t.Errorf("Lookup('|') should return the same keyword object")
	}
	if keytab.Lookup(";") != kw3 {
		t.Errorf("Lookup(';') should return the same keyword object")
	}
}

func TestKeytabDuplicate(t *testing.T) {
	keytab := NewKeytab()
	
	// Create keyword "test"
	kw1 := keytab.New("test")
	
	// Try to create it again - should return the same object
	kw2 := keytab.New("test")
	
	if kw1 != kw2 {
		t.Errorf("Creating duplicate keyword should return same object")
	}
	
	if kw1.Sym.Name != "test" {
		t.Errorf("Keyword name should be 'test', got '%s'", kw1.Sym.Name)
	}
}

func TestSetKeywordNums(t *testing.T) {
	keytab := NewKeytab()
	
	kw1 := keytab.New("first")
	kw2 := keytab.New("second")
	kw3 := keytab.New("third")
	
	// Before SetKeywordNums, all should be 0
	if kw1.Num != 0 || kw2.Num != 0 || kw3.Num != 0 {
		t.Errorf("Before SetKeywordNums, all Num should be 0")
	}
	
	// After SetKeywordNums, check the count
	count := keytab.SetKeywordNums()
	
	if count != 3 {
		t.Errorf("SetKeywordNums should return 3, got %d", count)
	}
	
	// Each keyword should have a unique Num between 0 and count-1
	nums := make(map[uint32]bool)
	for _, kw := range []*Keyword{kw1, kw2, kw3} {
		if kw.Num >= count {
			t.Errorf("Keyword Num %d should be < %d", kw.Num, count)
		}
		if nums[kw.Num] {
			t.Errorf("Keyword Num %d was assigned twice", kw.Num)
		}
		nums[kw.Num] = true
	}
}
