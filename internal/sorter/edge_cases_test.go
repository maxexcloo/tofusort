package sorter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/maxexcloo/tofusort/internal/parser"
	"github.com/zclconf/go-cty/cty"
)

func verifyPreservation(t testing.TB, input []byte) []byte {
	t.Helper()
	p := parser.New()
	file, err := p.ParseFile(input)
	if err != nil {
		t.Fatal(err)
	}
	before := bodySignature(file.Body, true)
	comments := commentSignature(file)
	if err := New().SortFile(file); err != nil {
		t.Fatal(err)
	}
	output := p.FormatFile(file)
	if got := bodySignature(file.Body, true); got != before {
		t.Fatalf("configuration meaning changed\nbefore: %s\nafter: %s", before, got)
	}
	if got := commentSignature(file); got != comments {
		t.Fatalf("comments changed: %s != %s", got, comments)
	}
	if err := New().SortFile(file); err != nil {
		t.Fatal(err)
	}
	if again := p.FormatFile(file); !bytes.Equal(again, output) {
		t.Fatalf("not idempotent:\n%s\nthen:\n%s", output, again)
	}
	return output
}

// Compare parsed structure independently of the sorter's source-edit machinery.
// Only attributes, unique constant object keys and recognised root blocks are unordered.
func bodySignature(body *hclsyntax.Body, root bool) string {
	var attrs, blocks []string
	for name, attr := range body.Attributes {
		attrs = append(attrs, name+"="+signature(reflect.ValueOf(attr.Expr)))
	}
	sort.Strings(attrs)
	var run []string
	flush := func() { sort.Strings(run); blocks = append(blocks, run...); run = nil }
	for _, block := range body.Blocks {
		sig := block.Type + fmt.Sprint(block.Labels) + bodySignature(block.Body, false)
		if _, known := blockTypeOrder[block.Type]; root && known {
			run = append(run, sig)
		} else {
			flush()
			blocks = append(blocks, sig)
		}
	}
	flush()
	return fmt.Sprint(attrs, blocks)
}

func signature(v reflect.Value) string {
	if !v.IsValid() {
		return "nil"
	}
	if v.CanInterface() {
		switch value := v.Interface().(type) {
		case hcl.Range, hcl.Pos:
			return ""
		case cty.Value:
			return value.GoString()
		case cty.Type:
			return value.GoString()
		case *hclsyntax.Operation:
			return fmt.Sprintf("%p", value)
		case *hclsyntax.ObjectConsExpr:
			var items []string
			keys := make(map[string]bool)
			unordered := true
			for _, item := range value.Items {
				key, diags := item.KeyExpr.Value(nil)
				if diags.HasErrors() || !key.IsKnown() || key.IsNull() || key.Type() != cty.String {
					unordered = false
				} else {
					if keys[key.AsString()] {
						unordered = false
					}
					keys[key.AsString()] = true
				}
				items = append(items, signature(reflect.ValueOf(item.KeyExpr))+":"+signature(reflect.ValueOf(item.ValueExpr)))
			}
			if unordered {
				sort.Strings(items)
			}
			return "object" + fmt.Sprint(items)
		}
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return "nil"
		}
		return v.Type().String() + signature(v.Elem())
	case reflect.Struct:
		var fields []string
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				fields = append(fields, v.Type().Field(i).Name+"="+signature(v.Field(i)))
			}
		}
		return v.Type().String() + fmt.Sprint(fields)
	case reflect.Slice, reflect.Array:
		var values []string
		for i := 0; i < v.Len(); i++ {
			values = append(values, signature(v.Index(i)))
		}
		return fmt.Sprint(values)
	default:
		return fmt.Sprint(v.Interface())
	}
}

func commentSignature(file *parser.File) string {
	var comments []string
	for _, token := range file.Tokens {
		if token.Type == hclsyntax.TokenComment {
			comments = append(comments, strings.TrimSuffix(string(token.Bytes), "\n"))
		}
	}
	sort.Strings(comments)
	return fmt.Sprint(comments)
}

