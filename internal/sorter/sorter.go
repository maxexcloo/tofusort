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
	"count":           0,
	"for_each":        1,
	"depends_on":      998,
	"force_new":       999,
	"lifecycle":       1000,
	"triggers_replace": 1001,
}

func (s *Sorter) SortFile(file *hclwrite.File) {
	body := file.Body()
	blocks := body.Blocks()
	attrs := body.Attributes()

	// Sort top-level attributes if there are any (e.g., for .tfvars files)
	if len(attrs) > 0 {
		s.sortBodyAttributes(body)
	}

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

	// Special handling for dynamic blocks - sort by block label first, then by content id/label/name, then by for_each expression
	if a.Type == "dynamic" && b.Type == "dynamic" {
		// Dynamic blocks have labels (the resource type they're generating) - case insensitive
		labelA := strings.ToLower(s.getDynamicBlockLabel(a.Block))
		labelB := strings.ToLower(s.getDynamicBlockLabel(b.Block))
		if labelA != labelB {
			return labelA < labelB
		}
		// If same label, sort by content id/label/name attributes - case insensitive
		contentKeyA := strings.ToLower(s.getDynamicContentSortKey(a.Block))
		contentKeyB := strings.ToLower(s.getDynamicContentSortKey(b.Block))
		if contentKeyA != contentKeyB {
			return contentKeyA < contentKeyB
		}
		// If same content key, sort by for_each expression content
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
		// DISABLED: Expression sorting causes corruption - keep original expressions
		// sortedExpr := s.sortExpression(expr)

		isMultiLine := s.isMultiLineAttribute(expr)
		attrInfo := AttrInfo{
			Name:        name,
			Expr:        expr,
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

func (s *Sorter) sortBodyAttributes(body *hclwrite.Body) {
	attrs := body.Attributes()
	if len(attrs) == 0 {
		return
	}

	// Build list of attributes with metadata
	var attrInfos []AttrInfo
	for name, attr := range attrs {
		expr := attr.Expr()
		isMultiLine := s.isMultiLineAttribute(expr)

		attrInfos = append(attrInfos, AttrInfo{
			Name:        name,
			Expr:        expr,
			IsMultiLine: isMultiLine,
		})
	}

	// Sort attributes alphabetically
	sort.Slice(attrInfos, func(i, j int) bool {
		return attrInfos[i].Name < attrInfos[j].Name
	})

	// Remove all existing attributes
	for name := range attrs {
		body.RemoveAttribute(name)
	}

	// Add attributes back in sorted order with proper spacing
	s.writeAttributeGroup(body, attrInfos)
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
	// TEMPORARILY DISABLED: Object/array sorting is causing syntax corruption
	// Need to fix key parsing for complex expressions before re-enabling
	return expr
}

// isJsonEncodeCall checks if the expression is a jsonencode(...) call
func (s *Sorter) isJsonEncodeCall(tokens hclwrite.Tokens) bool {
	if len(tokens) < 4 {
		return false
	}
	
	// Check first token is identifier "jsonencode"
	if tokens[0].Type != hclsyntax.TokenIdent || string(tokens[0].Bytes) != "jsonencode" {
		return false
	}
	
	// Check second token is opening parenthesis
	if tokens[1].Type != hclsyntax.TokenOParen {
		return false
	}
	
	// Check last token is closing parenthesis
	if tokens[len(tokens)-1].Type != hclsyntax.TokenCParen {
		return false
	}
	
	return true
}

// sortJsonEncode sorts the content inside jsonencode(...)
func (s *Sorter) sortJsonEncode(tokens hclwrite.Tokens) hclwrite.Tokens {
	// Extract content between jsonencode( and )
	contentStart := 2
	contentEnd := len(tokens) - 1
	
	// Keep the jsonencode( prefix
	result := make(hclwrite.Tokens, 0, len(tokens))
	result = append(result, tokens[0]) // jsonencode
	result = append(result, tokens[1]) // (
	
	// Extract and sort the content
	contentTokens := tokens[contentStart:contentEnd]
	if sortedContent := s.sortObjectLiteral(contentTokens); sortedContent != nil {
		result = append(result, sortedContent...)
	} else {
		// If we can't sort, keep original
		result = append(result, contentTokens...)
	}
	
	// Add closing parenthesis
	result = append(result, tokens[len(tokens)-1]) // )
	
	return result
}

// isObjectLiteral checks if tokens represent an object literal { ... }
func (s *Sorter) isObjectLiteral(tokens hclwrite.Tokens) bool {
	if len(tokens) < 2 {
		return false
	}
	
	// Find first non-whitespace token
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenNewline || token.Type == hclsyntax.TokenComment {
			continue
		}
		return token.Type == hclsyntax.TokenOBrace
	}
	
	return false
}

// sortObjectLiteral sorts the keys in an object literal
func (s *Sorter) sortObjectLiteral(tokens hclwrite.Tokens) hclwrite.Tokens {
	// Find the opening brace
	openBraceIdx := -1
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenOBrace {
			openBraceIdx = i
			break
		}
	}
	
	if openBraceIdx == -1 {
		return nil
	}
	
	// Parse the object entries
	entries := s.parseObjectEntries(tokens, openBraceIdx)
	if len(entries) == 0 {
		return nil
	}
	
	// Separate single-line and multi-line entries
	var singleLineEntries []ObjectEntry
	var multiLineEntries []ObjectEntry
	
	for _, entry := range entries {
		if s.isMultiLineObjectEntry(entry) {
			multiLineEntries = append(multiLineEntries, entry)
		} else {
			singleLineEntries = append(singleLineEntries, entry)
		}
	}
	
	// Sort both groups alphabetically by key
	sort.Slice(singleLineEntries, func(i, j int) bool {
		return singleLineEntries[i].Key < singleLineEntries[j].Key
	})
	sort.Slice(multiLineEntries, func(i, j int) bool {
		return multiLineEntries[i].Key < multiLineEntries[j].Key
	})
	
	// Combine: single-line first, then multi-line
	sortedEntries := append(singleLineEntries, multiLineEntries...)
	
	// Rebuild the tokens
	return s.rebuildObjectTokens(tokens, sortedEntries, openBraceIdx, len(singleLineEntries) > 0 && len(multiLineEntries) > 0)
}

type ObjectEntry struct {
	Key     string
	Tokens  hclwrite.Tokens
	StartIdx int
	EndIdx   int
}

// parseObjectEntries parses key-value pairs from object tokens
func (s *Sorter) parseObjectEntries(tokens hclwrite.Tokens, startIdx int) []ObjectEntry {
	var entries []ObjectEntry
	
	i := startIdx + 1 // Skip opening brace
	braceLevel := 1
	
	for i < len(tokens) && braceLevel > 0 {
		// Skip whitespace and comments
		for i < len(tokens) && (tokens[i].Type == hclsyntax.TokenNewline || tokens[i].Type == hclsyntax.TokenComment) {
			i++
		}
		
		if i >= len(tokens) {
			break
		}
		
		// Track brace level
		if tokens[i].Type == hclsyntax.TokenOBrace {
			braceLevel++
		} else if tokens[i].Type == hclsyntax.TokenCBrace {
			braceLevel--
			if braceLevel == 0 {
				break
			}
		}
		
		// Look for key tokens at top level
		if braceLevel == 1 && (tokens[i].Type == hclsyntax.TokenIdent || tokens[i].Type == hclsyntax.TokenQuotedLit) {
			entry := s.parseObjectEntry(tokens, i)
			if entry != nil {
				entries = append(entries, *entry)
				i = entry.EndIdx
			} else {
				i++
			}
		} else {
			i++
		}
	}
	
	return entries
}

// parseObjectEntry parses a single key-value pair starting at the given index
func (s *Sorter) parseObjectEntry(tokens hclwrite.Tokens, startIdx int) *ObjectEntry {
	if startIdx >= len(tokens) {
		return nil
	}
	
	// Extract key
	keyToken := tokens[startIdx]
	key := s.extractKeyName(keyToken)
	if key == "" {
		return nil
	}
	
	// Find the equals sign or colon
	equalIdx := -1
	for i := startIdx + 1; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenEqual || tokens[i].Type == hclsyntax.TokenColon {
			equalIdx = i
			break
		}
		// Stop if we hit another key or closing brace at the same level
		if tokens[i].Type == hclsyntax.TokenIdent || tokens[i].Type == hclsyntax.TokenQuotedLit || tokens[i].Type == hclsyntax.TokenCBrace {
			break
		}
	}
	
	if equalIdx == -1 {
		return nil
	}
	
	// Find the end of the value
	endIdx := s.findValueEnd(tokens, equalIdx+1)
	if endIdx == -1 {
		return nil
	}
	
	// Extract all tokens for this entry
	entryTokens := make(hclwrite.Tokens, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		entryTokens = append(entryTokens, tokens[i])
	}
	
	// Recursively sort nested objects within this entry
	entryTokens = s.recursiveSortTokens(entryTokens)
	
	return &ObjectEntry{
		Key:      key,
		Tokens:   entryTokens,
		StartIdx: startIdx,
		EndIdx:   endIdx,
	}
}

