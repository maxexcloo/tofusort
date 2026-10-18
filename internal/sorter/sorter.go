package sorter

import (
	"bytes"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/maxexcloo/tofusort/internal/parser"
	"github.com/zclconf/go-cty/cty"
)

var blockTypeOrder = map[string]int{
	"terraform": 0, "provider": 1, "variable": 2, "locals": 3,
	"data": 4, "resource": 5, "module": 6, "output": 7,
}

// Leading meta-arguments share one group in this order; depends_on trails.
var leadingMeta = map[string][]string{
	"data":     {"count", "for_each", "provider"},
	"module":   {"source", "version", "count", "for_each"},
	"provider": {"count", "for_each"},
	"resource": {"count", "for_each", "provider"},
}

// Identifier keys lead list-item objects in this order with --identifiers-first.
var identifierKeys = []string{"type", "name", "id"}

// Entry groups sort before keys. Root blocks add their type priority.
const (
	groupLeading = iota
	groupSingleLine
	groupMultiline
	groupDependsOn
	groupBlock
	groupRootBlock
)

type Options struct {
	IdentifiersFirst bool
}

type edit struct {
	start, end int
	content    []byte
}
type entry struct {
	start, end int
	key        string
	group      int
	content    []byte
	barrier    bool
	block      bool
}
type document struct {
	file    *parser.File
	edits   []edit
	options Options
}

func SortFile(file *parser.File, options Options) error {
	d := &document{file: file, options: options}
	d.body(file.Body, 0, len(file.Source), true, nil)
	output := d.render(0, len(file.Source))
	sorted, err := parser.ParseFile(output)
	if err != nil {
		return fmt.Errorf("sort produced invalid HCL; original file unchanged: %w", err)
	}
	formatted := parser.FormatFile(sorted)
	sorted, err = parser.ParseFile(formatted)
	if err != nil {
		return fmt.Errorf("format produced invalid HCL; original file unchanged: %w", err)
	}
	*file = *sorted
	return nil
}

func (d *document) render(start, end int) []byte {
	var out bytes.Buffer
	cursor := start
	for _, change := range d.edits {
		if change.start < start || change.end > end {
			continue
		}
		out.Write(d.file.Source[cursor:change.start])
		out.Write(change.content)
		cursor = change.end
	}
	out.Write(d.file.Source[cursor:end])
	return out.Bytes()
}

func (d *document) replace(start, end int, content []byte) {
	retained := d.edits[:0]
	for _, change := range d.edits {
		if change.start < start || change.end > end {
			retained = append(retained, change)
		}
	}
	d.edits = append(retained, edit{start, end, content})
	sort.Slice(d.edits, func(i, j int) bool { return d.edits[i].start < d.edits[j].start })
}

func (d *document) body(body *hclsyntax.Body, start, end int, root bool, meta []string) {
	var entries []entry
	for name, attr := range body.Attributes {
		d.expression(attr.Expr)
		r := attr.Range()
		content := d.render(r.Start.Byte, r.End.Byte)
		group, key := d.valueGroup(attr.Expr), name
		if i := slices.Index(meta, name); i >= 0 {
			group, key = groupLeading, fmt.Sprint(i)
		} else if meta != nil && name == "depends_on" {
			group = groupDependsOn
		}
		entries = append(entries, entry{start: r.Start.Byte, end: r.End.Byte, key: key, group: group, content: content})
	}
	for _, block := range body.Blocks {
		var meta []string
		if root {
			meta = leadingMeta[block.Type]
		}
		d.body(block.Body, block.OpenBraceRange.End.Byte, block.CloseBraceRange.Start.Byte, false, meta)
		r := block.Range()
		item := entry{start: r.Start.Byte, end: r.End.Byte, group: groupBlock, block: true, content: d.render(r.Start.Byte, r.End.Byte)}
		if root {
			priority, known := blockTypeOrder[block.Type]
			item.group = groupRootBlock + priority
			item.key = strings.Join(block.Labels, "\x00")
			item.barrier = !known
		}
		entries = append(entries, item)
	}
	d.reorder(start, end, entries, false)
}

