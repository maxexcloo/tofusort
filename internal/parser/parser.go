package parser

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

// ParseFile normalises CRLF line endings to LF, so sorted output always uses LF.
// Repeating the replacement also folds runs such as CR CR LF, keeping it idempotent.
func ParseFile(content []byte) (*File, error) {
	for bytes.Contains(content, []byte("\r\n")) {
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	}
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

func FormatFile(file *File) []byte {
	content := hclwrite.Format(file.Source)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	return content
}
