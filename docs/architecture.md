# Architecture

`tofusort` uses HCL's parser to locate complete entries and its formatter to lay
out the result. It does not implement an expression parser or scan for matching
braces itself.

## Components

- `cmd/tofusort/`: Cobra commands, file discovery, aggregate errors, non-mutating
  checks and dry-runs, and file replacement.
- `internal/parser/`: `hclsyntax.ParseConfig` produces the syntax tree and
  `LexConfig` identifies comments. A parsed file holds source bytes, its body and
  lexer tokens. `hclwrite.Format` handles indentation and alignment; non-empty
  output receives a final newline.
- `internal/sorter/`: Sorting policy, source-range edits and preservation tests.

## Sorting

The sorter walks each body and expression. Literal-key object constructors are
processed from the innermost outwards, including constructors in function
arguments, comprehensions and type constraints. Source ranges identify whole
key/value pairs, so operators, traversals, quoted keys and nested expressions do
not need custom token parsing. Computed and duplicate keys prevent peer sorting.
Objects inside string templates are left alone.

Body attributes use single-line and multiline value groups, then alphabetical
names. Resource, data, module and provider bodies put `count` and `for_each`
first and `depends_on` after ordinary attributes. Nested blocks follow attributes
in their original order, including lifecycle, provisioner, validation, static and
dynamic blocks. Root blocks use the documented type priority and labels; unknown
root blocks are fixed boundaries. All equal sort keys retain source order.

Same-line trailing comments and immediately preceding comment lines are attached
to complete entries. Ambiguous comments between inline peers keep those peers
in place. Detached comments delimit sortable runs. Whitespace and
comments outside those runs stay in place. Inline objects use comma separators;
multiline objects use newlines. HCL formats the resulting source.

Edits operate on the original byte offsets. Replacing a parent range consumes
its child edits. The complete output is reparsed both before and after formatting;
the caller's parsed file changes only after success. Syntax validation prevents
invalid output from being written, but is not a proof of semantic equivalence.

## Files & Errors

Independent paths and files continue after a failure. `check` compares the
original bytes with sorted output and never writes. Dry-run only reports paths.
Writes resolve symbolic links and use a temporary sibling, preserve ordinary
permission bits, sync and rename. Renaming creates a new inode, so hard links,
ownership and extended metadata are not preserved.

## Verification

Tests cover explicit sorting results, comment attachment and section boundaries,
quoted and computed keys, duplicate-key semantics, Unicode, heredocs, templates,
functions, comprehensions, ordered blocks and non-mutating CLI behaviour.

An independent structural comparison ignores source positions and permitted
mapping order while retaining expression structure, values, arguments and nested
block order. Every corpus and fuzz case checks this structure, comment contents,
valid output and idempotence. Constant-expression regressions also compare
actual evaluated values. `TOFUSORT_CORPUS` enables the same checks against a local
HCL directory without copying its contents into the repository or modifying it.

Legacy sorting examples ignore blank-line differences; dedicated formatting and
comment tests assert exact output. Run `mise run check` before handoff.
