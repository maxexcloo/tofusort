# Plan

## Correctness

- Add support for the documented `.tf.json` and `.tfvars.json` inputs, or narrow the README to the HCL formats the CLI actually accepts.
- Continue checking remaining inputs after a per-file failure and return a combined non-zero result.

## Documentation and licensing

- Add the canonical AGPL-3.0-only `LICENSE` file so the README badge and package metadata point to an actual licence.
- Replace the placeholder module and clone paths and remove or repair broken `CLAUDE.md` references.

## Tooling and CI

- Make `mise run check` non-mutating by separating formatting checks from formatting writes.
- Run build, lint, and tests in CI in addition to the container build.
- Remove the repository-specific `GOPATH` template so Mise configuration can be trusted and used without local interpolation issues.
- Pin the Alpine runtime image and APK package inputs to reproducible versions.
- Resolve the current Go formatting and Hadolint findings.