// recursiveSortTokens recursively sorts any nested objects within the token sequence
func (s *Sorter) recursiveSortTokens(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) == 0 {
		return tokens
	}
	
	// Look for nested objects and sort them
	result := make(hclwrite.Tokens, 0, len(tokens))
	i := 0
	
	for i < len(tokens) {
		if tokens[i].Type == hclsyntax.TokenOBrace {
			// Found a nested object, extract and sort it
			objEnd := s.findMatchingBrace(tokens, i)
			if objEnd > i {
				// Extract the object tokens
				objTokens := tokens[i:objEnd+1]
				
				// Try to sort this nested object
				if sortedObj := s.sortObjectLiteral(objTokens); sortedObj != nil {
					result = append(result, sortedObj...)
				} else {
					result = append(result, objTokens...)
				}
				
				i = objEnd + 1
			} else {
				result = append(result, tokens[i])
				i++
			}
		} else {
			result = append(result, tokens[i])
			i++
		}
	}
	
	return result
}

// findMatchingBrace finds the closing brace that matches the opening brace at the given index
func (s *Sorter) findMatchingBrace(tokens hclwrite.Tokens, openIdx int) int {
	if openIdx >= len(tokens) || tokens[openIdx].Type != hclsyntax.TokenOBrace {
		return -1
	}
	
	braceLevel := 1
	for i := openIdx + 1; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenOBrace {
			braceLevel++
		} else if tokens[i].Type == hclsyntax.TokenCBrace {
			braceLevel--
			if braceLevel == 0 {
				return i
			}
		}
	}
	
	return -1
}

