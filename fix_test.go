package main

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

var only = flag.String("only", "", "If non-empty, the fix test to run")

var tests = []struct {
	name    string
	in, out string
}{
	// Adding an import to an existing parenthesized import
	{
		name: "factored_imports_add",
		in: `package foo
import (
  "fmt"
)
func bar() {
var b bytes.Buffer
fmt.Println(b.String())
}
`,
		out: `package foo

import (
	"bytes"
	"fmt"
)

func bar() {
	var b bytes.Buffer
	fmt.Println(b.String())
}
`,
	},

	// Adding a new import line, without parens
	{
		name: "add_import_section",
		in: `package foo
func bar() {
var b bytes.Buffer
}
`,
		out: `package foo

import "bytes"

func bar() {
	var b bytes.Buffer
}
`,
	},

	// Adding two new imports, which should make a parenthesized import decl.
	{
		name: "add_import_paren_section",
		in: `package foo
func bar() {
_, _ := bytes.Buffer, zip.NewReader
}
`,
		out: `package foo

import (
	"archive/zip"
	"bytes"
)

func bar() {
	_, _ := bytes.Buffer, zip.NewReader
}
`,
	},

	// Make sure we don't add things twice
	{
		name: "no_double_add",
		in: `package foo
func bar() {
_, _ := bytes.Buffer, bytes.NewReader
}
`,
		out: `package foo

import "bytes"

func bar() {
	_, _ := bytes.Buffer, bytes.NewReader
}
`,
	},

	// Remove unused imports, 1 of a factored block
	{
		name: "remove_unused_1_of_2",
		in: `package foo
import (
"bytes"
"fmt"
)

func bar() {
_, _ := bytes.Buffer, bytes.NewReader
}
`,
		out: `package foo

import "bytes"

func bar() {
	_, _ := bytes.Buffer, bytes.NewReader
}
`,
	},

	// Remove unused imports, 2 of 2
	{
		name: "remove_unused_2_of_2",
		in: `package foo
import (
"bytes"
"fmt"
)

func bar() {
}
`,
		out: `package foo

func bar() {
}
`,
	},

	// Remove unused imports, 1 of 1
	{
		name: "remove_unused_1_of_1",
		in: `package foo

import "fmt"

func bar() {
}
`,
		out: `package foo

func bar() {
}
`,
	},

	// Add to first import group
	{
		name: "first_group",
		in: `package foo

import (
	"fmt"

	"bytes"
)

func bar() {
	_, _, _ = fmt.Println, bytes.Buffer, math.Pow
}
`,
		out: `package foo

import (
	"math"
	"fmt"

	"bytes"
)

func bar() {
	_, _, _ = fmt.Println, bytes.Buffer, math.Pow
}
`,
	},
}

func TestFixImports(t *testing.T) {
	for _, tt := range tests {
		if *only != "" && tt.name != *only {
			continue
		}
		var buf bytes.Buffer
		err := processFile("foo.go", strings.NewReader(tt.in), &buf, false)
		if err != nil {
			t.Errorf("error on %q: %v", tt.name, err)
			continue
		}
		if got := buf.String(); got != tt.out {
			t.Errorf("results diff on %q\nGOT:\n%s\nWANT:\n%s\n", tt.name, got, tt.out)
		}
	}
}
