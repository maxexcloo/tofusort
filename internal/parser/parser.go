package parser

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type Parser struct{}

type File struct {
	Source []byte
	Body   *hclsyntax.Body
	Tokens hclsyntax.Tokens
}

func New() *Parser { return &Parser{} }

func (p *Parser) ParseFile(content []byte) (*File, error) {
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

func (p *Parser) FormatFile(file *File) []byte {
	content := hclwrite.Format(file.Source)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	return content
}
