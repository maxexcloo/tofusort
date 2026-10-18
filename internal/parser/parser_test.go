package parser

import (
	"strings"
	"testing"
)

func TestFormatPreservesLiteralAndCommentWhitespace(t *testing.T) {
	input := `value = <<-EOT
first


{

  content
}
EOT

/* first


last */


other = 1
`
	file, err := ParseFile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	output := string(FormatFile(file))
	for _, literal := range []string{"first\n\n\n{\n\n  content\n}\n", "/* first\n\n\nlast */"} {
		if !strings.Contains(output, literal) {
			t.Errorf("formatting changed literal or comment %q:\n%s", literal, output)
		}
	}
}

func TestParseFileNormalisesCRLF(t *testing.T) {
	file, err := ParseFile([]byte("a = 1\r\nb = <<EOT\r\nx\r\nEOT\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "a = 1\nb = <<EOT\nx\nEOT\n"; string(file.Source) != want {
		t.Fatalf("source = %q, want %q", file.Source, want)
	}
}

func TestParseFileNormalisesLoneCR(t *testing.T) {
	file, err := ParseFile([]byte("a = 1\rb = 2\r\r\nc = 3\r"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "a = 1\nb = 2\n\nc = 3\n"; string(file.Source) != want {
		t.Fatalf("source = %q, want %q", file.Source, want)
	}
}
