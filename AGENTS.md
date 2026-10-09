# AGENTS.md

## Structure

- Keep CLI commands in `cmd/tofusort/`.
- Keep HCL parsing and formatting in `internal/parser/`.
- Keep sorting behaviour and its tests in `internal/sorter/`.
- Support HCL-format `.tf` and `.tfvars` files. Do not claim JSON support unless a
  JSON-aware implementation and tests are included.

## Style

- Follow Go naming and formatting conventions.
- Keep comments minimal and specific to non-obvious parser or sorting behaviour.
- Preserve relative comment positions when sorting HCL.
- Update `README.md` and `docs/architecture.md` when behaviour changes.

## Behaviour

- Continue processing independent inputs after a per-file failure and report all
  failures together.
- Keep `check` non-mutating and return a non-zero exit status for unsorted or
  invalid input.
- Preserve standard Unix-style flags and include file context in errors.