// findValueEnd finds the end index of a value expression
func (s *Sorter) findValueEnd(tokens hclwrite.Tokens, startIdx int) int {
	braceLevel := 0
	bracketLevel := 0
	parenLevel := 0
	
	for i := startIdx; i < len(tokens); i++ {
		token := tokens[i]
		
		switch token.Type {
		case hclsyntax.TokenOBrace:
			braceLevel++
		case hclsyntax.TokenCBrace:
			if braceLevel > 0 {
				braceLevel--
			} else {
				// End of object
				return i
			}
		case hclsyntax.TokenOBrack:
			bracketLevel++
		case hclsyntax.TokenCBrack:
			bracketLevel--
		case hclsyntax.TokenOParen:
			parenLevel++
		case hclsyntax.TokenCParen:
			parenLevel--
		case hclsyntax.TokenNewline:
			// If we're at the top level and hit a newline, check if the next meaningful token is a key
			if braceLevel == 0 && bracketLevel == 0 && parenLevel == 0 {
				nextIdx := s.findNextNonWhitespace(tokens, i+1)
				if nextIdx < len(tokens) && s.isKeyLikeToken(tokens[nextIdx]) {
					return i + 1
				}
			}
		case hclsyntax.TokenComma:
			// If we're at the top level, comma ends the value
			if braceLevel == 0 && bracketLevel == 0 && parenLevel == 0 {
				return i + 1
			}
		}
	}
	
	return len(tokens)
}

// isKeyLikeToken checks if a token could be the start of a key
func (s *Sorter) isKeyLikeToken(token *hclwrite.Token) bool {
	return token.Type == hclsyntax.TokenIdent || token.Type == hclsyntax.TokenQuotedLit
}

// findNextNonWhitespace finds the next non-whitespace token
func (s *Sorter) findNextNonWhitespace(tokens hclwrite.Tokens, start int) int {
	for i := start; i < len(tokens); i++ {
		if tokens[i].Type != hclsyntax.TokenNewline && tokens[i].Type != hclsyntax.TokenComment {
			return i
		}
	}
	return len(tokens)
}

// extractKeyName extracts the key name from a token
func (s *Sorter) extractKeyName(token *hclwrite.Token) string {
	key := string(token.Bytes)
	// Remove quotes if present
	if token.Type == hclsyntax.TokenQuotedLit && len(key) >= 2 && key[0] == '"' && key[len(key)-1] == '"' {
		key = key[1 : len(key)-1]
	}
	return key
}

