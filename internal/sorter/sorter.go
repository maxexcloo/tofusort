package sorter

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
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

	// For .tfvars files, clear the body entirely and rebuild clean
	body.Clear()

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
	// TEMPORARILY DISABLED: Object sorting is causing corruption with complex expressions
	// TODO: Re-implement with safer token reconstruction
	return false

	/*
		// Convert tokens back to source code
		var src strings.Builder
		for _, token := range tokens {
			src.Write(token.Bytes)
		}
		sourceText := src.String()

		// Check for complex expressions that are prone to corruption in token reconstruction
		// These include: quoted keys, complex expressions, for loops, function calls, interpolations
		complexPatterns := []string{
			`"[^"]*"\s*[=:]`,           // Quoted keys like "/" = "*"
			`\$\{`,                     // String interpolations like ${each.value}
			`\bfor\b.*\bin\b`,          // For loop expressions
			`\bone\(`,                  // Function calls like one()
			`\bsort\(`,                 // Function calls like sort()
			`\bdata\.`,                 // Data source references
			`\bvar\.`,                  // Variable references in complex contexts
			`\blocal\.`,                // Local references in complex contexts
			`\beach\.`,                 // Each references in complex contexts
			`\bif\b.*\?.*:`,           // Ternary conditionals
			`\[\s*[^]]*\s*:\s*[^]]*\s*\]`, // Array/map comprehensions
		}

		for _, pattern := range complexPatterns {
			if matched, _ := regexp.MatchString(pattern, sourceText); matched {
				return false
			}
		}

		// Parse the expression using HCL's parser
		expr, diags := hclsyntax.ParseExpression([]byte(sourceText), "", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return false
		}

		// Check if it's a simple object constructor or object literal
		return s.isSimpleExpression(expr)
	*/
}

// isSimpleExpression recursively checks if an HCL expression contains only simple, literal values
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

// isObjectCall checks if tokens represent an object(...) call
func (s *Sorter) isObjectCall(tokens hclwrite.Tokens) bool {
	if len(tokens) < 4 {
		return false
	}

	// Skip whitespace and comments
	i := 0
	for i < len(tokens) && (tokens[i].Type == hclsyntax.TokenNewline || tokens[i].Type == hclsyntax.TokenComment) {
		i++
	}

	if i >= len(tokens) || tokens[i].Type != hclsyntax.TokenIdent || string(tokens[i].Bytes) != "object" {
		return false
	}
	i++

	// Skip whitespace
	for i < len(tokens) && (tokens[i].Type == hclsyntax.TokenNewline || tokens[i].Type == hclsyntax.TokenComment) {
		i++
	}

	return i < len(tokens) && tokens[i].Type == hclsyntax.TokenOParen
}

