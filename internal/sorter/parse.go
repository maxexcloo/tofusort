package sorter

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type File struct {
	Source []byte
	Body   *hclsyntax.Body
	Tokens hclsyntax.Tokens
}

// ParseFile normalises CRLF and lone CR line endings to LF, so sorted output
// always uses LF.
func ParseFile(content []byte) (*File, error) {
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	content = bytes.ReplaceAll(content, []byte("\r"), []byte("\n"))
	file, diags := hclsyntax.ParseConfig(content, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}
	tokens, diags := hclsyntax.LexConfig(content, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to lex HCL: %s", diags.Error())
	}
	return &File{Source: content, Body: file.Body.(*hclsyntax.Body), Tokens: tokens}, nil
}

func format(source []byte) []byte {
	content := hclwrite.Format(source)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	return content
}
