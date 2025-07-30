package sorter

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type Sorter struct{}

type AttrInfo struct {
	Name        string
	Expr        *hclwrite.Expression
	IsMultiLine bool
}

type BlockInfo struct {
	Block *hclwrite.Block
	Type  string
	Name  string
}

func New() *Sorter {
	return &Sorter{}
}

var blockTypeOrder = map[string]int{
	"terraform": 0,
	"provider":  1,
	"variable":  2,
	"locals":    3,
	"data":      4,
	"resource":  5,
	"module":    6,
	"output":    7,
}

var metaArgumentOrder = map[string]int{
	"count":      0,
	"for_each":   1,
	"lifecycle":  998,
	"depends_on": 999,
}

func (s *Sorter) SortFile(file *hclwrite.File) {
	body := file.Body()
	blocks := body.Blocks()

	if len(blocks) == 0 {
		return
	}

	blockInfos := make([]BlockInfo, len(blocks))
	for i, block := range blocks {
		blockType := block.Type()
		blockName := ""
		if labels := block.Labels(); len(labels) > 0 {
			blockName = strings.Join(labels, ".")
		}

		blockInfos[i] = BlockInfo{
			Block: block,
			Type:  blockType,
			Name:  blockName,
		}
	}

	sort.Slice(blockInfos, func(i, j int) bool {
		return s.compareBlocks(blockInfos[i], blockInfos[j])
	})

	body.Clear()
	for i, blockInfo := range blockInfos {
		s.sortBlockAttributes(blockInfo.Block)
		body.AppendBlock(blockInfo.Block)

		if i < len(blockInfos)-1 {
			body.AppendNewline()
		}
	}
}

func (s *Sorter) compareBlocks(a, b BlockInfo) bool {
	orderA, existsA := blockTypeOrder[a.Type]
	orderB, existsB := blockTypeOrder[b.Type]

	if existsA && existsB {
		if orderA != orderB {
			return orderA < orderB
		}
	} else if existsA != existsB {
		return existsA
	} else if !existsA && !existsB {
		if a.Type != b.Type {
			return a.Type < b.Type
		}
	}

	// Special handling for validation blocks - sort by error_message
	if a.Type == "validation" && b.Type == "validation" {
		errorMsgA := s.getValidationErrorMessage(a.Block)
		errorMsgB := s.getValidationErrorMessage(b.Block)
		if errorMsgA != errorMsgB {
			return errorMsgA < errorMsgB
		}
	}

	// Special handling for dynamic blocks - sort by block label first, then by for_each expression
	if a.Type == "dynamic" && b.Type == "dynamic" {
		// Dynamic blocks have labels (the resource type they're generating)
		labelA := s.getDynamicBlockLabel(a.Block)
		labelB := s.getDynamicBlockLabel(b.Block)
		if labelA != labelB {
			return labelA < labelB
		}
		// If same label, sort by for_each expression content
		forEachA := s.getDynamicForEachContent(a.Block)
		forEachB := s.getDynamicForEachContent(b.Block)
		if forEachA != forEachB {
			return forEachA < forEachB
		}
	}

	return a.Name < b.Name
}

func (s *Sorter) sortBlockAttributes(block *hclwrite.Block) {
	body := block.Body()
	attrs := body.Attributes()
	nestedBlocks := body.Blocks()

	if len(attrs) == 0 && len(nestedBlocks) == 0 {
		return
	}

	// Categorize attributes
	var earlyAttrs []AttrInfo
	var regularAttrs []AttrInfo
	var lateAttrs []AttrInfo

	for name, attr := range attrs {
		expr := attr.Expr()
		// Sort the expression if it contains objects
		sortedExpr := s.sortExpression(expr)
		
		isMultiLine := s.isMultiLineAttribute(sortedExpr)
		attrInfo := AttrInfo{
			Name:        name,
			Expr:        sortedExpr,
			IsMultiLine: isMultiLine,
		}

		if s.isEarlyAttribute(name) {
			earlyAttrs = append(earlyAttrs, attrInfo)
		} else if s.isLateAttribute(name) {
			lateAttrs = append(lateAttrs, attrInfo)
		} else {
			regularAttrs = append(regularAttrs, attrInfo)
		}
	}

	// Categorize blocks
	var regularBlocks []*hclwrite.Block
	var lifecycleBlocks []*hclwrite.Block

	for _, nestedBlock := range nestedBlocks {
		if nestedBlock.Type() == "lifecycle" {
			lifecycleBlocks = append(lifecycleBlocks, nestedBlock)
		} else {
			regularBlocks = append(regularBlocks, nestedBlock)
		}
	}

	// Sort all categories
	sort.Slice(earlyAttrs, func(i, j int) bool {
		return s.compareEarlyAttributes(earlyAttrs[i].Name, earlyAttrs[j].Name)
	})
	sort.Slice(regularAttrs, func(i, j int) bool {
		return regularAttrs[i].Name < regularAttrs[j].Name
	})
	sort.Slice(lateAttrs, func(i, j int) bool {
		return s.compareLateAttributes(lateAttrs[i].Name, lateAttrs[j].Name)
	})
	sort.Slice(regularBlocks, func(i, j int) bool {
		return s.compareBlocks(BlockInfo{Block: regularBlocks[i], Type: regularBlocks[i].Type()}, 
		                       BlockInfo{Block: regularBlocks[j], Type: regularBlocks[j].Type()})
	})

	// Now rebuild the body in the correct order
	// First remove all existing attributes and blocks
	for name := range attrs {
		body.RemoveAttribute(name)
	}
	for _, block := range nestedBlocks {
		body.RemoveBlock(block)
	}

	// Add content in the correct order
	// 1. Early meta-arguments (count, for_each)
	s.writeAttributeGroup(body, earlyAttrs)

	// Add blank line after early meta-arguments if we have them and other content
	hasOtherContent := len(regularAttrs) > 0 || len(lateAttrs) > 0 ||
		len(regularBlocks) > 0 || len(lifecycleBlocks) > 0
	if len(earlyAttrs) > 0 && hasOtherContent {
		body.AppendNewline()
	}

	// 2. Regular attributes (grouped by single-line vs multi-line)
	s.writeAttributeGroup(body, regularAttrs)

	// 3. Regular nested blocks (not lifecycle) - recursively sort them
	for i, block := range regularBlocks {
		// Add blank line before blocks if we have attributes or previous blocks
		if len(regularAttrs) > 0 || i > 0 {
			body.AppendNewline()
		}
		s.sortBlockAttributes(block)
		body.AppendBlock(block)
	}

	// 4. Late meta-arguments (depends_on attributes)
	if len(lateAttrs) > 0 {
		// Add blank line before late attributes if we have regular content
		if len(regularAttrs) > 0 || len(regularBlocks) > 0 {
			body.AppendNewline()
		}
		s.writeAttributeGroup(body, lateAttrs)
	}

	// 5. Late blocks (lifecycle) - recursively sort them
	for _, block := range lifecycleBlocks {
		// Add blank line before lifecycle blocks
		if len(regularAttrs) > 0 || len(regularBlocks) > 0 || len(lateAttrs) > 0 {
			body.AppendNewline()
		}
		s.sortBlockAttributes(block)
		body.AppendBlock(block)
	}
}

