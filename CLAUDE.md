# CLAUDE.md - Development Guide

## Project Overview
**Purpose**: Tool to sort OpenTofu/Terraform configuration files alphabetically
**Status**: Active
**Language**: Go (for native HCL v2 parser integration)

## Code Standards

### Organization
- **Config/Data**: Alphabetical and recursive (imports, dependencies, object keys, mise tasks)
- **Documentation**: Sort sections, lists, and references alphabetically when logical
- **Files**: Alphabetical in documentation and directories
- **Functions**: Group by purpose, alphabetical within groups
- **Variables**: Alphabetical within scope

### Quality
- **Comments**: Minimal - only for complex business logic
- **Documentation**: Update ARCHITECTURE.md and README.md with every feature change
- **Error handling**: Always handle parser errors gracefully
- **Formatting**: Run `mise run fmt` before commits
- **KISS principle**: Keep it simple - prefer readable code over clever code
- **Naming**: Go conventions (camelCase for functions, PascalCase for types)
- **Testing**: Unit tests for all sorting logic
- **Trailing newlines**: Required in all files

## Commands
```bash
# Build
mise run build       # Create production binary

# Development
mise run dev         # Development validation cycle
mise run test        # Run full test suite

# Format
mise run fmt         # Code formatting

# Check
mise run check       # All validation (fmt + lint + test)
```

## Development Guidelines

### Contribution Standards
- **Code Changes**: Follow sorting rules and maintain test coverage
- **Documentation**: Keep all docs synchronized and cross-referenced
- **Feature Changes**: Update README.md and ARCHITECTURE.md when adding features

### Documentation Structure
- **ARCHITECTURE.md**: Technical design and implementation details
- **CLAUDE.md**: Development standards and project guidelines
- **README.md**: Tool overview and usage guide

## Command Interface Standards
- **Clear output**: Provide informative messages about what was processed
- **Consistent flags**: Use standard Unix-style flags (-r, --dry-run)
- **Error messages**: Include file names and line numbers where possible
- **Exit codes**: 0 for success, 1 for failure, follow standard conventions

## Development Workflow Standards

### Environment Management
- Define common tasks as mise scripts
- Pin tool versions in `.mise.toml`
- Use **mise** for consistent development environments

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

## Project Structure
- **.mise.toml**: mise configuration for tool versioning
- **cmd/tofusort/**: CLI layer with main, sort, and check commands
- **go.mod**: Go module dependencies
- **internal/parser/**: HCL parsing and formatting logic
- **internal/sorter/**: Core sorting engine with comprehensive test suite
- **samples/**: Sample OpenTofu/Terraform files for testing

## README Guidelines
- **Badges**: Include relevant status badges (license, status, language, docker)
- **Code examples**: Always include working examples in code blocks
- **Installation**: Provide copy-paste commands that work
- **Quick Start**: Get users running in under 5 minutes
- **Structure**: Title → Badges → Description → Quick Start → Features → Installation → Usage → Contributing

## Tech Stack
- **Backend**: Go for native HCL v2 parser integration
- **CLI**: Cobra framework for command-line interface
- **Testing**: Go unit tests and integration tests

---

*Development guide for the tofusort open source project.*