func TestEdgeCases(t *testing.T) {
	cases := []string{
		"",
		"# Only a comment\n",
		"z = 1 # last\na = 2\n",
		"z = 1\na = 2 # last\n",
		"# Z note\nz = 1\n# A note\na = 2\n",
		"# Header\n\nz = 1\na = 2\n\n# Footer\n",
		"z = 1\n\n# Section\n\ny = 2\nx = 3\n",
		"x = { z = 1, a = 2 }\n",
		"x = { z = 1, a = 2, }\n",
		"x = {\nz = 1 # Z\na = 2 # A\n}\n",
		"x = {\nz = 1, # Z\na = 2, # A\n}\n",
		"x = { z = 1 /* Z */, a = 2 /* A */ }\n",
		"x = { z = 1, /* Z */ a = 2 }\n",
		"x = { a = 1, /* A */ z = 2 }\n",
		"x = { /* Heading */\nz = 1\na = 2\n}\n",
		"x = { z = 1, a = 2 /* footer\nmultiline */ }\n",
		"x = { z = <<EOT\nhello\n\n\nEOT\na = 1\n}\n",
		"x = { z = <<-EOT\n  hello\n  EOT\na = 1\n}\n",
		"x = { z = { b=1, a=2 }, z = 3 }\n",
		"x = { (var.key) = 1, a = { z=1, a=2 } }\n",
		"x = { \"z.key\" = var.z, \"a.key\" = var.a }\n",
		"x = { é = 1, a = 2, _first = 3 }\n",
		"x = merge({ z=1, a=2 }, { z=3 })\n",
		"x = jsonencode({ z = [3,2,1], a = [{ z=1, a=2 }] })\n",
		"x = [for x in var.x : { z=x.z, a=x.a } if x.enabled]\n",
		"x = { for k, v in var.x : k => { z=v.z, a=v.a }... }\n",
		"x = var.enabled ? { z=1, a=2 } : { z=3, a=4 }\n",
		"x = [merge({ z=1,a=2 }, var.x...), -1, (2+3), true, null]\n",
		"x = \"${jsonencode({ z = 1, a = 2 })}\"\n",
		"x = \"%{ for x in var.x }${x}%{ endfor }\"\n",
		"variable \"x\" { type = list(object({ z=optional(string), a=number })) }\n",
		"resource \"x\" \"y\" {\nz=1\na=2\nlifecycle { prevent_destroy=true }\nprovisioner \"local-exec\" { command=\"first\" }\nprovisioner \"local-exec\" { command=\"second\" }\n}\n",
		"run \"z\" { command=apply }\nrun \"a\" { command=plan }\n",
		"resource \"z\" \"x\" {}\nunknown {}\nresource \"a\" \"x\" {}\n",
		"z=1\r\na=2\r\n",
	}
	for i, input := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) { verifyPreservation(t, []byte(input)) })
	}
}

func TestExternalCorpus(t *testing.T) {
	root := os.Getenv("TOFUSORT_CORPUS")
	if root == "" {
		t.Skip("set TOFUSORT_CORPUS to test a local HCL directory without modifying it")
	}
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".tf" && filepath.Ext(path) != ".tfvars" {
			return nil
		}
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		count++
		t.Run(filepath.Base(path), func(t *testing.T) { verifyPreservation(t, input) })
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("corpus contained no HCL files")
	}
	t.Logf("verified %d files without modifying the corpus", count)
}

func FuzzSortPreservesStructure(f *testing.F) {
	for _, seed := range []string{"#", "x=0 # preserve trailing spaces  ", `x = merge({ z = 1, a = 2 }, var.x)`, `variable "schema" { type = list(object({ z = string, a = number })) }`, "z=1\na=2\n", "x={z=1,a=2}\n", "x={z=1,a=2} # note\n", "x=[{z=1,a=2},3]\n", "x=<<EOT\nhello\nEOT\n"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 100000 {
			t.Skip()
		}
		if _, err := parser.New().ParseFile([]byte(src)); err != nil {
			t.Skip()
		}
		verifyPreservation(t, []byte(src))
	})
}

