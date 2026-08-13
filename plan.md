# Repository Review Plan

The repository is compact and well organised, and `mise run check` passes.
However, two serious data-preservation problems should be addressed before the
tool is relied on broadly.

## Findings

### 1. High: formatting can change heredoc and template values

`internal/parser/parser.go` runs regular expressions over the fully rendered
source, including string and heredoc contents. Collapsing three or more
newlines can remove intentional blank lines from a heredoc and change the
Terraform value. The block-start expression can similarly match text inside a
template.

Blank-line cleanup should operate on HCL tokens or nodes, or be removed in
favour of `hclwrite.Format`.

### 2. High: comments are deleted or detached from their entries

`internal/sorter/sorter.go` clears the top-level body before reinserting
blocks, deleting standalone file headers and comments between blocks.

Within blocks and `.tfvars` files, attributes are removed while comment tokens
remain in place. The attributes are then appended elsewhere, so preceding
comments no longer accompany them. Object parsing also skips comments before
entries without restoring them.

This contradicts the project requirement and README claim that relative
comment positions are preserved. It can also invalidate positional `tfsec`,
Checkov, lint, or explanatory comments.

### 3. Medium: ordered nested blocks can be reordered unsafely

All regular nested blocks are sorted, while dynamic blocks receive additional
content-based sorting. Some provider schemas treat repeated blocks as ordered
lists, so changing their order can change behaviour.

Nested `BlockInfo` values also omit labels, meaning many blocks compare equal.
All block sorts use unstable `sort.Slice`, so original order is not guaranteed
when the comparator considers entries equal.

Ordered peers should be explicitly preserved, generally using stable sorting
and a clear allowlist of unordered constructs.

### 4. Medium: CLI errors are printed twice and include usage noise

Cobra prints `RunE` errors and usage by default, after which `main` prints the
returned error again. Aggregated per-file errors can consequently appear
twice.

Set `SilenceErrors` and `SilenceUsage` on the root command and let `main` own
error presentation.

### 5. Low: explicitly supplied unsupported files silently succeed

The `sort` and `check` paths treat a directly supplied unsupported extension as
successful. For example, checking a `.tf.json` file or a mistyped extension can
report that all files are sorted.

Directory discovery may reasonably skip unsupported files, but explicit file
arguments should return an unsupported-format error.

## Simplification and Easy Improvements

- Reduce the more than 1,200 lines of hand-written token parsing in
  `internal/sorter/sorter.go`. Whitespace reconstruction accounts for much of
  the complexity and several preservation risks.
- Remove or use `isSimpleExpression`. It has no initial caller, so its recursive
  calls do not make it reachable and the associated `nolint` explanation is
  misleading.
- If the parser regular expressions are retained, compile them once at package
  scope rather than during every formatting operation.
- Replace direct truncating writes with a sibling temporary file, sync, mode
  preservation, and atomic rename.
- Expand CLI and parser tests. Current statement coverage is 29.1% for the CLI
  and 0% for the parser.
- Add cases for `check` non-mutation, mixed invalid and unsorted files,
  recursive traversal, dry-run behaviour, comments, heredocs, file modes, and
  unsupported explicit inputs.
- Correct README and architecture claims about complete comment preservation
  and recursive sorting of all nested structures until the implementation
  guarantees those behaviours.

## Suggested Order of Work

1. Add regression tests for heredoc content and comment attachment.
2. Remove source-wide blank-line regular expressions.
3. Redesign sorting around complete token ranges so comments travel with their
   entries.
4. Define which blocks and entries are unordered, and preserve everything else
   with stable sorting.
5. Correct CLI error handling and unsupported-input reporting.
6. Add atomic file replacement.
7. Remove dead token-parsing code and update the documentation.

## Verification

Run `mise run check` after each behavioural change and before handoff.
