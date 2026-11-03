package sorter

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
	output := string(format(file.Source))
	for _, literal := range []string{"first\n\n\n{\n\n  content\n}\n", "/* first\n\n\nlast */"} {
		if !strings.Contains(output, literal) {
			t.Errorf("formatting changed literal or comment %q:\n%s", literal, output)
		}
	}
}

func TestParseFileNormalisesLineEndings(t *testing.T) {
	tests := map[string]string{
		"a = 1\r\nb = <<EOT\r\nx\r\nEOT\r\n": "a = 1\nb = <<EOT\nx\nEOT\n",
		"a = 1\rb = 2\r\r\nc = 3\r":          "a = 1\nb = 2\n\nc = 3\n",
	}
	for input, want := range tests {
		file, err := ParseFile([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if string(file.Source) != want {
			t.Errorf("source = %q, want %q", file.Source, want)
		}
	}
}