func (s *Sorter) isEarlyAttribute(name string) bool {
	order, exists := metaArgumentOrder[name]
	return exists && order < 500
}

func (s *Sorter) isLateAttribute(name string) bool {
	order, exists := metaArgumentOrder[name]
	return exists && order >= 998
}

func (s *Sorter) compareEarlyAttributes(a, b string) bool {
	orderA, existsA := metaArgumentOrder[a]
	orderB, existsB := metaArgumentOrder[b]

	if existsA && existsB {
		return orderA < orderB
	}
	return a < b
}

func (s *Sorter) compareLateAttributes(a, b string) bool {
	orderA, existsA := metaArgumentOrder[a]
	orderB, existsB := metaArgumentOrder[b]

	if existsA && existsB {
		return orderA < orderB
	}
	return a < b
}

func (s *Sorter) isMultiLineAttribute(expr *hclwrite.Expression) bool {
	tokens := expr.BuildTokens(nil)
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenNewline {
			return true
		}
	}
	return false
}

func (s *Sorter) writeAttributeGroup(body *hclwrite.Body, attrs []AttrInfo) {
	if len(attrs) == 0 {
		return
	}

	// Separate single-line and multi-line attributes
	var singleLineAttrs []AttrInfo
	var multiLineAttrs []AttrInfo

	for _, attr := range attrs {
		if attr.IsMultiLine {
			multiLineAttrs = append(multiLineAttrs, attr)
		} else {
			singleLineAttrs = append(singleLineAttrs, attr)
		}
	}

	// Write single-line attributes first (grouped together)
	for _, attrInfo := range singleLineAttrs {
		body.SetAttributeRaw(attrInfo.Name, attrInfo.Expr.BuildTokens(nil))
	}

	// Write multi-line attributes with blank lines before each one
	for _, attrInfo := range multiLineAttrs {
		// Add blank line before each multi-line attribute
		body.AppendNewline()
		body.SetAttributeRaw(attrInfo.Name, attrInfo.Expr.BuildTokens(nil))
	}
}

// sortExpression attempts to sort object expressions (both HCL objects and jsonencode calls)
func (s *Sorter) sortExpression(expr *hclwrite.Expression) *hclwrite.Expression {
	// For now, disable object sorting due to complexity of preserving formatting
	// This is a stretch goal that needs more careful implementation
	return expr
}

// getValidationErrorMessage extracts the error_message from a validation block
func (s *Sorter) getValidationErrorMessage(block *hclwrite.Block) string {
	if block.Type() != "validation" {
		return ""
	}
	
	body := block.Body()
	attrs := body.Attributes()
	
	if errorMsgAttr, exists := attrs["error_message"]; exists {
		// Extract the string value from the expression
		tokens := errorMsgAttr.Expr().BuildTokens(nil)
		for _, token := range tokens {
			if token.Type == hclsyntax.TokenQuotedLit {
				// Remove quotes and return the content
				content := string(token.Bytes)
				if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
					return content[1 : len(content)-1]
				}
				return content
			}
		}
	}
	
	return ""
}

// getDynamicBlockLabel extracts the label from a dynamic block
func (s *Sorter) getDynamicBlockLabel(block *hclwrite.Block) string {
	if block.Type() != "dynamic" {
		return ""
	}
	
	labels := block.Labels()
	if len(labels) > 0 {
		return labels[0]
	}
	
	return ""
}

// getDynamicForEachContent extracts the for_each expression content from a dynamic block
func (s *Sorter) getDynamicForEachContent(block *hclwrite.Block) string {
	if block.Type() != "dynamic" {
		return ""
	}
	
	body := block.Body()
	attrs := body.Attributes()
	
	if forEachAttr, exists := attrs["for_each"]; exists {
		// Convert the expression to string for comparison
		tokens := forEachAttr.Expr().BuildTokens(nil)
		var content strings.Builder
		for _, token := range tokens {
			content.Write(token.Bytes)
		}
		return content.String()
	}
	
	return ""
}