// isSafeObjectCall validates that an object() call is safe to sort
func (s *Sorter) isSafeObjectCall(tokens hclwrite.Tokens) bool {
	// Convert tokens to text for analysis
	var source strings.Builder
	for _, token := range tokens {
		source.Write(token.Bytes)
	}
	text := source.String()

	// Reject any complex expressions that could be corrupted
	unsafePatterns := []string{
		"${",            // String interpolation
		"for ",          // For expressions
		" for ",         // For expressions (with space)
		"if ",           // Conditional expressions
		" if ",          // Conditional expressions (with space)
		"templatefile(", // Function calls
		"merge(",        // Function calls
		"try(",          // Function calls
		"length(",       // Function calls
		"split(",        // Function calls
		"join(",         // Function calls
		"contains(",     // Function calls
		"keys(",         // Function calls
		"values(",       // Function calls
		"concat(",       // Function calls
		"flatten(",      // Function calls
		"distinct(",     // Function calls
		"reverse(",      // Function calls
		"sort(",         // Function calls
		"zipmap(",       // Function calls
		"toset(",        // Function calls
		"tolist(",       // Function calls
		"tomap(",        // Function calls
		"jsondecode(",   // Function calls
		"jsonencode(",   // Function calls
		"yamldecode(",   // Function calls
		"yamlencode(",   // Function calls
		"base64decode(", // Function calls
		"base64encode(", // Function calls
		"replace(",      // Function calls
		"can(",          // Function calls
		"upper(",        // Function calls
		"lower(",        // Function calls
		"title(",        // Function calls
		"substr(",       // Function calls
		"regex(",        // Function calls
		"cidrhost(",     // Function calls
		"cidrnetmask(",  // Function calls
		"endswith(",     // Function calls
		"startswith(",   // Function calls
	}

	for _, pattern := range unsafePatterns {
		if strings.Contains(text, pattern) {
			return false
		}
	}

	// Only allow simple object() calls with basic types
	// Must contain only: object({ key = type, ... })
	// where type is: string, number, bool, any, list(...), map(...), set(...), or nested object(...)

	// Check for any function calls other than object()
	// Count all opening parentheses
	objectCount := strings.Count(text, "object(")
	listCount := strings.Count(text, "list(")
	mapCount := strings.Count(text, "map(")
	setCount := strings.Count(text, "set(")
	parenthesesCount := strings.Count(text, "(")

	// Allow only object(), list(), map(), set() calls (HCL type constructors)
	allowedCallsCount := objectCount + listCount + mapCount + setCount

	// If there are more opening parentheses than allowed type constructor calls,
	// there are other function calls - reject for safety
	if parenthesesCount > allowedCallsCount {
		return false
	}

	return true
}

// isSafeObjectLiteral validates that an object literal is safe to sort
func (s *Sorter) isSafeObjectLiteral(tokens hclwrite.Tokens) bool {
	// Convert tokens to text for analysis
	var source strings.Builder
	for _, token := range tokens {
		source.Write(token.Bytes)
	}
	text := source.String()

	// Use the same safety patterns as object() calls
	unsafePatterns := []string{
		"${",            // String interpolation
		"for ",          // For expressions
		" for ",         // For expressions (with space)
		"if ",           // Conditional expressions
		" if ",          // Conditional expressions (with space)
		"templatefile(", // Function calls
		"merge(",        // Function calls
		"try(",          // Function calls
		"length(",       // Function calls
		"split(",        // Function calls
		"join(",         // Function calls
		"contains(",     // Function calls
		"keys(",         // Function calls
		"values(",       // Function calls
		"concat(",       // Function calls
		"flatten(",      // Function calls
		"distinct(",     // Function calls
		"reverse(",      // Function calls
		"sort(",         // Function calls
		"zipmap(",       // Function calls
		"toset(",        // Function calls
		"tolist(",       // Function calls
		"tomap(",        // Function calls
		"jsondecode(",   // Function calls
		"jsonencode(",   // Function calls
		"yamldecode(",   // Function calls
		"yamlencode(",   // Function calls
		"base64decode(", // Function calls
		"base64encode(", // Function calls
		"replace(",      // Function calls
		"can(",          // Function calls
		"upper(",        // Function calls
		"lower(",        // Function calls
		"title(",        // Function calls
		"substr(",       // Function calls
		"regex(",        // Function calls
		"cidrhost(",     // Function calls
		"cidrnetmask(",  // Function calls
		"endswith(",     // Function calls
		"startswith(",   // Function calls
	}

	for _, pattern := range unsafePatterns {
		if strings.Contains(text, pattern) {
			return false
		}
	}

	// Only allow very simple object literals
	// Must be just key-value pairs with basic types (strings, numbers, booleans, arrays, nested objects)

	// Reject any parentheses (function calls, expressions)
	if strings.Contains(text, "(") {
		return false
	}

	// Reject any equals arrows (for expressions use =>)
	if strings.Contains(text, "=>") {
		return false
	}

	// Reject any complex operators
	complexOperators := []string{
		" : ", // Object key syntax in for expressions (with spaces)
		": ",  // Object key syntax in for expressions (space after)
		" :",  // Object key syntax in for expressions (space before)
		"=>",  // For expressions arrow (already checked above, but double-check)
		"...", // Spread operator
		"?",   // Conditional operator
		"||",  // Logical operators
		"&&",  // Logical operators
	}

	for _, op := range complexOperators {
		if strings.Contains(text, op) {
			return false
		}
	}

	return true
}

