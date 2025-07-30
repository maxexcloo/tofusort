# CLAUDE.md - Development Guide

## Project Overview
**Purpose**: Tool to sort Terraform/OpenTofu configuration files alphabetically
**Status**: Active
**Language**: Go (for native HCL v2 parser integration)

## Code Standards

### Organization
- **Config/Data**: Alphabetical and recursive (imports, dependencies, object keys)
- **Documentation**: Sort alphabetically and recursively when it makes logical sense - apply to sections, subsections, lists, and references
- **Files**: Alphabetical in documentation and directories
- **Functions**: Group by purpose, alphabetical within groups
- **Variables**: Alphabetical within scope

### Quality
- **Comments**: Minimal - only for complex business logic
- **Documentation**: Update README.md and docs with every feature change
- **Error handling**: Always handle parser errors gracefully
- **Formatting**: Run `go fmt` before commits
- **KISS principle**: Keep it simple - prefer readable code over clever code
- **Naming**: Go conventions (camelCase for functions, PascalCase for types)
- **Testing**: Unit tests for all sorting logic
- **Trailing newlines**: Required in all files

## Development Guidelines

### Documentation Structure
- **ARCHITECTURE.md**: Technical design and implementation details
- **CLAUDE.md**: Development standards and project guidelines (this file)
- **README.md**: Tool overview and usage guide

### Contribution Standards
- **Code Changes**: Follow sorting rules and maintain test coverage
- **Documentation**: Keep all docs synchronized and cross-referenced
- **Feature Changes**: Update README.md and ARCHITECTURE.md when adding features

## Command Interface Standards
- **Clear output**: Provide informative messages about what was processed
- **Consistent flags**: Use standard Unix-style flags (-r, --dry-run)
- **Error messages**: Include file names and line numbers where possible
- **Exit codes**: 0 for success, 1 for failure, follow standard conventions

## Development Workflow Standards

### Environment Management
- Use **mise** for consistent development environments
- Pin tool versions in `.mise.toml`
- Define common tasks as mise scripts

### Required Development Tasks
- **build**: Create production binary
- **check**: All validation (fmt + lint + test)
- **dev**: Development validation cycle
- **fmt**: Code formatting
- **lint**: Code quality checks
- **test**: Run full test suite

## Error Handling Standards
- **Contextual errors**: Show surrounding code when possible
- **Graceful degradation**: Continue processing when individual files fail
- **Informative messages**: Include file paths and line numbers
- **User-friendly output**: Clear explanations for common issues

## Extension Guidelines
- **Backward compatibility**: Maintain API stability
- **Configuration files**: Plan for user customization
- **Feature flags**: Allow gradual feature rollout
- **Plugin architecture**: Design for future extensibility

---
*Development guide for the tofusort open source project.*
