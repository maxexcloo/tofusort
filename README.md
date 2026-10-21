# tofusort

Sorts OpenTofu and Terraform HCL files. It's written in Go and uses the native
HCL parser. It handles `.tf` and `.tfvars` files, but not JSON configuration.

## Installation

```bash
mise install
mise run build
./tofusort --help
```

To install the binary into the configured Go binary directory, run
`mise run install`, or install it without cloning:

```bash
go install github.com/maxexcloo/tofusort/cmd/tofusort@latest
```

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

# Put type, name and id first in list-item objects
./tofusort sort --identifiers-first main.tf
./tofusort check --identifiers-first main.tf
```

Both commands take multiple paths and keep going when one fails, then report
every error with its file. `check` exits non-zero for unsorted, invalid or
inaccessible inputs. Directory scans skip unsupported extensions, but naming an
unsupported file directly is an error. Use `--` before filenames that start with
a dash. Directory symlinks you pass in are followed, including with
`--recursive`, but directory symlinks found during a scan aren't. Recursive scans
skip `.git` and `.terraform` directories below the given root.

## Sorting Rules

Attributes with single-line values come first, then multiline values, sorted
alphabetically within each group. Following the HashiCorp style guide,
meta-arguments go first as one group: `source`, `version`, `count` and
`for_each` in modules; `count`, `for_each` and `provider` in resources and data
sources; `count` and `for_each` in providers. `depends_on` goes after the
ordinary attributes in these bodies. Nested blocks come after attributes and keep
their original order.

Comment lines directly above an entry and trailing comments on the same line stay
with that entry. An ambiguous comment between inline peers keeps those peers in
place. Detached section comments act as boundaries, so entries on either side
are never mixed. File headers and footers separated from entries by blank lines
stay where they are.

HCL's formatter handles indentation and alignment. Objects stay inline when no
newline separates their entries, even if a value spans lines, and entries in
multiline objects keep their own trailing commas. The sorter puts blank lines
between attribute groups and blocks. It doesn't rewrite literal or template
contents or run regular expressions over source text. CRLF and lone CR line
endings are normalised to LF, so `check` reports those files as unsorted until
`sort` rewrites them.

Bare and quoted literal object keys follow the same single-line-first rule,
including objects inside functions, comprehensions and type constraints. Objects
with computed or duplicate keys keep their peer order, though any child objects
that can be sorted on their own still are. With `--identifiers-first`, objects
directly inside list literals put `type`, `name`, then `id` before other keys,
whatever the value layout. Other object keys keep their usual grouping, and
comment boundaries and list element order are kept.

List elements, function arguments, nested blocks and expressions inside string
templates keep their order. Unknown top-level block types stay put as fixed
boundaries, so procedural constructs aren't shuffled.

Recognised top-level blocks are ordered `terraform`, `provider`, `variable`,
`locals`, `data`, `resource`, `module`, `output`, then alphabetically by label
within each type. Equal labels keep their source order.

Output is parsed again before anything is written. tofusort writes a temporary
file next to the original, flushes it and renames it over the original, keeping
ordinary permission bits and symbolic links. The replacement is a new inode, so
hard links, ownership and extended metadata aren't kept.

## Docker

```bash
docker build -t tofusort .
docker run --rm -v "$(pwd):/workspace" -w /workspace tofusort sort main.tf
```

The Container workflow runs checks and publishes AMD64 and ARM64 images for
pushes to `main` and version tags. Pushes to `main` that only touch Markdown,
`.gitignore` or `LICENSE` skip it. Pull requests run the same checks and image
build without publishing.

## Development

Install the [shared Mise configuration](https://github.com/maxexcloo/dotfiles#shared-tools--tasks)
before running local Mise tasks.

`mise run check` runs the shared checks (actionlint, Mise and Oxfmt
formatting, whitespace and Renovate validation) plus Go vet, tests and gofmt
verification.

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