func (d *document) expression(expr hclsyntax.Expression) {
	var objects []*hclsyntax.ObjectConsExpr
	var templates []hcl.Range
	listItems := make(map[*hclsyntax.ObjectConsExpr]bool)
	_ = hclsyntax.VisitAll(expr, func(node hclsyntax.Node) hcl.Diagnostics {
		switch node := node.(type) {
		case *hclsyntax.ObjectConsExpr:
			objects = append(objects, node)
		case *hclsyntax.TemplateExpr:
			templates = append(templates, node.Range())
		case *hclsyntax.TemplateWrapExpr:
			templates = append(templates, node.Range())
		case *hclsyntax.TupleConsExpr:
			if d.options.IdentifiersFirst {
				for _, item := range node.Exprs {
					for {
						parentheses, ok := item.(*hclsyntax.ParenthesesExpr)
						if !ok {
							break
						}
						item = parentheses.Expression
					}
					if object, ok := item.(*hclsyntax.ObjectConsExpr); ok {
						listItems[object] = true
					}
				}
			}
		}
		return nil
	})
	sort.Slice(objects, func(i, j int) bool { return objects[i].Range().Start.Byte > objects[j].Range().Start.Byte })
	for _, object := range objects {
		r := object.Range()
		inTemplate := false
		for _, template := range templates {
			if r.Start.Byte >= template.Start.Byte && r.End.Byte <= template.End.Byte {
				inTemplate = true
				break
			}
		}
		if inTemplate {
			continue
		}
		var entries []entry
		keys := make(map[string]bool)
		sortable := true
		for _, item := range object.Items {
			key, ok := literalKey(item.KeyExpr)
			if !ok || keys[key] {
				sortable = false
				break
			}
			keys[key] = true
			start, end := item.KeyExpr.Range().Start.Byte, item.ValueExpr.Range().End.Byte
			group, sortKey := d.valueGroup(item.ValueExpr), key
			if i := slices.Index(identifierKeys, key); listItems[object] && i >= 0 {
				group, sortKey = groupLeading, fmt.Sprint(i)
			}
			entries = append(entries, entry{start: start, end: end, key: sortKey, group: group, content: d.render(start, end)})
		}
		// Computed and duplicate keys can make source order significant.
		if sortable {
			d.reorder(r.Start.Byte+1, r.End.Byte-1, entries, true)
		}
	}
}

func (d *document) valueGroup(expr hclsyntax.Expression) int {
	r := expr.Range()
	if bytes.Contains(d.render(r.Start.Byte, r.End.Byte), []byte("\n")) {
		return groupMultiline
	}
	return groupSingleLine
}

func literalKey(expr hclsyntax.Expression) (string, bool) {
	key, ok := expr.(*hclsyntax.ObjectConsKeyExpr)
	if !ok || key.ForceNonLiteral {
		return "", false
	}
	switch wrapped := key.Wrapped.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		if len(wrapped.Traversal) != 1 {
			return "", false
		}
	case *hclsyntax.TemplateExpr:
		if !wrapped.IsStringLiteral() {
			return "", false
		}
	default:
		return "", false
	}
	value, diags := key.Value(nil)
	if diags.HasErrors() || !value.IsKnown() || value.IsNull() || value.Type() != cty.String {
		return "", false
	}
	return value.AsString(), true
}

