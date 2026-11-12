# tofusort

Sorts OpenTofu and Terraform HCL files. It handles `.tf` and `.tfvars` files, but
not JSON configuration.

## Installation

```bash
mise install
mise run build
./tofusort --help
```

Or install it without cloning:

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

`check` exits non-zero for unsorted, invalid or inaccessible inputs.

## Sorting Rules

Following the HashiCorp style guide, meta-arguments such as `source`, `count`
and `for_each` come first and `depends_on` follows the other attributes. Other
attributes and literal object keys sort alphabetically, single-line values
before multiline ones, followed by nested blocks in their original order. List
elements, function arguments and objects with computed keys keep their order.

Top-level blocks are ordered `terraform`, `provider`, `variable`, `locals`,
`data`, `resource`, `module`, `output`, then by label; unknown block types stay
in place. Comments stay with the entry they precede or trail, and detached
comments act as boundaries that entries never cross.

## Docker

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$(pwd):/workspace" -w /workspace ghcr.io/maxexcloo/tofusort sort main.tf
```

## Development

Install the [shared Mise configuration](https://github.com/maxexcloo/dotfiles#shared-tools--tasks)
before running local Mise tasks.

```bash
mise run check
mise run fmt

# Exercise an external HCL corpus without modifying it
TOFUSORT_CORPUS="$PWD/../homelab" mise exec -- go test ./internal/sorter -run TestExternalCorpus -v

# Explore parser and sorting edge cases
mise exec -- go test ./internal/sorter -run '^$' -fuzz FuzzSortPreservesStructure -fuzztime=30s
```

## Licence

AGPL-3.0 - see [LICENSE](LICENSE).
