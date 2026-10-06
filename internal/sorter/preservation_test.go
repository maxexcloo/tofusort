package sorter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/maxexcloo/tofusort/internal/parser"
)

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
			p := parser.New()
			file, err := p.ParseFile([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			if err := New().SortFile(file); err != nil {
				t.Fatal(err)
			}
			output := p.FormatFile(file)
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
			again, err := p.ParseFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if err := New().SortFile(again); err != nil {
				t.Fatal(err)
			}
			if second := p.FormatFile(again); !bytes.Equal(output, second) {
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
	p := parser.New()
	file, err := p.ParseFile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	blocks := file.Body.Blocks[0].Body.Blocks
	if err := New().SortFile(file); err != nil {
		t.Fatal(err)
	}
	got := file.Body.Blocks[0].Body.Blocks
	for i := range blocks {
		if got[i].Type != blocks[i].Type || strings.Join(got[i].Labels, "\x00") != strings.Join(blocks[i].Labels, "\x00") {
			t.Fatalf("nested blocks reordered:\n%s", p.FormatFile(file))
		}
	}
}
