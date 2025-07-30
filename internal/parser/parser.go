package parser

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/hcl/v2"
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
	text := string(content)

	// Replace multiple consecutive empty lines with single empty line
	// This regex matches 3 or more consecutive newlines and replaces with 2 newlines
	re := regexp.MustCompile(`\n\n\n+`)
	text = re.ReplaceAllString(text, "\n\n")

	// Clean up blank lines at the start of blocks (after opening brace)
	// This handles the case where we have "{\n\n\n  attribute"
	blockStartRe := regexp.MustCompile(`\{\n\n+(\s+)`)
	text = blockStartRe.ReplaceAllString(text, "{\n$1")

	// Ensure file ends with exactly one newline
	text = regexp.MustCompile(`\n*$`).ReplaceAllString(text, "\n")

	return []byte(text)
}
