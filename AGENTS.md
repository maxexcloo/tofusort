# AGENTS.md

## Structure

- Keep CLI commands in `cmd/tofusort/`.
- Keep HCL parsing and formatting in `internal/parser/`.
- Keep only `AGENTS.md` and `README.md` as root Markdown files; put other project
  documentation in `docs/`.
- Keep sorting behaviour and its tests in `internal/sorter/`.
- Support HCL-format `.tf` and `.tfvars` files. Do not claim JSON support unless a
  JSON-aware implementation and tests are included.

## Style

- Follow Go naming and formatting conventions.
- Keep comments minimal and specific to non-obvious parser or sorting behaviour.
- Preserve relative comment positions when sorting HCL.
- Sort unordered peer entries by value shape, then alphabetically within each
  shape. Preserve semantic and procedural order.
- Update `README.md` and `docs/architecture.md` when behaviour changes.
- Use `.yaml`, never `.yml`, for project-owned YAML files unless external tooling
  requires a fixed filename.
- Preserve `LICENSE` and its legal text; never relicense without explicit approval.
- Use Australian English in project-owned prose and identifiers. Preserve external
  names and terminology.

## Behaviour

- Continue processing independent inputs after a per-file failure and report all
  failures together.
- Keep `check` non-mutating and return a non-zero exit status for unsorted or
  invalid input.
- Preserve standard Unix-style flags and include file context in errors.

## Verification

- Run `mise run check` before handoff.
