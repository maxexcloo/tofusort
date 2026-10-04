package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) ParseFile(content []byte) (*hclwrite.File, error) {
	file, diags := hclwrite.ParseConfig(content, "", hcl.Pos{})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}
	return file, nil
}

func (p *Parser) FormatFile(file *hclwrite.File) []byte {
	formatted := hclwrite.Format(file.Bytes())

	// Clean up excessive blank lines
	return p.cleanupBlankLines(formatted)
}

func (p *Parser) cleanupBlankLines(content []byte) []byte {
	tokens, diags := hclsyntax.LexConfig(content, "", hcl.InitialPos)
	if diags.HasErrors() {
		return content
	}
	blankLinesRe := regexp.MustCompile(`\n\n\n+`)
	blockStartRe := regexp.MustCompile(`\{\n\n+(\s+)`)
	clean := func(part []byte) []byte {
		part = blankLinesRe.ReplaceAll(part, []byte("\n\n"))
		return blockStartRe.ReplaceAll(part, []byte("{\n$1"))
	}

	var result bytes.Buffer
	start := 0
	for _, token := range tokens {
		switch token.Type {
		case hclsyntax.TokenComment, hclsyntax.TokenQuotedLit, hclsyntax.TokenStringLit:
			result.Write(clean(content[start:token.Range.Start.Byte]))
			result.Write(token.Bytes)
			start = token.Range.End.Byte
		}
	}
	result.Write(clean(content[start:]))
	return []byte(strings.Trim(result.String(), "\n") + "\n")
}
