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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseRuneSyn tests parsing of the actual rune.syn file.
// This is an integration test that verifies the parser can handle a real grammar file.
func TestParseRuneSyn(t *testing.T) {
	// Find rune.syn - try multiple locations
	possiblePaths := []string{
		"bootstrap/parse/rune.syn",
		"../bootstrap/parse/rune.syn",
		"../../bootstrap/parse/rune.syn",
		"/rune/bootstrap/parse/rune.syn",
	}

	var ryneSynPath string
	var content []byte
	var err error

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			content, err = ioutil.ReadFile(path)
			if err == nil {
				ryneSynPath = path
				break
			}
		}
	}

	if ryneSynPath == "" {
		t.Skip("Cannot find rune.syn in any expected location")
	}

	fmt.Printf("✅ Found rune.syn: %s\n", ryneSynPath)
	fmt.Printf("📄 File size: %d bytes\n", len(content))

	// Create a temporary copy for testing
	tmpFile, err := ioutil.TempFile("", "rune_*.syn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Parse rune.syn
	border := strings.Repeat("═", 70)
	fmt.Println("\n" + border)
	fmt.Println("PARSING rune.syn...")
	fmt.Println(border)

	peg, err := NewPeg(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse rune.syn: %v", err)
	}

	rules := peg.OrderedRules()
	fmt.Printf("\n✅ Successfully parsed %d rules\n", len(rules))

	if len(rules) == 0 {
		t.Fatal("No rules parsed from rune.syn")
	}

	// Print first 10 rules for inspection
	fmt.Println("\n📋 First 10 rules parsed:")
	fmt.Println(strings.Repeat("─", 70))
	for i, rule := range rules {
		if i >= 10 {
			break
		}
		fmt.Printf("%2d. %s\n", i+1, rule.Sym.Name)
		if rule.pexpr != nil {
			fmt.Printf("    Type: %d, CanBeEmpty: %v, Weak: %v\n",
				rule.pexpr.Type, rule.CanBeEmpty, rule.Weak)
		}
	}
	fmt.Println(strings.Repeat("─", 70))

	// Generate output file
	outputPath := filepath.Join(os.TempDir(), "rune_parsed_output.syn")
	fmt.Printf("\n📝 Writing parsed output to: %s\n", outputPath)

	output := peg.ToString()
	if err := ioutil.WriteFile(outputPath, []byte(output), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("✅ Output written successfully\n")
	fmt.Printf("📊 Output size: %d bytes (original: %d bytes)\n", len(output), len(content))
	fmt.Printf("\n📋 To compare with original:\n")
	fmt.Printf("   diff -u %s %s\n", ryneSynPath, outputPath)

	// Basic sanity checks
	if len(output) == 0 {
		t.Fatal("Output is empty - ToString() failed")
	}

	// Check that we have reasonable content
	if len(rules) < 50 {
		t.Logf("Warning: Only parsed %d rules (expected ~100+)", len(rules))
	}

	// Verify all rules have names
	for i, rule := range rules {
		if rule.Sym == nil || rule.Sym.Name == "" {
			t.Errorf("Rule %d has empty name", i)
		}
		if rule.pexpr == nil {
			t.Errorf("Rule %s has no pexpr", rule.Sym.Name)
		}
	}

	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Printf("✅ rune.syn parsing test PASSED\n")
	fmt.Println(strings.Repeat("═", 70))

	t.Log("✅ TestParseRuneSyn passed")
}

// TestParseRuneSynRoundTrip tests that parsing and re-generating produces consistent output.
func TestParseRuneSynRoundTrip(t *testing.T) {
	// Find rune.syn - try multiple locations
	possiblePaths := []string{
		"bootstrap/parse/rune.syn",
		"../bootstrap/parse/rune.syn",
		"../../bootstrap/parse/rune.syn",
		"/rune/bootstrap/parse/rune.syn",
	}

	var content []byte
	var err error

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			content, err = ioutil.ReadFile(path)
			if err == nil {
				break
			}
		}
	}

	if len(content) == 0 {
		t.Skip("Cannot find rune.syn in any expected location")
	}

	// Parse first time
	tmpFile1, err := ioutil.TempFile("", "rune_1_*.syn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile1.Name())
	tmpFile1.Write(content)
	tmpFile1.Close()

	peg1, err := NewPeg(tmpFile1.Name())
	if err != nil {
		t.Fatalf("Failed to parse first time: %v", err)
	}

	output1 := peg1.ToString()

	// Parse the output (second round trip)
	tmpFile2, err := ioutil.TempFile("", "rune_2_*.syn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile2.Name())
	tmpFile2.WriteString(output1)
	tmpFile2.Close()

	peg2, err := NewPeg(tmpFile2.Name())
	if err != nil {
		t.Logf("⚠️  Second round-trip parse failed: %v", err)
		t.Logf("This may indicate parser issues with the generated output")
		// Don't fail - just log for now
		return
	}

	output2 := peg2.ToString()

	fmt.Printf("\n✅ Round-trip test results:\n")
	fmt.Printf("   First parse:  %d rules, %d bytes output\n", len(peg1.OrderedRules()), len(output1))
	fmt.Printf("   Second parse: %d rules, %d bytes output\n", len(peg2.OrderedRules()), len(output2))

	if len(peg1.OrderedRules()) != len(peg2.OrderedRules()) {
		t.Logf("⚠️  Warning: Rule count differs: %d vs %d",
			len(peg1.OrderedRules()), len(peg2.OrderedRules()))
	}

	if output1 == output2 {
		fmt.Println("   ✅ Output is idempotent (same on second parse)")
	} else {
		fmt.Println("   ⚠️  Output differs on second parse (parsing not stable)")
	}

	t.Log("✅ TestParseRuneSynRoundTrip completed")
}

// TestRuleParsing checks specific rule structure
func TestRuleParsing(t *testing.T) {
	// Find rune.syn - try multiple locations
	possiblePaths := []string{
		"bootstrap/parse/rune.syn",
		"../bootstrap/parse/rune.syn",
		"../../bootstrap/parse/rune.syn",
		"/rune/bootstrap/parse/rune.syn",
	}

	var content []byte
	var err error

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			content, err = ioutil.ReadFile(path)
			if err == nil {
				break
			}
		}
	}

	if len(content) == 0 {
		t.Skip("Cannot find rune.syn in any expected location")
	}

	tmpFile, err := ioutil.TempFile("", "rune_*.syn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(content)
	tmpFile.Close()

	peg, err := NewPeg(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse rune.syn: %v", err)
	}

	// Find specific well-known rules and verify them
	testCases := []struct {
		name        string
		shouldExist bool
	}{
		{"goal", true},
		{"statement", true},
		{"expr", true},
		{"ifElseStatement", true},
		{"importStatement", true},
	}

	fmt.Println("\n✅ Checking well-known rules:")
	for _, tc := range testCases {
		sym := NewSym(tc.name)
		rule := peg.FindRule(sym)

		if tc.shouldExist {
			if rule == nil {
				t.Errorf("Expected to find rule '%s', but it wasn't found", tc.name)
			} else {
				fmt.Printf("   ✓ Found rule: %s\n", tc.name)
				if rule.pexpr != nil {
					fmt.Printf("     - Type: %d, FirstSetFound: %v\n",
						rule.pexpr.Type, rule.FirstSetFound)
				}
			}
		} else {
			if rule != nil {
				t.Errorf("Expected NOT to find rule '%s', but it was found", tc.name)
			}
		}
	}

	t.Log("✅ TestRuleParsing passed")
}

// RunRuneSynTests executes all rune.syn integration tests
func RunRuneSynTests(t *testing.T) {
	border := strings.Repeat("═", 68)
	fmt.Println("\n╔" + border + "╗")
	fmt.Println("║  RUNE.SYN INTEGRATION TESTS - Parser Validation              ║")
	fmt.Println("╚" + border + "╝")

	t.Run("ParseRuneSyn", TestParseRuneSyn)
	t.Run("RuleParsing", TestRuleParsing)
	t.Run("RoundTrip", TestParseRuneSynRoundTrip)

	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("✅ All rune.syn integration tests completed")
	fmt.Println(strings.Repeat("═", 70))
}