// isMultiLineObjectEntry checks if an object entry spans multiple lines
func (s *Sorter) isMultiLineObjectEntry(entry ObjectEntry) bool {
	// Check if the value contains nested structures (braces, brackets) or spans multiple lines
	braceLevel := 0
	bracketLevel := 0
	hasNestedStructure := false
	lineCount := 0
	
	for _, token := range entry.Tokens {
		switch token.Type {
		case hclsyntax.TokenOBrace:
			braceLevel++
			hasNestedStructure = true
		case hclsyntax.TokenCBrace:
			braceLevel--
		case hclsyntax.TokenOBrack:
			bracketLevel++
			hasNestedStructure = true
		case hclsyntax.TokenCBrack:
			bracketLevel--
		case hclsyntax.TokenNewline:
			lineCount++
		}
	}
	
	// Multi-line if it has nested structures or spans multiple lines
	return hasNestedStructure || lineCount > 0
}

// rebuildObjectTokens rebuilds the object with sorted entries
func (s *Sorter) rebuildObjectTokens(tokens hclwrite.Tokens, entries []ObjectEntry, openBraceIdx int, needsSeparatorLine bool) hclwrite.Tokens {
	result := make(hclwrite.Tokens, 0, len(tokens))
	
	// Copy everything up to and including the opening brace
	for i := 0; i <= openBraceIdx; i++ {
		result = append(result, tokens[i])
	}
	
	// Check if we need a newline after opening brace
	needNewline := false
	if openBraceIdx+1 < len(tokens) && tokens[openBraceIdx+1].Type == hclsyntax.TokenNewline {
		result = append(result, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
		needNewline = true
	}
	
	// Add sorted entries
	singleLineCount := 0
	for _, entry := range entries {
		if !s.isMultiLineObjectEntry(entry) {
			singleLineCount++
		} else {
			break
		}
	}
	
	for i, entry := range entries {
		isMultiLine := s.isMultiLineObjectEntry(entry)
		
		// Add separator line between single-line and multi-line groups
		if needsSeparatorLine && i == singleLineCount && singleLineCount > 0 {
			result = append(result, &hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
			})
		}
		
		// Add proper indentation for multiline formatting
		if needNewline && len(entry.Tokens) > 0 {
			// Preserve or add proper indentation
			firstToken := entry.Tokens[0]
			if firstToken.SpacesBefore == 0 {
				firstToken = &hclwrite.Token{
					Type:         firstToken.Type,
					Bytes:        firstToken.Bytes,
					SpacesBefore: 2, // Standard indentation
				}
				result = append(result, firstToken)
				result = append(result, entry.Tokens[1:]...)
			} else {
				result = append(result, entry.Tokens...)
			}
		} else {
			result = append(result, entry.Tokens...)
		}
		
		// Add newline after entry
		if i < len(entries)-1 && needNewline {
			result = append(result, &hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
			})
			
			// Only add extra blank line between multi-line entries or when transitioning from single to multi
			nextIsMultiLine := s.isMultiLineObjectEntry(entries[i+1])
			if (isMultiLine && nextIsMultiLine) || (!isMultiLine && nextIsMultiLine && i >= singleLineCount-1) {
				result = append(result, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
		}
	}
	
	// Find and add the closing brace and any trailing content
	closeBraceIdx := s.findClosingBrace(tokens, openBraceIdx)
	if closeBraceIdx >= 0 {
		// Add newline before closing brace if needed
		if needNewline && len(entries) > 0 {
			result = append(result, &hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
			})
		}
		
		// Add closing brace and everything after
		for i := closeBraceIdx; i < len(tokens); i++ {
			result = append(result, tokens[i])
		}
	}
	
	return result
}

// findClosingBrace finds the matching closing brace for an opening brace
func (s *Sorter) findClosingBrace(tokens hclwrite.Tokens, openBraceIdx int) int {
	braceLevel := 1
	for i := openBraceIdx + 1; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenOBrace {
			braceLevel++
		} else if tokens[i].Type == hclsyntax.TokenCBrace {
			braceLevel--
			if braceLevel == 0 {
				return i
			}
		}
	}
	return -1
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

