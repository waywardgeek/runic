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

package main

import (
	"flag"
	"fmt"
	"os"
	parser "rune-go-parser"
)

func main() {
	// Define flags
	noSimplify := flag.Bool("no-simplify", false, "Disable node tree simplification (show full parse tree)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--no-simplify] <grammar.syn> <input.rn>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Parses input.rn using grammar.syn and dumps the Node tree\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	grammarFile := args[0]
	inputFile := args[1]

	// Parse the grammar
	fmt.Printf("Loading grammar from %s...\n", grammarFile)
	peg, err := parseGrammar(grammarFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing grammar: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Grammar loaded: %d rules\n\n", len(peg.OrderedRules()))

	// Parse the input file
	fmt.Printf("Parsing input file %s...\n", inputFile)
	peg.SetSimplifyNodes(!*noSimplify)
	node, err := peg.Parse(inputFile, false) // allowUnderscores=false
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing input: %v\n", err)
		os.Exit(1)
	}

	if node == nil {
		fmt.Fprintf(os.Stderr, "Parse failed: no node returned\n")
		os.Exit(1)
	}

	fmt.Printf("✅ Parse successful!\n\n")
	if *noSimplify {
		fmt.Println("Parse Tree (unsimplified):")
	} else {
		fmt.Println("Parse Tree (simplified):")
	}
	fmt.Println("===========")
	node.Dump()
}

// parseGrammar loads and parses a .syn grammar file
func parseGrammar(filename string) (*parser.Peg, error) {
	// NewPeg automatically reads and parses the grammar file
	peg, err := parser.NewPeg(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create Peg: %w", err)
	}

	return peg, nil
}