func TestSortingWithCommentsAndExpressions(t *testing.T) {
	input := `# File header

locals {
  # Z belongs here.
  z = 1 # Z inline
  sequence = [{ z = 1, a = 2 }, { z = 3, a = 4 }]
  a = 2

  mapping = merge(
    var.defaults,
    {
      "z.key" = var.z
      # A belongs here.
      "a.key" = var.a
    }
  )
}

# File footer
`
	expected := `# File header

locals {
  a        = 2
  sequence = [{ a = 2, z = 1 }, { a = 4, z = 3 }]
  # Z belongs here.
  z = 1 # Z inline

  mapping = merge(
    var.defaults,
    {
      # A belongs here.
      "a.key" = var.a
      "z.key" = var.z
    }
  )
}

# File footer
`
	output := verifyPreservation(t, []byte(input))
	if string(output) != expected {
		t.Fatalf("unexpected sorting:\n%s", output)
	}
}

func TestDetachedCommentsAreBoundaries(t *testing.T) {
	input := "z=1\ny=2\n\n# Next section\n\nb=3\na=4\n"
	output := verifyPreservation(t, []byte(input))
	expected := "y = 2\nz = 1\n\n# Next section\n\na = 4\nb = 3\n"
	if string(output) != expected {
		t.Fatalf("section boundary changed:\n%s", output)
	}
}

func TestSortObjectsInsideExpressions(t *testing.T) {
	cases := [][2]string{
		{`x = merge({ z = 1, a = 2 }, { z = 3 })`, `x = merge({ a = 2, z = 1 }, { z = 3 })`},
		{`x = jsonencode({ "z.key" = var.z, "a.key" = var.a })`, `x = jsonencode({ "a.key" = var.a, "z.key" = var.z })`},
		{`x = [for x in var.x : { z = x.z, a = x.a } if x.enabled]`, `x = [for x in var.x : { a = x.a, z = x.z } if x.enabled]`},
		{`x = var.enabled ? { z = 1, a = 2 } : { z = 3, a = 4 }`, `x = var.enabled ? { a = 2, z = 1 } : { a = 4, z = 3 }`},
		{`variable "schema" { type = list(object({ z = optional(string), a = number })) }`, `variable "schema" { type = list(object({ a = number, z = optional(string) })) }`},
		{`x = { (var.key) = 1, a = { z = 1, a = 2 } }`, `x = { (var.key) = 1, a = { a = 2, z = 1 } }`},
		{`x = "${jsonencode({ z = 1, a = 2 })}"`, `x = "${jsonencode({ z = 1, a = 2 })}"`},
	}
	for _, pair := range cases {
		t.Run(pair[0], func(t *testing.T) {
			output := verifyPreservation(t, []byte(pair[0]))
			expected, err := parser.New().ParseFile([]byte(pair[1]))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(output, parser.New().FormatFile(expected)) {
				t.Fatalf("unexpected output:\n%s", output)
			}
		})
	}
}

func TestInlineCommentDoesNotCaptureNextComment(t *testing.T) {
	input := "z = 1 # Z inline\n# A note\na = 2\n"
	expected := "# A note\na = 2\nz = 1 # Z inline\n"
	output := verifyPreservation(t, []byte(input))
	if string(output) != expected {
		t.Fatalf("comment attached to wrong entry:\n%s", output)
	}
}

func TestAmbiguousInlineCommentKeepsPeerOrder(t *testing.T) {
	input := "x = { z = 1, /* between peers */ a = 2 }\n"
	output := verifyPreservation(t, []byte(input))
	if string(output) != input {
		t.Fatalf("ambiguous comment moved:\n%s", output)
	}
}
