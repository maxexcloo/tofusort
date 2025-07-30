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

## Project Structure
- **.mise.toml**: Development environment configuration
- **cmd/tofusort/**: Main CLI application
- **internal/parser/**: HCL parsing logic using hclwrite
- **internal/sorter/**: Sorting algorithms for different block types
- **internal/formatter/**: Integration with tofu fmt
- **sample/**: Test Terraform files for validation
- **go.mod**: Go module dependencies

## Project Specs

### Core Features
- **Block sorting**: Alphabetical by type (data, locals, module, output, provider, resource, terraform, variable)
- **CLI interface**: Support for single files, directories, and recursive operation
- **Comment preservation**: Maintain all comments in their relative positions
- **File support**: .tf and .tfvars files (both HCL and JSON syntax)
- **Meta-arguments**: count/for_each first with blank line, lifecycle/depends_on last
- **Nested sorting**: Recursive alphabetical sorting of all nested structures
- **Parsing strategy**: HashiCorp's HCL v2 native parser (hclwrite package)
- **Spacing rules**: Single-line attributes grouped, multi-line attributes with blank lines
- **tofu fmt integration**: Pre and post formatting for consistency

### Technical Implementation
- **Parser**: Use `github.com/hashicorp/hcl/v2/hclwrite` for AST manipulation
- **AST preservation**: Maintain formatting, comments, and expressions as-is
- **Block identification**: Parse blocks by type and label for sorting
- **Attribute handling**: Sort attributes within blocks while preserving expressions
- **Error recovery**: Continue processing other files if one fails

### Sorting Rules
1. **Top-level blocks**: Sort by type, then by name within type
2. **Block type order**: terraform → provider → variable → locals → data → resource → module → output
3. **Within blocks**: Sort attributes alphabetically, except meta-arguments
4. **Meta-argument order**: count/for_each → other attributes → lifecycle → depends_on
5. **Nested blocks**: Apply same rules recursively
   - **validation blocks**: Sort by error_message content alphabetically
   - **dynamic blocks**: Sort by label name, then by for_each expression content
6. **Spacing**: Single-line attributes grouped, multi-line attributes with blank lines

## Environment Setup

### Using mise
The project uses [mise](https://mise.jdx.dev/) for consistent development environments:

```bash
# Install mise (if not already installed)
curl https://mise.run | sh

# Initialize environment
mise install

# Verify setup
mise list
```

**.mise.toml** configuration:
```toml
[tools]
go = "1.23"
golangci-lint = "latest"
goreleaser = "latest"

[env]
GOPATH = "{{env.HOME}}/go"
PATH = "{{env.GOPATH}}/bin:{{env.PATH}}"
```

## Dependencies
```go
// go.mod
module github.com/yourusername/tofusort

go 1.23

require (
    github.com/hashicorp/hcl/v2 v2.24.0
    github.com/spf13/cobra v1.9.1
)
```

## CLI Design
```bash
# Sort a single file
tofusort sort main.tf

# Sort directory recursively
tofusort sort -r ./modules

# Dry run to see what would change
tofusort sort --dry-run main.tf

# Custom spacing rules
tofusort sort --compact main.tf

# Check if files are sorted (CI mode)
tofusort check main.tf
```

## Development Workflow
```bash
# Initial setup
mise install  # Installs Go and tools defined in .mise.toml
go mod download

# Test
go test ./...

# Run locally
go run ./cmd/tofusort sort sample/example.tf

# Build
go build -o tofusort ./cmd/tofusort

# Lint
golangci-lint run

# Format
go fmt ./...
```

## Error Handling
- **Invalid HCL**: Report file and line number, skip file
- **File permissions**: Report access errors clearly
- **Parsing failures**: Show context and continue with other files
- **Format conflicts**: Warn when tofu fmt would conflict with sorting

## Future Considerations
- **Object literal sorting**: Sort keys within HCL object expressions and jsonencode() calls
- **Array element sorting**: Sort elements in simple array literals
- **Custom sort orders**: Configuration file for project-specific rules
- **Semantic grouping**: Option to group related resources
- **Import organization**: Sort import blocks by source
- **Provider grouping**: Keep provider configurations with their resources

---
*Development guide for the tofusort open source project.*
