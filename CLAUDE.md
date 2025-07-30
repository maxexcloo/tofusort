# CLAUDE.md - Development Guide

## Project Overview
**Purpose**: Tool to sort Terraform/OpenTofu configuration files alphabetically
**Status**: Active
**Language**: Go (for native HCL v2 parser integration)

## Code Standards

### Organization
- **Config/Data**: Alphabetical and recursive (imports, dependencies, object keys)
- **Documentation**: Sort sections, lists, and references alphabetically when logical
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
- **README.md**: Tool overview and usage guide
- **ARCHITECTURE.md**: Technical design and implementation details
- **CLAUDE.md**: Development standards and project guidelines (this file)

### Contribution Standards
- **Feature Changes**: Update README.md and ARCHITECTURE.md when adding features
- **Code Changes**: Follow sorting rules and maintain test coverage
- **Documentation**: Keep all docs synchronized and cross-referenced

## Command Interface Standards
- **Consistent flags**: Use standard Unix-style flags (-r, --dry-run)
- **Clear output**: Provide informative messages about what was processed
- **Exit codes**: 0 for success, 1 for failure, follow standard conventions
- **Error messages**: Include file names and line numbers where possible

## Development Workflow Standards

### Environment Management
- Use **mise** for consistent development environments
- Pin tool versions in `.mise.toml`
- Define common tasks as mise scripts

### Required Development Tasks
- **build**: Create production binary
- **test**: Run full test suite
- **lint**: Code quality checks
- **fmt**: Code formatting
- **check**: All validation (fmt + lint + test)
- **dev**: Development validation cycle

## Error Handling Standards
- **Graceful degradation**: Continue processing when individual files fail
- **Informative messages**: Include file paths and line numbers
- **Contextual errors**: Show surrounding code when possible
- **User-friendly output**: Clear explanations for common issues

## Extension Guidelines
- **Plugin architecture**: Design for future extensibility
- **Configuration files**: Plan for user customization
- **Backward compatibility**: Maintain API stability
- **Feature flags**: Allow gradual feature rollout

---
*Development guide for the tofusort open source project.*
