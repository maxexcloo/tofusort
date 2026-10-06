package sorter

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/maxexcloo/tofusort/internal/parser"
	"github.com/zclconf/go-cty/cty"
)

type Sorter struct{}

func New() *Sorter { return &Sorter{} }

var blockTypeOrder = map[string]int{
	"terraform": 0, "provider": 1, "variable": 2, "locals": 3,
	"data": 4, "resource": 5, "module": 6, "output": 7,
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
	file  *parser.File
	edits []edit
}

func (s *Sorter) SortFile(file *parser.File) error {
	d := &document{file: file}
	d.body(file.Body, 0, len(file.Source), true, false)
	output := d.render(0, len(file.Source))
	p := parser.New()
	sorted, err := p.ParseFile(output)
	if err != nil {
		return fmt.Errorf("sort produced invalid HCL; original file unchanged: %w", err)
	}
	formatted := p.FormatFile(sorted)
	sorted, err = p.ParseFile(formatted)
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

func (d *document) body(body *hclsyntax.Body, start, end int, root, meta bool) {
	var entries []entry
	for name, attr := range body.Attributes {
		d.expression(attr.Expr)
		r := attr.Range()
		content := d.render(r.Start.Byte, r.End.Byte)
		group := 2
		if bytes.Contains(d.render(attr.Expr.Range().Start.Byte, attr.Expr.Range().End.Byte), []byte("\n")) {
			group = 3
		}
		if meta {
			switch name {
			case "count":
				group = 0
			case "for_each":
				group = 1
			case "depends_on":
				group = 4
			}
		}
		entries = append(entries, entry{start: r.Start.Byte, end: r.End.Byte, key: name, group: group, content: content})
	}
	for _, block := range body.Blocks {
		hasMeta := root && (block.Type == "resource" || block.Type == "data" || block.Type == "module" || block.Type == "provider")
		d.body(block.Body, block.OpenBraceRange.End.Byte, block.CloseBraceRange.Start.Byte, false, hasMeta)
		r := block.Range()
		item := entry{start: r.Start.Byte, end: r.End.Byte, group: 5, block: true, content: d.render(r.Start.Byte, r.End.Byte)}
		if root {
			priority, known := blockTypeOrder[block.Type]
			item.group = 10 + priority
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
	_ = hclsyntax.VisitAll(expr, func(node hclsyntax.Node) hcl.Diagnostics {
		switch node := node.(type) {
		case *hclsyntax.ObjectConsExpr:
			objects = append(objects, node)
		case *hclsyntax.TemplateExpr:
			templates = append(templates, node.Range())
		case *hclsyntax.TemplateWrapExpr:
			templates = append(templates, node.Range())
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
			group := 0
			if bytes.Contains(d.render(item.ValueExpr.Range().Start.Byte, end), []byte("\n")) {
				group = 1
			}
			entries = append(entries, entry{start: start, end: end, key: key, group: group, content: d.render(start, end)})
		}
		// Computed and duplicate keys can make source order significant.
		if sortable {
			d.reorder(r.Start.Byte+1, r.End.Byte-1, entries, true)
		}
	}
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
		for _, token := range d.file.Tokens {
			if token.Range.Start.Byte < entries[i].end || token.Range.End.Byte > next {
				continue
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
		for j := len(d.file.Tokens) - 1; j >= 0; j-- {
			token := d.file.Tokens[j]
			if token.Range.Start.Byte < prev || token.Range.End.Byte > leading || token.Type != hclsyntax.TokenComment {
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
		multiline := bytes.Contains(d.file.Source[start:end], []byte("\n"))
		for i, item := range run {
			if i > 0 {
				if object && !multiline {
					out.WriteString(", ")
				} else {
					out.WriteByte('\n')
					if !object && (item.block || run[i-1].block || item.group != run[i-1].group || item.group == 3) {
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
