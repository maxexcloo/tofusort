# AGENTS.md

## Structure

- Keep CLI commands in `cmd/tofusort/` and HCL parsing, formatting and sorting
  with its tests in `internal/sorter/`.

## Behaviour

`README.md` defines the supported formats and behaviour. Keep implementation and
tests consistent with it. `leadingMeta` in `internal/sorter/sorter.go` defines the
leading meta-argument order.

## Verification

Run `mise run check`.