// Attach same-line trailing comments and immediately preceding comment lines.
// Detached comments remain fixed boundaries that entries cannot cross.
func (d *document) attach(entries []entry, start, end int, object bool) []entry {
	for i := range entries {
		next := end
		if i+1 < len(entries) {
			next = entries[i+1].start
		}
		first := sort.Search(len(d.file.Tokens), func(j int) bool {
			return d.file.Tokens[j].Range.Start.Byte >= entries[i].end
		})
		for _, token := range d.file.Tokens[first:] {
			if token.Range.End.Byte > next {
				break
			}
			if token.Type != hclsyntax.TokenComment {
				continue
			}
			gap := d.file.Source[entries[i].end:token.Range.Start.Byte]
			if bytes.ContainsAny(gap, "\r\n") {
				break
			}
			// A block comment between two entries on one line has no clear owner.
			if i+1 < len(entries) && !bytes.Contains(d.file.Source[token.Range.Start.Byte:next], []byte("\n")) {
				entries[i].barrier = true
				entries[i+1].barrier = true
				break
			}
			if object {
				gap = bytes.ReplaceAll(gap, []byte(","), nil)
			}
			entries[i].content = append(entries[i].content, gap...)
			entries[i].content = append(entries[i].content, token.Bytes...)
			entries[i].end = token.Range.End.Byte
			if bytes.HasSuffix(token.Bytes, []byte("\n")) {
				break
			}
		}
	}
	for i := range entries {
		prev := start
		if i > 0 {
			prev = entries[i-1].end
		}
		leading := entries[i].start
		last := sort.Search(len(d.file.Tokens), func(j int) bool {
			return d.file.Tokens[j].Range.End.Byte > leading
		})
		for j := last - 1; j >= 0; j-- {
			token := d.file.Tokens[j]
			if token.Range.Start.Byte < prev {
				break
			}
			if token.Type != hclsyntax.TokenComment {
				continue
			}
			gap := d.file.Source[token.Range.End.Byte:leading]
			if len(bytes.TrimSpace(gap)) > 0 || bytes.Count(gap, []byte("\n")) > 1 {
				break
			}
			// Line comments contain their terminating newline; another is a blank line.
			if bytes.HasSuffix(token.Bytes, []byte("\n")) && bytes.Contains(gap, []byte("\n")) {
				break
			}
			leading = token.Range.Start.Byte
		}
		if leading != entries[i].start {
			prefix := append([]byte(nil), d.file.Source[leading:entries[i].start]...)
			entries[i].content = append(prefix, entries[i].content...)
			entries[i].start = leading
		}
	}
	return entries
}

func (d *document) reorder(start, end int, entries []entry, object bool) {
	if len(entries) == 0 {
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].start < entries[j].start })
	// Object layout follows the gaps around entries, not newlines inside values.
	inline := object
	previous := start
	for _, item := range entries {
		inline = inline && !bytes.Contains(d.file.Source[previous:item.start], []byte("\n"))
		previous = item.end
	}
	inline = inline && !bytes.Contains(d.file.Source[previous:end], []byte("\n"))
	if object && !inline {
		// Newline-separated entries keep their own trailing comma.
		for i := range entries {
			gap := d.file.Source[entries[i].end:end]
			if rest := bytes.TrimLeft(gap, " \t"); len(rest) > 0 && rest[0] == ',' {
				comma := entries[i].end + len(gap) - len(rest) + 1
				entries[i].content = append(entries[i].content, d.file.Source[entries[i].end:comma]...)
				entries[i].end = comma
			}
		}
	}
	entries = d.attach(entries, start, end, object)
	var out bytes.Buffer
	cursor := start
	for first := 0; first < len(entries); {
		last := first + 1
		for last < len(entries) && !entries[last-1].barrier && !entries[last].barrier && whitespace(d.file.Source[entries[last-1].end:entries[last].start], object) {
			last++
		}
		out.Write(d.file.Source[cursor:entries[first].start])
		run := append([]entry(nil), entries[first:last]...)
		sort.SliceStable(run, func(i, j int) bool {
			if run[i].group != run[j].group {
				return run[i].group < run[j].group
			}
			return run[i].key < run[j].key
		})
		for i, item := range run {
			if i > 0 {
				if inline {
					out.WriteString(", ")
				} else {
					out.WriteByte('\n')
					if !object && (item.block || run[i-1].block || item.group != run[i-1].group || item.group == groupMultiline) {
						out.WriteByte('\n')
					}
				}
			}
			out.Write(bytes.TrimSuffix(item.content, []byte("\n")))
		}
		trailing := bytes.TrimLeft(d.file.Source[entries[last-1].end:end], " \t\r")
		if bytes.HasSuffix(d.file.Source[entries[last-1].start:entries[last-1].end], []byte("\n")) ||
			(bytes.HasSuffix(run[len(run)-1].content, []byte("\n")) && !bytes.HasPrefix(trailing, []byte("\n"))) {
			out.WriteByte('\n')
		}
		cursor = entries[last-1].end
		first = last
	}
	out.Write(d.file.Source[cursor:end])
	d.replace(start, end, out.Bytes())
}

func whitespace(src []byte, object bool) bool {
	if object {
		return len(bytes.Trim(src, " \t\r\n,")) == 0
	}
	return len(bytes.TrimSpace(src)) == 0
}
