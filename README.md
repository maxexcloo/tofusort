# tofusort

Sort OpenTofu and Terraform HCL files with Go and the native HCL parser.
Supports `.tf` and `.tfvars`; JSON configuration is not supported.

## Installation

```bash
mise install
mise run build
./tofusort --help
```

To install the binary into the configured Go binary directory, run
`mise run install`.

## Usage

```bash
# Sort a file or directory
./tofusort sort main.tf

# Sort directories recursively
./tofusort sort -r ./modules

# Report changes without writing
./tofusort sort --dry-run main.tf

# Check sorting and formatting without writing
./tofusort check main.tf
```

Both commands accept multiple paths and continue after individual failures,
reporting all errors with file context. `check` exits non-zero for unsorted,
invalid or inaccessible inputs. Directory scans skip unsupported extensions;
explicit unsupported file arguments produce errors. Use `--` before filenames
that start with a dash.

## Sorting Rules

- **Attributes:** Single-line values first, then multiline values, alphabetical
  within each group. In resource, data, module and provider bodies, `count` and
  `for_each` lead; `depends_on` follows ordinary attributes. Nested blocks follow
  attributes and retain their original order.
- **Comments:** Immediately preceding comment lines and same-line trailing
  comments stay with their entry. Ambiguous comments between inline peers keep
  those peers in place. Detached section comments are boundaries:
  entries on opposite sides are not mixed. File headers and footers separated
  from entries by blank lines remain in place.
- **Formatting:** HCL's formatter handles indentation and alignment. Inline
  objects remain inline. The sorter separates attribute groups and blocks with
  blank lines; it does not rewrite literal or template contents or run regular
  expressions over source text.
- **Objects:** Bare and quoted literal keys follow the same single-line-first
  rule, including objects inside functions, comprehensions and type constraints.
  Objects with computed or duplicate keys retain peer order; their independently
  sortable child objects can still be sorted.
- **Order preservation:** List elements, function arguments, nested blocks and
  expressions inside string templates retain their order. Unknown top-level
  block types remain fixed boundaries, so procedural constructs are not shuffled.
- **Top-level blocks:** Recognised types follow `terraform`, `provider`,
  `variable`, `locals`, `data`, `resource`, `module`, `output`, then alphabetical
  labels within each type. Equal labels retain source order.

Output is parsed again before any write. A temporary sibling file is written,
flushed and renamed over the original; ordinary permission bits and symbolic
links are preserved. Replacement creates a new inode and does not preserve
hard links, ownership or extended metadata.

## Docker

```bash
docker build -t tofusort .
docker run --rm -v "$(pwd):/workspace" -w /workspace tofusort sort main.tf
```

## Development

```bash
mise run check
mise run fmt

# Exercise an external HCL corpus without modifying it
TOFUSORT_CORPUS="$PWD/../homelab" mise exec -- go test ./internal/sorter -run TestExternalCorpus -v

# Explore parser and sorting edge cases
mise exec -- go test ./internal/sorter -run '^$' -fuzz FuzzSortPreservesStructure -fuzztime=30s
```

Follow [AGENTS.md](AGENTS.md). See [architecture](docs/architecture.md) for the
implementation and preservation tests. Licensed under [AGPL-3.0](LICENSE).