// sortObjectCallTokens sorts the content inside an object() call
func (s *Sorter) sortObjectCallTokens(tokens hclwrite.Tokens) hclwrite.Tokens {
	// Find the opening parenthesis
	parenIdx := -1
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenOParen {
			parenIdx = i
			break
		}
	}

	if parenIdx == -1 {
		return nil
	}

	// Find the matching closing parenthesis
	parenLevel := 0
	closeParenIdx := -1
	for i := parenIdx; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenOParen {
			parenLevel++
		} else if tokens[i].Type == hclsyntax.TokenCParen {
			parenLevel--
			if parenLevel == 0 {
				closeParenIdx = i
				break
			}
		}
	}

	if closeParenIdx == -1 {
		return nil
	}

	// Extract the content between parentheses
	innerTokens := tokens[parenIdx+1 : closeParenIdx]

	// Check if the inner content is an object literal
	if s.isObjectLiteral(innerTokens) {
		sortedInner := s.sortObjectLiteral(innerTokens)
		if sortedInner != nil {
			// Rebuild the object() call with sorted content
			var result hclwrite.Tokens
			result = append(result, tokens[:parenIdx+1]...)    // "object("
			result = append(result, sortedInner...)            // sorted object content
			result = append(result, tokens[closeParenIdx:]...) // ")"
			return result
		}
	}

	return nil
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
		if tokens[i].Type == hclsyntax.TokenOBrace {
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
		} else if tokens[i].Type == hclsyntax.TokenOBrack {
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

// findMatchingBracket finds the closing bracket that matches the opening bracket at the given index
func (s *Sorter) findMatchingBracket(tokens hclwrite.Tokens, openIdx int) int {
	if openIdx >= len(tokens) || tokens[openIdx].Type != hclsyntax.TokenOBrack {
		return -1
	}

	bracketLevel := 1
	for i := openIdx + 1; i < len(tokens); i++ {
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

// isSimpleObjectLiteral checks if tokens represent a simple object literal without complex expressions
func (s *Sorter) isSimpleObjectLiteral(tokens hclwrite.Tokens) bool {
	if len(tokens) < 2 {
		return false
	}

	// Must start with { and end with }
	start := -1
	end := -1

	for i, token := range tokens {
		if token.Type == hclsyntax.TokenOBrace && start == -1 {
			start = i
		}
		if token.Type == hclsyntax.TokenCBrace {
			end = i
		}
	}

	if start == -1 || end == -1 || start >= end {
		return false
	}

	// Check if there are any function calls, complex expressions, or interpolations
	// that could cause corruption when we manipulate tokens
	inString := false
	stringDelim := byte(0)

	for i := start + 1; i < end; i++ {
		token := tokens[i]
		tokenBytes := token.Bytes

		if len(tokenBytes) == 0 {
			continue
		}

		// Track string state
		if !inString && (tokenBytes[0] == '"' || tokenBytes[0] == '\'') {
			inString = true
			stringDelim = tokenBytes[0]
			continue
		}
		if inString && tokenBytes[0] == stringDelim {
			inString = false
			continue
		}
		if inString {
			continue
		}

		// Avoid complex expressions that could cause corruption
		switch token.Type {
		case hclsyntax.TokenOParen, hclsyntax.TokenCParen:
			// Function calls - avoid
			return false
		case hclsyntax.TokenTemplateInterp, hclsyntax.TokenTemplateControl:
			// String interpolation - avoid
			return false
		}
	}

	return true
}

// extractObjectContent extracts the content between object( and the matching )
func (s *Sorter) extractObjectContent(text string) string {
	if !strings.HasPrefix(text, "object(") {
		return ""
	}

	// Find the matching closing parenthesis
	parenCount := 0
	start := 7 // Length of "object("

	for i := start; i < len(text); i++ {
		if text[i] == '(' {
			parenCount++
		} else if text[i] == ')' {
			if parenCount == 0 {
				// Found the matching closing parenthesis
				return text[start:i]
			}
			parenCount--
		}
	}

	return ""
}

// isMultiLineObjectValue checks if an object value is multi-line
func (s *Sorter) isMultiLineObjectValue(value string) bool {
	value = strings.TrimSpace(value)
	// Check if value contains newlines or is a complex structure
	return strings.Contains(value, "\n") ||
		(strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}")) ||
		(strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]"))
}

// isSimpleObjectLiteralText checks if text represents a simple object literal
func (s *Sorter) isSimpleObjectLiteralText(text string) bool {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") || !strings.HasSuffix(text, "}") {
		return false
	}

	// Avoid complex expressions that could cause issues
	// Allow object() calls but avoid other complex expressions
	if strings.Contains(text, "${") {
		return false
	}

	// Allow object() function calls as they're common in HCL type definitions
	if strings.Contains(text, "(") && !strings.Contains(text, "object(") {
		return false
	}

	return true
}

// sortObjectLiteralText sorts the keys in an object literal text
func (s *Sorter) sortObjectLiteralText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) < 2 {
		return text
	}

	// Remove outer braces
	inner := strings.TrimSpace(text[1 : len(text)-1])
	if inner == "" {
		return text // Empty object
	}

	// Parse key-value pairs
	entries := s.parseObjectEntriesFromText(inner)
	if len(entries) == 0 {
		return text // Parsing failed, return original
	}

	// Separate single-line and multi-line entries
	var singleLineEntries []objectEntry
	var multiLineEntries []objectEntry

	for _, entry := range entries {
		if s.isMultiLineObjectValue(entry.value) {
			multiLineEntries = append(multiLineEntries, entry)
		} else {
			singleLineEntries = append(singleLineEntries, entry)
		}
	}

	// Sort both groups alphabetically by key (case-insensitive)
	sort.Slice(singleLineEntries, func(i, j int) bool {
		return strings.ToLower(singleLineEntries[i].key) < strings.ToLower(singleLineEntries[j].key)
	})
	sort.Slice(multiLineEntries, func(i, j int) bool {
		return strings.ToLower(multiLineEntries[i].key) < strings.ToLower(multiLineEntries[j].key)
	})

	// Rebuild the object with proper spacing
	var result strings.Builder
	result.WriteString("{\n")

	// Write single-line entries first (grouped together)
	for _, entry := range singleLineEntries {
		result.WriteString("    ")
		result.WriteString(entry.key)
		result.WriteString(" = ")
		result.WriteString(entry.value)
		result.WriteString("\n")
	}

	// Add blank line between single-line and multi-line if both exist
	if len(singleLineEntries) > 0 && len(multiLineEntries) > 0 {
		result.WriteString("\n")
	}

	// Write multi-line entries with blank lines between them
	for i, entry := range multiLineEntries {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString("    ")
		result.WriteString(entry.key)
		result.WriteString(" = ")
		result.WriteString(entry.value)
		result.WriteString("\n")
	}

	result.WriteString("  }")
	return result.String()
}

