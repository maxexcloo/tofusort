# Architecture

`tofusort` uses HCL's parser to find complete entries and HCL's formatter to lay
out the result. It has no expression parser of its own and doesn't scan for
matching braces.

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

The sorter walks each body and expression. Object constructors with literal
keys are processed from the innermost outwards, including ones in function
arguments, comprehensions and type constraints. Source ranges mark whole
key/value pairs, so operators, traversals, quoted keys and nested expressions
don't need custom token parsing. Computed and duplicate keys stop peers from being
sorted. Objects inside string templates are left alone.

Both commands support the opt-in `--identifiers-first` policy. It puts `type`,
`name`, then `id` first in objects directly inside list literals, including
parenthesised objects. Other objects use the default policy. Identifier order is
applied before single-line and multiline grouping and respects comment
boundaries, computed keys and duplicate keys. List elements keep their original
order.

Body attributes are grouped by single-line and multiline values, then sorted by
name. Top-level resource, data, module and provider bodies put their leading
meta-arguments first as one group, in the order set by `leadingMeta`, and
`depends_on` after the ordinary attributes. Nested blocks follow the attributes
in their original order, including lifecycle, provisioner, validation, static and
dynamic blocks. Root blocks use the documented type priority and labels, and
unknown root blocks are fixed boundaries. Equal sort keys always keep source
order.

Trailing comments on the same line and comment lines directly above an entry are
attached to that entry. An ambiguous comment between inline peers keeps those
peers in place. Detached comments mark the edges of sortable runs, and whitespace
and comments outside those runs stay where they are. Objects stay inline with
comma separators when no newline separates their entries, even if a value spans
lines. Multiline objects use newlines, and each entry keeps its own trailing
comma. To attach comments, the sorter searches the token stream for adjacent gaps
instead of scanning the whole file for each entry. HCL formats the resulting
source once, inside the sorter.

Edits work on the original byte offsets. Replacing a parent range absorbs its
child edits. The complete output is parsed again before and after formatting, and
the caller's parsed file only changes once that succeeds. Syntax validation stops
invalid output from being written, but it doesn't prove the result means the same
thing.

## Files & Errors

A failure in one path or file doesn't stop the others. `check` compares the
original bytes with the sorted output and never writes. Dry-run only reports
paths. Directory symlinks passed in directly are followed for both shallow and
recursive scans, and files are reported under the given path. Directory symlinks
found below the given root aren't followed. Recursive scans skip `.git` and
`.terraform` directories below the root. Writes resolve symbolic links, write a
temporary file next to the target, keep ordinary permission bits, sync and
rename. Renaming creates a new inode, so hard links, ownership and extended
metadata aren't kept.

## Verification

Tests cover explicit sorting results, comment attachment and section boundaries,
quoted and computed keys, duplicate-key semantics, Unicode, heredocs, templates,
functions, comprehensions, ordered blocks and non-mutating CLI behaviour.

A separate structural comparison ignores source positions and allowed mapping
order but keeps expression structure, values, arguments and nested block order.
Every corpus and fuzz case checks this structure, comment contents, valid output
and idempotence. Constant-expression regression tests also compare the actual
evaluated values. `TOFUSORT_CORPUS` runs the same checks against a local HCL
directory without copying its contents into the repository or changing it.

`hclwrite.Format` joins a number literal and a separated traversal into one
number, so `A=0 .0` becomes `A = 0.0` and a legacy index turns into a literal.
I've accepted this upstream limitation, and fuzzing skips number literals followed
by a dot.

Legacy sorting examples ignore blank-line differences; dedicated formatting and
comment tests assert exact output.
