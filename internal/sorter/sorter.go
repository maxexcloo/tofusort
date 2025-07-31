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
	"count":            0,
	"for_each":         1,
	"depends_on":       998,
	"force_new":        999,
	"lifecycle":        1000,
	"triggers_replace": 1001,
}

func (s *Sorter) SortFile(file *hclwrite.File) {
	body := file.Body()
	blocks := body.Blocks()
	attrs := body.Attributes()

	// If there are no blocks, this is likely a .tfvars file with only attributes
	if len(blocks) == 0 {
		if len(attrs) > 0 {
			s.sortBodyAttributes(body)
		}
		return
	}

	// If there are blocks (and possibly attributes), rebuild the entire body structure
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

	// Sort attributes if they exist
	var sortedAttrs []AttrInfo
	if len(attrs) > 0 {
		for name, attr := range attrs {
			expr := attr.Expr()
			sortedExpr := s.sortExpression(expr)
			isMultiLine := s.isMultiLineAttribute(sortedExpr)

			sortedAttrs = append(sortedAttrs, AttrInfo{
				Name:        name,
				Expr:        sortedExpr,
				IsMultiLine: isMultiLine,
			})
		}

		sort.Slice(sortedAttrs, func(i, j int) bool {
			return sortedAttrs[i].Name < sortedAttrs[j].Name
		})
	}

	// Clear and rebuild the entire body
	body.Clear()

	// Add sorted attributes first (if any)
	if len(sortedAttrs) > 0 {
		s.writeAttributeGroup(body, sortedAttrs)
		// Add blank line between attributes and blocks
		if len(blockInfos) > 0 {
			body.AppendNewline()
		}
	}

	// Add sorted blocks
	for i, blockInfo := range blockInfos {
		s.sortBlockAttributes(blockInfo.Block)

		// Add blank line before certain block types for grouping
		if i > 0 {
			currentOrder, currentExists := blockTypeOrder[blockInfo.Type]
			prevOrder, prevExists := blockTypeOrder[blockInfos[i-1].Type]

			// Add blank line in specific cases only
			if currentExists && prevExists {
				// Add blank line when transitioning from early group (0-2) to later group (3+)
				if prevOrder <= 2 && currentOrder >= 3 {
					body.AppendNewline()
				}
				// Add blank line between all blocks in the later group (3+)
				if prevOrder >= 3 && currentOrder >= 3 {
					body.AppendNewline()
				}
				// NO blank lines between terraform/provider/variable (both <= 2) - do nothing
			} else if !currentExists || !prevExists {
				// Add blank line for unknown block types
				body.AppendNewline()
			}
		}

		body.AppendBlock(blockInfo.Block)

		// Always add a newline after each block
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
	var singleLineAttrs []AttrInfo
	var multiLineAttrs []AttrInfo
	var lateAttrs []AttrInfo

	for name, attr := range attrs {
		expr := attr.Expr()
		// DISABLED: Expression sorting causes corruption - keep original expressions
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
		} else if isMultiLine {
			multiLineAttrs = append(multiLineAttrs, attrInfo)
		} else {
			singleLineAttrs = append(singleLineAttrs, attrInfo)
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
	sort.Slice(singleLineAttrs, func(i, j int) bool {
		return singleLineAttrs[i].Name < singleLineAttrs[j].Name
	})
	sort.Slice(multiLineAttrs, func(i, j int) bool {
		return multiLineAttrs[i].Name < multiLineAttrs[j].Name
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
	hasOtherContent := len(singleLineAttrs) > 0 || len(multiLineAttrs) > 0 || len(lateAttrs) > 0 ||
		len(regularBlocks) > 0 || len(lifecycleBlocks) > 0
	if len(earlyAttrs) > 0 && hasOtherContent {
		body.AppendNewline()
	}

	// 2. Single-line regular attributes
	s.writeAttributeGroup(body, singleLineAttrs)

	// 3. Regular nested blocks (not lifecycle) - recursively sort them
	for i, block := range regularBlocks {
		// Add blank line before blocks if we have attributes or previous blocks
		if len(singleLineAttrs) > 0 || i > 0 {
			body.AppendNewline()
		}
		s.sortBlockAttributes(block)
		body.AppendBlock(block)
	}

	// 4. Multi-line regular attributes
	if len(multiLineAttrs) > 0 {
		// Add blank line before multi-line attributes if we have regular content
		if len(singleLineAttrs) > 0 || len(regularBlocks) > 0 {
			body.AppendNewline()
		}
		s.writeAttributeGroup(body, multiLineAttrs)
	}

	// 5. Late meta-arguments (depends_on attributes)
	if len(lateAttrs) > 0 {
		// Add blank line before late attributes if we have regular content
		if len(singleLineAttrs) > 0 || len(regularBlocks) > 0 || len(multiLineAttrs) > 0 {
			body.AppendNewline()
		}
		s.writeAttributeGroup(body, lateAttrs)
	}

	// 6. Late blocks (lifecycle) - recursively sort them
	for _, block := range lifecycleBlocks {
		// Add blank line before lifecycle blocks
		if len(singleLineAttrs) > 0 || len(regularBlocks) > 0 || len(multiLineAttrs) > 0 || len(lateAttrs) > 0 {
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

		// Sort the expression content if it's an object or similar
		sortedExpr := s.sortExpression(expr)

		isMultiLine := s.isMultiLineAttribute(sortedExpr)

		attrInfos = append(attrInfos, AttrInfo{
			Name:        name,
			Expr:        sortedExpr,
			IsMultiLine: isMultiLine,
		})
	}

	// Sort attributes alphabetically
	sort.Slice(attrInfos, func(i, j int) bool {
		return attrInfos[i].Name < attrInfos[j].Name
	})

	// Remove existing attributes first
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
	for i, attrInfo := range multiLineAttrs {
		// Add blank line before multi-line attributes (except the first if no single-line attrs)
		if len(singleLineAttrs) > 0 || i > 0 {
			body.AppendNewline()
		}
		body.SetAttributeRaw(attrInfo.Name, attrInfo.Expr.BuildTokens(nil))
	}
}

// sortExpression attempts to sort object expressions using token-based approach
func (s *Sorter) sortExpression(expr *hclwrite.Expression) *hclwrite.Expression {
	tokens := expr.BuildTokens(nil)

	// Parse the expression using HCL's AST parser to understand its structure
	if s.isSimpleObjectExpression(tokens) {
		// Only sort if it's a simple object with literal values
		sortedTokens := s.sortObjectLiteral(tokens)
		if sortedTokens != nil {
			return s.tokensToExpression(sortedTokens)
		}
	}

	// Also check if this is an array that might contain objects
	if len(tokens) > 0 && tokens[0].Type == hclsyntax.TokenOBrack {
		// This is an array, process its contents
		sortedTokens := s.sortArrayContents(tokens)
		if sortedTokens != nil {
			return s.tokensToExpression(sortedTokens)
		}
	}

	return expr
}

// isSimpleObjectExpression uses HCL AST parsing to determine if an expression is a simple object
func (s *Sorter) isSimpleObjectExpression(tokens hclwrite.Tokens) bool {
	if len(tokens) < 3 {
		return false
	}

	// Convert tokens back to source code
	var src strings.Builder
	for _, token := range tokens {
		src.Write(token.Bytes)
	}
	sourceText := src.String()

	// Check for complex expressions that should never be sorted
	if strings.Contains(sourceText, "for ") ||
		strings.Contains(sourceText, "${") ||
		strings.Contains(sourceText, "=>") {
		return false
	}

	// Check if it's an object() type definition - these should be sorted
	if strings.HasPrefix(strings.TrimSpace(sourceText), "object(") {
		return true
	}

	// Check if it's inside a function call like merge() - these should not be sorted
	// This is a simple heuristic - if we see function names, don't sort
	if strings.Contains(sourceText, "merge(") ||
		strings.Contains(sourceText, "concat(") ||
		strings.Contains(sourceText, "flatten(") {
		return false
	}

	// Only allow simple object literals like { key = value, ... }
	return strings.HasPrefix(strings.TrimSpace(sourceText), "{") &&
		strings.HasSuffix(strings.TrimSpace(sourceText), "}")
}

// isSimpleExpression recursively checks if an HCL expression contains only simple, literal values
//
//nolint:unused // Used recursively but linter doesn't detect it
func (s *Sorter) isSimpleExpression(expr hclsyntax.Expression) bool {
	switch e := expr.(type) {
	case *hclsyntax.ObjectConsExpr:
		// Object constructor { key = value, ... }
		for _, item := range e.Items {
			// Check that keys are simple (literals or identifiers)
			if !s.isSimpleExpression(item.KeyExpr) {
				return false
			}
			// Check that values are simple
			if !s.isSimpleExpression(item.ValueExpr) {
				return false
			}
		}
		return true

	case *hclsyntax.TupleConsExpr:
		// Array constructor [item1, item2, ...]
		for _, elem := range e.Exprs {
			if !s.isSimpleExpression(elem) {
				return false
			}
		}
		return true

	case *hclsyntax.LiteralValueExpr:
		// Literal values (strings, numbers, booleans, null)
		return true

	case *hclsyntax.ScopeTraversalExpr:
		// Variable references and identifiers are simple (like "string", "number", "bool" in type definitions)
		return true

	case *hclsyntax.ObjectConsKeyExpr:
		// Object keys (identifiers or literals used as keys)
		return true

	case *hclsyntax.TemplateExpr:
		// Template expressions like quoted strings
		// Only allow simple literal templates (no interpolation)
		return e.IsStringLiteral()

	case *hclsyntax.FunctionCallExpr:
		// Only allow specific HCL type constructor functions
		allowedTypeFunctions := map[string]bool{
			"object": true,
			"list":   true,
			"set":    true,
			"map":    true,
			"tuple":  true,
		}

		if !allowedTypeFunctions[e.Name] {
			return false
		}

		// Check arguments are simple
		for _, arg := range e.Args {
			if !s.isSimpleExpression(arg) {
				return false
			}
		}
		return true

	default:
		// Any other expression type (for loops, conditionals, etc.) is not simple
		return false
	}
}

// tokensToExpression converts tokens back to an expression by creating a temporary attribute
func (s *Sorter) tokensToExpression(tokens hclwrite.Tokens) *hclwrite.Expression {
	// Create a temporary body and set an attribute with the tokens
	body := hclwrite.NewEmptyFile().Body()
	body.SetAttributeRaw("temp", tokens)

	// Extract the expression from the temporary attribute
	attrs := body.Attributes()
	if tempAttr, exists := attrs["temp"]; exists {
		return tempAttr.Expr()
	}

	return nil
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
	Key      string
	Tokens   hclwrite.Tokens
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

	// Look for nested objects and arrays and sort them
	result := make(hclwrite.Tokens, 0, len(tokens))
	i := 0

	for i < len(tokens) {
		switch tokens[i].Type {
		case hclsyntax.TokenOBrace:
			// Found a nested object, extract and sort it
			objEnd := s.findMatchingBrace(tokens, i)
			if objEnd > i {
				// Extract the object tokens
				objTokens := tokens[i : objEnd+1]

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
		case hclsyntax.TokenOBrack:
			// Found an array, process its contents for nested objects
			arrEnd := s.findMatchingBracket(tokens, i)
			if arrEnd > i {
				// Process array contents
				arrayTokens := s.sortArrayContents(tokens[i : arrEnd+1])
				result = append(result, arrayTokens...)
				i = arrEnd + 1
			} else {
				result = append(result, tokens[i])
				i++
			}
		default:
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
		switch tokens[i].Type {
		case hclsyntax.TokenOBrace:
			braceLevel++
		case hclsyntax.TokenCBrace:
			braceLevel--
			if braceLevel == 0 {
				return i
			}
		}
	}

	return -1
}

// findMatchingBracket finds the closing bracket that matches the opening bracket at the given index
func (s *Sorter) findMatchingBracket(tokens hclwrite.Tokens, openIdx int) int {
	if openIdx >= len(tokens) || tokens[openIdx].Type != hclsyntax.TokenOBrack {
		return -1
	}

	bracketLevel := 1
	for i := openIdx + 1; i < len(tokens); i++ {
		switch tokens[i].Type {
		case hclsyntax.TokenOBrack:
			bracketLevel++
		case hclsyntax.TokenCBrack:
			bracketLevel--
			if bracketLevel == 0 {
				return i
			}
		}
	}

	return -1
}

// sortArrayContents processes array contents and sorts any objects within the array
func (s *Sorter) sortArrayContents(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) < 3 { // Need at least [ content ]
		return tokens
	}

	result := make(hclwrite.Tokens, 0, len(tokens))
	result = append(result, tokens[0]) // Opening bracket

	// Process the content between brackets
	contentTokens := tokens[1 : len(tokens)-1]
	contentResult := s.recursiveSortTokens(contentTokens)
	result = append(result, contentResult...)

	result = append(result, tokens[len(tokens)-1]) // Closing bracket
	return result
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
	// Find the last non-newline token
	lastNonNewlineIdx := -1
	for i := len(entry.Tokens) - 1; i >= 0; i-- {
		if entry.Tokens[i].Type != hclsyntax.TokenNewline {
			lastNonNewlineIdx = i
			break
		}
	}

	// Count newlines that appear before the last non-newline token (internal newlines)
	internalNewlines := 0
	for i := 0; i <= lastNonNewlineIdx; i++ {
		token := entry.Tokens[i]
		if token.Type == hclsyntax.TokenNewline {
			internalNewlines++
		}
	}

	// Multi-line ONLY if there are newlines within the value content
	// Presence of brackets/braces alone doesn't make it multi-line if everything is on one line
	// This allows single-line arrays like ["a", "b"] and objects like { key = "value" }
	// to be grouped with other single-line entries
	return internalNewlines > 0
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

	// Count single-line entries (they all come first due to sorting)
	singleLineCount := 0
	for _, entry := range entries {
		if s.isMultiLineObjectEntry(entry) {
			break // Found first multi-line entry, all remaining are multi-line
		}
		singleLineCount++
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
			// Clean up leading and trailing newlines to ensure proper spacing
			cleanedTokens := s.cleanLeadingAndTrailingNewlines(entry.Tokens)

			// Preserve or add proper indentation
			if len(cleanedTokens) > 0 {
				firstToken := cleanedTokens[0]
				if firstToken.SpacesBefore == 0 {
					firstToken = &hclwrite.Token{
						Type:         firstToken.Type,
						Bytes:        firstToken.Bytes,
						SpacesBefore: 2, // Standard indentation
					}
					result = append(result, firstToken)
					result = append(result, cleanedTokens[1:]...)
				} else {
					result = append(result, cleanedTokens...)
				}
			}
		} else {
			// Clean up leading and trailing newlines to ensure proper spacing
			cleanedTokens := s.cleanLeadingAndTrailingNewlines(entry.Tokens)
			result = append(result, cleanedTokens...)
		}

		// Only add blank lines where needed, don't add regular newlines as entry tokens should handle that
		if i < len(entries)-1 {
			nextIsMultiLine := s.isMultiLineObjectEntry(entries[i+1])

			// Add blank line between multi-line entries ONLY
			// Single-line entries should be grouped together without blank lines
			shouldAddBlankLine := isMultiLine && nextIsMultiLine

			if shouldAddBlankLine {
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
		// Don't add extra newline before closing brace as entries already have newlines

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
		switch tokens[i].Type {
		case hclsyntax.TokenOBrace:
			braceLevel++
		case hclsyntax.TokenCBrace:
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

// cleanTrailingNewlines removes extra trailing newlines, keeping only one
// cleanLeadingAndTrailingNewlines removes leading and trailing newlines from tokens
// This ensures that entries don't have unwanted blank lines around them
func (s *Sorter) cleanLeadingAndTrailingNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) == 0 {
		return tokens
	}

	// Find first non-newline token
	firstNonNewlineIdx := -1
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type != hclsyntax.TokenNewline {
			firstNonNewlineIdx = i
			break
		}
	}

	if firstNonNewlineIdx == -1 {
		// All tokens are newlines, keep just one
		return hclwrite.Tokens{&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		}}
	}

	// Find last non-newline token
	lastNonNewlineIdx := -1
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Type != hclsyntax.TokenNewline {
			lastNonNewlineIdx = i
			break
		}
	}

	// Extract the tokens between first and last non-newline (inclusive) and add exactly one trailing newline
	result := make(hclwrite.Tokens, 0, lastNonNewlineIdx-firstNonNewlineIdx+2)
	result = append(result, tokens[firstNonNewlineIdx:lastNonNewlineIdx+1]...)
	result = append(result, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})

	return result
}
