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
	p := New()
	file, err := p.ParseFile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	output := string(p.FormatFile(file))
	for _, literal := range []string{"first\n\n\n{\n\n  content\n}\n", "/* first\n\n\nlast */"} {
		if !strings.Contains(output, literal) {
			t.Errorf("formatting changed literal or comment %q:\n%s", literal, output)
		}
	}
	if strings.Contains(output, "*/\n\n\nother") {
		t.Errorf("formatting did not collapse syntactic blank lines:\n%s", output)
	}
}