// getDynamicContentSortKey extracts sorting key from dynamic block content (id, label, or name)
func (s *Sorter) getDynamicContentSortKey(block *hclwrite.Block) string {
	if block.Type() != "dynamic" {
		return ""
	}

	body := block.Body()
	contentBlocks := body.Blocks()

	// Find the content block
	for _, contentBlock := range contentBlocks {
		if contentBlock.Type() == "content" {
			contentBody := contentBlock.Body()
			contentAttrs := contentBody.Attributes()

			// Check for sorting attributes in priority order: id, label, name
			for _, attrName := range []string{"id", "label", "name"} {
				if attr, exists := contentAttrs[attrName]; exists {
					// Convert the expression to string for comparison
					tokens := attr.Expr().BuildTokens(nil)
					var content strings.Builder
					for _, token := range tokens {
						content.Write(token.Bytes)
					}
					return content.String()
				}
			}
		}
	}

	return ""
}

// isArrayLiteral checks if tokens represent an array literal [ ... ]
func (s *Sorter) isArrayLiteral(tokens hclwrite.Tokens) bool {
	if len(tokens) < 2 {
		return false
	}
	
	// Find first non-whitespace token
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenNewline || token.Type == hclsyntax.TokenComment {
			continue
		}
		return token.Type == hclsyntax.TokenOBrack
	}
	
	return false
}

// sortArrayLiteral sorts elements in an array literal
func (s *Sorter) sortArrayLiteral(tokens hclwrite.Tokens) hclwrite.Tokens {
	// Find the opening bracket
	openBracketIdx := -1
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenOBrack {
			openBracketIdx = i
			break
		}
	}
	
	if openBracketIdx == -1 {
		return nil
	}
	
	// Parse the array elements
	elements := s.parseArrayElements(tokens, openBracketIdx)
	if len(elements) == 0 {
		return nil
	}
	
	// Sort elements alphabetically
	sort.Slice(elements, func(i, j int) bool {
		return elements[i].Value < elements[j].Value
	})
	
	// Rebuild the tokens
	return s.rebuildArrayTokens(tokens, elements, openBracketIdx)
}

type ArrayElement struct {
	Value    string
	Tokens   hclwrite.Tokens
	StartIdx int
	EndIdx   int
}

// parseArrayElements parses elements from array tokens
func (s *Sorter) parseArrayElements(tokens hclwrite.Tokens, startIdx int) []ArrayElement {
	var elements []ArrayElement
	
	i := startIdx + 1 // Skip opening bracket
	bracketLevel := 1
	
	for i < len(tokens) && bracketLevel > 0 {
		// Skip whitespace and comments
		for i < len(tokens) && (tokens[i].Type == hclsyntax.TokenNewline || tokens[i].Type == hclsyntax.TokenComment) {
			i++
		}
		
		if i >= len(tokens) {
			break
		}
		
		// Track bracket level
		if tokens[i].Type == hclsyntax.TokenOBrack {
			bracketLevel++
		} else if tokens[i].Type == hclsyntax.TokenCBrack {
			bracketLevel--
			if bracketLevel == 0 {
				break
			}
		}
		
		// Look for array elements at top level
		if bracketLevel == 1 && s.isArrayElementStart(tokens[i]) {
			element := s.parseArrayElement(tokens, i)
			if element != nil {
				elements = append(elements, *element)
				i = element.EndIdx
			} else {
				i++
			}
		} else {
			i++
		}
	}
	
	return elements
}

// isArrayElementStart checks if a token could be the start of an array element
func (s *Sorter) isArrayElementStart(token *hclwrite.Token) bool {
	return token.Type == hclsyntax.TokenIdent || 
		   token.Type == hclsyntax.TokenQuotedLit || 
		   token.Type == hclsyntax.TokenNumberLit ||
		   token.Type == hclsyntax.TokenOBrace ||
		   token.Type == hclsyntax.TokenOBrack
}

// parseArrayElement parses a single array element starting at the given index
func (s *Sorter) parseArrayElement(tokens hclwrite.Tokens, startIdx int) *ArrayElement {
	if startIdx >= len(tokens) {
		return nil
	}
	
	// Find the end of this element
	endIdx := s.findArrayElementEnd(tokens, startIdx)
	if endIdx == -1 {
		return nil
	}
	
	// Extract all tokens for this element
	elementTokens := make(hclwrite.Tokens, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		elementTokens = append(elementTokens, tokens[i])
	}
	
	// Extract the value for sorting comparison
	value := s.extractArrayElementValue(elementTokens)
	
	return &ArrayElement{
		Value:    value,
		Tokens:   elementTokens,
		StartIdx: startIdx,
		EndIdx:   endIdx,
	}
}

