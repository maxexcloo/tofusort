package sorter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/maxexcloo/tofusort/internal/parser"
)

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
		"# CR before end of file\r",
		"# CR runs\r\r\n",
		"A0=b\r",
		"A=1E000020000000\n",
	}
	for i, input := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) { verifyPreservation(t, []byte(input), Options{}) })
	}
}

func TestIdentifiersFirst(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		options Options
	}{
		{
			name:  "default",
			input: `items = [{ type = "t", name = "n", id = 3, a = 1 }]`,
			want:  `items = [{ a = 1, id = 3, name = "n", type = "t" }]`,
		},
		{
			name:    "list order and quoted keys",
			input:   `items = [{ a = 1, id = 3, name = "n", "type" = "t" }, { a = 2, name = "m" }]`,
			want:    `items = [{ "type" = "t", name = "n", id = 3, a = 1 }, { name = "m", a = 2 }]`,
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "parenthesised object",
			input:   `items = [({ a = 1, name = "n", type = "t" })]`,
			want:    `items = [({ type = "t", name = "n", a = 1 })]`,
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "ordinary objects keep default policy",
			input:   `items = { type = "t", name = "n", id = 3, a = 1 }`,
			want:    `items = { a = 1, id = 3, name = "n", type = "t" }`,
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "nested object keeps default policy",
			input:   `items = [{ a = { type = "t", name = "n", id = 3, a = 1 }, name = "outer" }]`,
			want:    `items = [{ name = "outer", a = { a = 1, id = 3, name = "n", type = "t" } }]`,
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "duplicate keys retain peer order",
			input:   `items = [{ a = 1, name = "n", type = "t", a = 2 }]`,
			want:    `items = [{ a = 1, name = "n", type = "t", a = 2 }]`,
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "computed keys retain peer order",
			input:   `items = [{ (var.key) = 1, name = "n", type = "t" }]`,
			want:    `items = [{ (var.key) = 1, name = "n", type = "t" }]`,
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "comment attachment",
			input:   "items = [{\n# A\na = 1\n# Name\nname = \"n\" # trailing\n# Type\ntype = \"t\"\n}]",
			want:    "items = [{\n# Type\ntype = \"t\"\n# Name\nname = \"n\" # trailing\n# A\na = 1\n}]",
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "detached comment boundary",
			input:   "items = [{\nid = 3\n\n# Section\n\nname = \"n\"\ntype = \"t\"\na = 1\n}]",
			want:    "items = [{\nid = 3\n\n# Section\n\ntype = \"t\"\nname = \"n\"\na = 1\n}]",
			options: Options{IdentifiersFirst: true},
		},
		{
			name:    "multiline identifier precedes ordinary values",
			input:   "items = [{\na = 1\nid = 3\nname = <<EOT\nname\nEOT\ntype = \"t\"\n}]",
			want:    "items = [{\ntype = \"t\"\nname = <<EOT\nname\nEOT\nid = 3\na = 1\n}]",
			options: Options{IdentifiersFirst: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := verifyPreservation(t, []byte(tt.input), tt.options)
			want, err := parser.ParseFile([]byte(tt.want))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(output, parser.FormatFile(want)) {
				t.Fatalf("unexpected order:\n%s\nwant:\n%s", output, parser.FormatFile(want))
			}
		})
	}
}

func BenchmarkSortFlatAttributes(b *testing.B) {
	for _, count := range []int{2000, 4000, 8000} {
		var input strings.Builder
		for i := count; i > 0; i-- {
			fmt.Fprintf(&input, "attribute_%05d = %d\n", i, i)
		}
		source := []byte(input.String())
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				file, err := parser.ParseFile(source)
				if err != nil {
					b.Fatal(err)
				}
				if err := SortFile(file, Options{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
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
	output := verifyPreservation(t, []byte(input), Options{})
	if string(output) != expected {
		t.Fatalf("unexpected sorting:\n%s", output)
	}
}

func TestDetachedCommentsAreBoundaries(t *testing.T) {
	input := "z=1\ny=2\n\n# Next section\n\nb=3\na=4\n"
	output := verifyPreservation(t, []byte(input), Options{})
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
			output := verifyPreservation(t, []byte(pair[0]), Options{})
			expected, err := parser.ParseFile([]byte(pair[1]))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(output, parser.FormatFile(expected)) {
				t.Fatalf("unexpected output:\n%s", output)
			}
		})
	}
}

func TestInlineCommentDoesNotCaptureNextComment(t *testing.T) {
	input := "z = 1 # Z inline\n# A note\na = 2\n"
	expected := "# A note\na = 2\nz = 1 # Z inline\n"
	output := verifyPreservation(t, []byte(input), Options{})
	if string(output) != expected {
		t.Fatalf("comment attached to wrong entry:\n%s", output)
	}
}

func TestTrailingSpacesDoNotJoinMovedComment(t *testing.T) {
	for _, input := range []string{"B=0#\nA=0 \n", "B=0#\nA=0 "} {
		output := verifyPreservation(t, []byte(input), Options{})
		if expected := "A = 0\nB = 0 #\n"; string(output) != expected {
			t.Fatalf("unexpected output for %q:\n%q", input, output)
		}
	}
}

func TestAmbiguousInlineCommentKeepsPeerOrder(t *testing.T) {
	input := "x = { z = 1, /* between peers */ a = 2 }\n"
	output := verifyPreservation(t, []byte(input), Options{})
	if string(output) != input {
		t.Fatalf("ambiguous comment moved:\n%s", output)
	}
}

func TestMultilineObjectEntriesKeepCommas(t *testing.T) {
	cases := map[string]string{
		"x = {\n  b = 1,\n  a = 2,\n}\n":    "x = {\n  a = 2,\n  b = 1,\n}\n",
		"x = {\n  b = 1, # B\n  a = 2\n}\n": "x = {\n  a = 2\n  b = 1, # B\n}\n",
	}
	for input, expected := range cases {
		if output := verifyPreservation(t, []byte(input), Options{}); string(output) != expected {
			t.Errorf("unexpected commas for %q:\n%s", input, output)
		}
	}
}

func TestInlineObjectWithMultilineValueStaysInline(t *testing.T) {
	input := "x = { b = {\n  y = 1\n}, a = 2 }\n"
	expected := "x = { a = 2, b = {\n  y = 1\n} }\n"
	if output := verifyPreservation(t, []byte(input), Options{}); string(output) != expected {
		t.Fatalf("inline object layout changed:\n%s", output)
	}
}
