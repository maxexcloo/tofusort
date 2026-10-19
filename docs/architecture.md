# Architecture

`tofusort` uses HCL's parser to locate complete entries and its formatter to lay
out the result. It does not implement an expression parser or scan for matching
braces itself.

## Components

- `cmd/tofusort/`: Cobra commands, file discovery, aggregate errors, non-mutating
  checks and dry-runs, and file replacement. Both commands share file discovery
  and the read, parse, sort and format path.
- `internal/parser/`: `hclsyntax.ParseConfig` produces the syntax tree and
  `LexConfig` identifies comments. Parsing normalises CRLF and lone CR line
  endings to LF first. A parsed file holds source bytes, its body and lexer
  tokens. `hclwrite.Format` handles indentation and alignment; non-empty output
  receives a final newline.
- `internal/sorter/`: Sorting policy, source-range edits and preservation tests.

## Sorting

The sorter walks each body and expression. Literal-key object constructors are
processed from the innermost outwards, including constructors in function
arguments, comprehensions and type constraints. Source ranges identify whole
key/value pairs, so operators, traversals, quoted keys and nested expressions do
not need custom token parsing. Computed and duplicate keys prevent peer sorting.
Objects inside string templates are left alone.

Both commands share the opt-in `--identifiers-first` policy. It gives `type`,
`name`, then `id` precedence in objects directly inside list literals, including
parenthesised objects. Other objects use the default policy. Identifier precedence
applies before single-line and multiline grouping and respects comment boundaries,
computed keys and duplicate keys. List elements retain their original order.

Body attributes use single-line and multiline value groups, then alphabetical
names. Top-level resource, data, module and provider bodies put their leading
meta-arguments first as one group, in the order defined by `leadingMeta`, and
`depends_on` after ordinary attributes. Nested blocks follow attributes
in their original order, including lifecycle, provisioner, validation, static and
dynamic blocks. Root blocks use the documented type priority and labels; unknown
root blocks are fixed boundaries. All equal sort keys retain source order.

Same-line trailing comments and immediately preceding comment lines are attached
to complete entries. Ambiguous comments between inline peers keep those peers
in place. Detached comments delimit sortable runs. Whitespace and comments
outside those runs stay in place. Objects stay inline with comma separators
when no newline separates their entries, even if a value spans lines; multiline
objects use newlines, and each entry keeps its own trailing comma. Comment
attachment searches the token stream for adjacent gaps rather than scanning the
whole file for each entry. HCL formats the resulting source once, within the
sorter.

Edits operate on the original byte offsets. Replacing a parent range consumes
its child edits. The complete output is reparsed both before and after formatting;
the caller's parsed file changes only after success. Syntax validation prevents
invalid output from being written, but is not a proof of semantic equivalence.

## Files & Errors

Independent paths and files continue after a failure. `check` compares the
original bytes with sorted output and never writes. Dry-run only reports paths.
Explicit directory symlinks are followed for both shallow and recursive scans,
and files are reported under the supplied path;
directory symlinks encountered below the supplied root are not traversed.
Recursive scans skip `.git` and `.terraform` directories below the root.
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

`hclwrite.Format` joins a number literal and a separated traversal into one
number, so `A=0 .0` becomes `A = 0.0`, changing a legacy index into a literal.
This upstream limitation is accepted, and fuzzing skips number literals followed
by a dot.

Legacy sorting examples ignore blank-line differences; dedicated formatting and
comment tests assert exact output.