// findArrayElementEnd finds the end index of an array element
func (s *Sorter) findArrayElementEnd(tokens hclwrite.Tokens, startIdx int) int {
	braceLevel := 0
	bracketLevel := 0
	parenLevel := 0
	
	for i := startIdx; i < len(tokens); i++ {
		token := tokens[i]
		
		switch token.Type {
		case hclsyntax.TokenOBrace:
			braceLevel++
		case hclsyntax.TokenCBrace:
			braceLevel--
		case hclsyntax.TokenOBrack:
			bracketLevel++
		case hclsyntax.TokenCBrack:
			if bracketLevel > 0 {
				bracketLevel--
			} else {
				// End of array
				return i
			}
		case hclsyntax.TokenOParen:
			parenLevel++
		case hclsyntax.TokenCParen:
			parenLevel--
		case hclsyntax.TokenComma:
			// If we're at the top level, comma ends the element
			if braceLevel == 0 && bracketLevel == 0 && parenLevel == 0 {
				return i + 1
			}
		}
	}
	
	return len(tokens)
}

// extractArrayElementValue extracts a comparable value from array element tokens
func (s *Sorter) extractArrayElementValue(tokens hclwrite.Tokens) string {
	if len(tokens) == 0 {
		return ""
	}
	
	// For simple values, use the first meaningful token
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenNewline || token.Type == hclsyntax.TokenComment {
			continue
		}
		
		value := string(token.Bytes)
		
		// Remove quotes for string literals
		if token.Type == hclsyntax.TokenQuotedLit && len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			return value[1 : len(value)-1]
		}
		
		return value
	}
	
	return ""
}

// rebuildArrayTokens rebuilds the array with sorted elements
func (s *Sorter) rebuildArrayTokens(tokens hclwrite.Tokens, elements []ArrayElement, openBracketIdx int) hclwrite.Tokens {
	result := make(hclwrite.Tokens, 0, len(tokens))
	
	// Copy everything up to and including the opening bracket
	for i := 0; i <= openBracketIdx; i++ {
		result = append(result, tokens[i])
	}
	
	// Check if we need a newline after opening bracket
	needNewline := false
	if openBracketIdx+1 < len(tokens) && tokens[openBracketIdx+1].Type == hclsyntax.TokenNewline {
		result = append(result, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
		needNewline = true
	}
	
	// Add sorted elements
	for i, element := range elements {
		// Add proper indentation for multiline formatting
		if needNewline && len(element.Tokens) > 0 {
			// Preserve or add proper indentation
			firstToken := element.Tokens[0]
			if firstToken.SpacesBefore == 0 {
				firstToken = &hclwrite.Token{
					Type:         firstToken.Type,
					Bytes:        firstToken.Bytes,
					SpacesBefore: 2, // Standard indentation
				}
				result = append(result, firstToken)
				result = append(result, element.Tokens[1:]...)
			} else {
				result = append(result, element.Tokens...)
			}
		} else {
			result = append(result, element.Tokens...)
		}
		
		// Add comma if not the last element
		if i < len(elements)-1 {
			result = append(result, &hclwrite.Token{
				Type:  hclsyntax.TokenComma,
				Bytes: []byte(","),
			})
		}
		
		// Add newline after element
		if needNewline {
			result = append(result, &hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
			})
		}
	}
	
	// Find and add the closing bracket and any trailing content
	closeBracketIdx := s.findClosingBracket(tokens, openBracketIdx)
	if closeBracketIdx >= 0 {
		// Add newline before closing bracket if needed
		if needNewline && len(elements) > 0 {
			result = append(result, &hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
			})
		}
		
		// Add closing bracket and everything after
		for i := closeBracketIdx; i < len(tokens); i++ {
			result = append(result, tokens[i])
		}
	}
	
	return result
}

// findClosingBracket finds the matching closing bracket for an opening bracket
func (s *Sorter) findClosingBracket(tokens hclwrite.Tokens, openBracketIdx int) int {
	bracketLevel := 1
	for i := openBracketIdx + 1; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenOBrack {
			bracketLevel++
		} else if tokens[i].Type == hclsyntax.TokenCBrack {
			bracketLevel--
			if bracketLevel == 0 {
				return i
			}
		}
	}
	return -1
}
