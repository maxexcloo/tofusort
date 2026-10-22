# AGENTS.md

## Structure

- Keep CLI commands in `cmd/tofusort/` and HCL parsing, formatting and sorting
  with its tests in `internal/sorter/`.
- Support HCL-format `.tf` and `.tfvars` files. Do not claim JSON support unless a
  JSON-aware implementation and tests are included.

## Behaviour

`README.md` defines the behaviour. Keep these invariants:

- Continue after a per-file failure and report every failure with its file.
- Keep `check` non-mutating and non-zero for unsorted or invalid input.
- Keep attached comments with their entries and detached comments as boundaries.
- Order leading meta-arguments per the HashiCorp style guide, as defined by
  `leadingMeta` in `internal/sorter/sorter.go`.

## Verification

Run `mise run check`.
