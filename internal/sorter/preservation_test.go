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
	"github.com/zclconf/go-cty/cty"
)

func verifyPreservation(t testing.TB, input []byte, options Options) []byte {
	t.Helper()
	file, err := ParseFile(input)
	if err != nil {
		t.Fatal(err)
	}
	before := bodySignature(file.Body, true)
	comments := commentSignature(file)
	output, err := SortFile(input, options)
	if err != nil {
		t.Fatal(err)
	}
	file, err = ParseFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := bodySignature(file.Body, true); got != before {
		t.Fatalf("configuration meaning changed\nbefore: %s\nafter: %s", before, got)
	}
	if got := commentSignature(file); got != comments {
		t.Fatalf("comments changed: %s != %s", got, comments)
	}
	again, err := SortFile(output, options)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, output) {
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

func commentSignature(file *File) string {
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
		file, err := ParseFile([]byte(src))
		if err != nil || hasSpacedNumberTraversal(file.Tokens) {
			t.Skip()
		}
		verifyPreservation(t, []byte(src), Options{})
		verifyPreservation(t, []byte(src), Options{IdentifiersFirst: true})
	})
}