type objectEntry struct {
	key   string
	value string
}

// parseObjectEntriesFromText parses key-value pairs from object literal text using HCL parsing
func (s *Sorter) parseObjectEntriesFromText(text string) []objectEntry {
	var entries []objectEntry

	// Create a temporary HCL file to parse the object content
	tempContent := "temp = " + text
	tempFile, diags := hclwrite.ParseConfig([]byte(tempContent), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return entries // Return empty if parsing fails
	}

	body := tempFile.Body()
	attrs := body.Attributes()
	tempAttr, exists := attrs["temp"]
	if !exists {
		return entries
	}

	// Get the tokens for the expression and parse them
	tokens := tempAttr.Expr().BuildTokens(nil)
	result := s.parseObjectTokensToEntries(tokens)
	return result
}

// parseObjectTokensToEntries extracts key-value pairs from object tokens
func (s *Sorter) parseObjectTokensToEntries(tokens hclwrite.Tokens) []objectEntry {
	var entries []objectEntry

	// Find the opening brace
	openBraceIdx := -1
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenOBrace {
			openBraceIdx = i
			break
		}
	}

	if openBraceIdx == -1 {
		return entries
	}

	// Parse key-value pairs between braces
	i := openBraceIdx + 1
	braceLevel := 0

	for i < len(tokens) {
		// Skip whitespace and comments
		for i < len(tokens) && (tokens[i].Type == hclsyntax.TokenNewline ||
			tokens[i].Type == hclsyntax.TokenComment ||
			string(tokens[i].Bytes) == " " ||
			string(tokens[i].Bytes) == "\t") {
			i++
		}

		if i >= len(tokens) {
			break
		}

		// Check for closing brace at top level
		if tokens[i].Type == hclsyntax.TokenCBrace && braceLevel == 0 {
			break
		}

		// Look for key (identifier or quoted literal)
		if tokens[i].Type == hclsyntax.TokenIdent || tokens[i].Type == hclsyntax.TokenQuotedLit {
			keyToken := tokens[i]
			key := strings.Trim(string(keyToken.Bytes), "\"")
			i++

			// Skip whitespace to equals
			for i < len(tokens) && (string(tokens[i].Bytes) == " " || string(tokens[i].Bytes) == "\t") {
				i++
			}

			// Expect equals sign
			if i >= len(tokens) || tokens[i].Type != hclsyntax.TokenEqual {
				break
			}
			i++ // Skip equals

			// Skip whitespace after equals
			for i < len(tokens) && (string(tokens[i].Bytes) == " " || string(tokens[i].Bytes) == "\t") {
				i++
			}

			if i >= len(tokens) {
				break
			}

			// Collect value tokens
			valueStart := i
			braceLevel = 0
			bracketLevel := 0
			parenLevel := 0

			for i < len(tokens) {
				token := tokens[i]

				if token.Type == hclsyntax.TokenOBrace {
					braceLevel++
				} else if token.Type == hclsyntax.TokenCBrace {
					if braceLevel == 0 {
						// End of object at top level
						break
					}
					braceLevel--
				} else if token.Type == hclsyntax.TokenOBrack {
					bracketLevel++
				} else if token.Type == hclsyntax.TokenCBrack {
					bracketLevel--
				} else if token.Type == hclsyntax.TokenOParen {
					parenLevel++
				} else if token.Type == hclsyntax.TokenCParen {
					parenLevel--
				} else if token.Type == hclsyntax.TokenIdent && braceLevel == 0 && bracketLevel == 0 && parenLevel == 0 {
					// Next key at top level
					break
				}

				i++
			}

			// Build value string from tokens
			var valueBuilder strings.Builder
			for j := valueStart; j < i; j++ {
				valueBuilder.Write(tokens[j].Bytes)
			}
			value := strings.TrimSpace(valueBuilder.String())

			// Remove trailing comma if present
			value = strings.TrimSuffix(value, ",")
			value = strings.TrimSpace(value)

			if key != "" && value != "" {
				entries = append(entries, objectEntry{key: key, value: value})
			}
		} else {
			i++
		}
	}

	return entries
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

func (s *Sorter) cleanTrailingNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) == 0 {
		return tokens
	}

	// Find the last non-newline token
	lastNonNewlineIdx := -1
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Type != hclsyntax.TokenNewline {
			lastNonNewlineIdx = i
			break
		}
	}

	if lastNonNewlineIdx == -1 {
		// All tokens are newlines, keep just one
		return hclwrite.Tokens{&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		}}
	}

	// Keep everything up to the last non-newline token, plus exactly one newline
	result := make(hclwrite.Tokens, 0, lastNonNewlineIdx+2)
	for i := 0; i <= lastNonNewlineIdx; i++ {
		result = append(result, tokens[i])
	}

	// Add exactly one trailing newline
	result = append(result, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})

	return result
}
