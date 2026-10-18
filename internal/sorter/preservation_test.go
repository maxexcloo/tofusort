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

func verifyPreservation(t testing.TB, input []byte, options Options) []byte {
	t.Helper()
	file, err := parser.ParseFile(input)
	if err != nil {
		t.Fatal(err)
	}
	before := bodySignature(file.Body, true)
	comments := commentSignature(file)
	if err := SortFile(file, options); err != nil {
		t.Fatal(err)
	}
	output := parser.FormatFile(file)
	if got := bodySignature(file.Body, true); got != before {
		t.Fatalf("configuration meaning changed\nbefore: %s\nafter: %s", before, got)
	}
	if got := commentSignature(file); got != comments {
		t.Fatalf("comments changed: %s != %s", got, comments)
	}
	if err := SortFile(file, options); err != nil {
		t.Fatal(err)
	}
	if again := parser.FormatFile(file); !bytes.Equal(again, output) {
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
			// Binary notation stays exact and fast; GoString expands huge exponents in decimal.
			if value.Type() == cty.Number && value.IsKnown() && !value.IsNull() {
				return "number:" + value.AsBigFloat().Text('p', 0)
			}
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
			comments = append(comments, strings.TrimRight(string(token.Bytes), "\r\n"))
		}
	}
	sort.Strings(comments)
	return fmt.Sprint(comments)
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
		t.Run(filepath.Base(path), func(t *testing.T) { verifyPreservation(t, input, Options{}) })
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

// hasSpacedNumberTraversal reports a number literal followed by a separate dot,
// such as `0 .0`. hclwrite.Format joins these into one number, changing meaning;
// this upstream limitation is accepted, so fuzzing skips the input class.
func hasSpacedNumberTraversal(tokens hclsyntax.Tokens) bool {
	for i := 1; i < len(tokens); i++ {
		if tokens[i-1].Type == hclsyntax.TokenNumberLit && tokens[i].Type == hclsyntax.TokenDot {
			return true
		}
	}
	return false
}

func FuzzSortPreservesStructure(f *testing.F) {
	for _, seed := range []string{"#", "x=0 # preserve trailing spaces  ", `x = merge({ z = 1, a = 2 }, var.x)`, `variable "schema" { type = list(object({ z = string, a = number })) }`, "z=1\na=2\n", "x={z=1,a=2}\n", "x={z=1,a=2} # note\n", "x=[{z=1,a=2},3]\n", "x=<<EOT\nhello\nEOT\n"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 100000 {
			t.Skip()
		}
		file, err := parser.ParseFile([]byte(src))
		if err != nil || hasSpacedNumberTraversal(file.Tokens) {
			t.Skip()
		}
		verifyPreservation(t, []byte(src), Options{})
		verifyPreservation(t, []byte(src), Options{IdentifiersFirst: true})
	})
}

func TestSortPreservesValues(t *testing.T) {
	inputs := []string{
		"value = { z = { b = 1, a = 2 }, z = 3 }\n",
		"value = { z = <<-EOT\n  hello\n\n\n  world\n  EOT\na = 1\n}\n",
		"value = { z = 1, a = 2 }\n",
		"value = [{ z = 1, a = 2 }, { b = 3, a = 4 }]\n",
		"value = { z = { b = 1, a = 2 }, a = [3, 2, 1] }\n",
		"value = { z = <<EOT\nhello\n\n\nworld\nEOT\na = 1\n}\n",
		"value = { z = 1, a = 2, z = 3 }\n",
		"value = { z = \"literal { text }\", a = true }\n",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			file, err := parser.ParseFile([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			if err := SortFile(file, Options{}); err != nil {
				t.Fatal(err)
			}
			output := parser.FormatFile(file)
			before, d := hclsyntax.ParseConfig([]byte(input), "input.tfvars", hcl.InitialPos)
			if d.HasErrors() {
				t.Fatal(d)
			}
			after, d := hclsyntax.ParseConfig(output, "output.tfvars", hcl.InitialPos)
			if d.HasErrors() {
				t.Fatalf("invalid output: %s\n%s", d, output)
			}
			want, d := before.Body.(*hclsyntax.Body).Attributes["value"].Expr.Value(nil)
			if d.HasErrors() {
				t.Fatal(d)
			}
			got, d := after.Body.(*hclsyntax.Body).Attributes["value"].Expr.Value(nil)
			if d.HasErrors() {
				t.Fatal(d)
			}
			if !want.RawEquals(got) {
				t.Fatalf("value changed:\n%s", output)
			}
			again, err := parser.ParseFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if err := SortFile(again, Options{}); err != nil {
				t.Fatal(err)
			}
			if second := parser.FormatFile(again); !bytes.Equal(output, second) {
				t.Fatalf("not idempotent:\n%s\nthen:\n%s", output, second)
			}
		})
	}
}

func TestSortPreservesNestedBlockOrder(t *testing.T) {
	input := `resource "example" "test" {
  dynamic "rule" {
    for_each = ["z"]
    content { name = "z" }
  }
  rule { name = "middle" }
  dynamic "rule" {
    for_each = ["a"]
    content { name = "a" }
  }
  provisioner "local-exec" { command = "first" }
  provisioner "remote-exec" { inline = ["second"] }
}
`
	file, err := parser.ParseFile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	blocks := file.Body.Blocks[0].Body.Blocks
	if err := SortFile(file, Options{}); err != nil {
		t.Fatal(err)
	}
	got := file.Body.Blocks[0].Body.Blocks
	for i := range blocks {
		if got[i].Type != blocks[i].Type || strings.Join(got[i].Labels, "\x00") != strings.Join(blocks[i].Labels, "\x00") {
			t.Fatalf("nested blocks reordered:\n%s", parser.FormatFile(file))
		}
	}
}
