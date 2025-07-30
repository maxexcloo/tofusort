# tofusort

**Purpose**: Tool to sort Terraform/OpenTofu configuration files alphabetically  
**Status**: Active  
**Language**: Go

## Features

- **Block sorting**: Alphabetical by type (terraform → provider → variable → locals → data → resource → module → output)
- **Attribute sorting**: Alphabetical within blocks, with meta-argument ordering
- **Comment preservation**: Maintains all comments in their relative positions
- **File support**: Handles `.tf` and `.tfvars` files (HCL and JSON syntax)
- **Nested sorting**: Recursive alphabetical sorting of all nested structures
- **Spacing management**: Automatic formatting with proper blank line handling

### Advanced Features

- **Validation blocks**: Sorted by `error_message` content
- **Dynamic blocks**: Sorted by label name, then by `for_each` expression
- **Meta-arguments**: `count`/`for_each` first, `lifecycle`/`depends_on` last
- **Multi-line attributes**: Proper spacing with blank lines

## Installation

```bash
git clone <repository-url>
cd tofusort
mise install
mise run build
```

## Usage

### Basic Commands

```bash
# Sort a single file
tofusort sort main.tf

# Sort directory recursively  
tofusort sort -r ./modules

# Preview changes (dry run)
tofusort sort --dry-run main.tf

# Check if files are sorted (CI mode)
tofusort check main.tf
```

### Development Commands

```bash
# Build binary
mise run build

# Run all checks
mise run check

# Format and lint
mise run fmt
mise run lint

# Test development build
mise run dev
```

## Sorting Rules

### Block Types
1. `terraform` → `provider` → `variable` → `locals` → `data` → `resource` → `module` → `output`
2. Within types: alphabetical by name
3. Nested blocks: recursive application of same rules

### Attributes
1. **Early meta-arguments**: `count`, `for_each` (with blank line after)
2. **Regular attributes**: alphabetical order
3. **Late meta-arguments**: `lifecycle`, `depends_on`

### Special Blocks
- **Validation**: Sorted by `error_message` content
- **Dynamic**: Sorted by label, then `for_each` expression
- **Lifecycle**: Always appears last within parent block

### Spacing
- Single-line attributes: grouped together
- Multi-line attributes: blank line before each
- Nested blocks: blank line before each

## Example

**Before**:
```hcl
provider "github" {
  token = var.github_token
}

provider "aws" {
  region = "us-west-2"
}

variable "github_token" {
  type = string
}
```

**After**:
```hcl
variable "github_token" {
  type = string
}

provider "aws" {
  region = "us-west-2"
}

provider "github" {
  token = var.github_token
}
```

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed system design and component documentation.

## Development

### Environment Setup
```bash
# Install tools and dependencies
mise install

# Run development build and test
mise run dev

# Full check (format, lint, test)
mise run check
```

### Code Standards
- **KISS principle**: Keep implementations simple and readable
- **Error recovery**: Continue processing when individual files fail
- **Testing**: Unit tests for all sorting logic
- **Documentation**: Update docs with every feature change

## Future Enhancements

- **Object literal sorting**: Sort keys within HCL objects and `jsonencode()` calls
- **Array element sorting**: Sort elements in simple array literals
- **Custom sort orders**: Configuration file for project-specific rules
- **Semantic grouping**: Option to group related resources together

---

*A tool for maintaining consistent Terraform configuration organization.*