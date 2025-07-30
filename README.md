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

- **Dynamic blocks**: Sorted by label name, then by `for_each` expression
- **Meta-arguments**: `count`/`for_each` first, `lifecycle`/`depends_on` last
- **Multi-line attributes**: Proper spacing with blank lines
- **Validation blocks**: Sorted by `error_message` content

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
# Check if files are sorted (CI mode)
tofusort check main.tf

# Preview changes (dry run)
tofusort sort --dry-run main.tf

# Sort a single file
tofusort sort main.tf

# Sort directory recursively  
tofusort sort -r ./modules
```

### Development Commands

```bash
# Build binary
mise run build

# Format and lint
mise run fmt
mise run lint

# Run all checks
mise run check

# Test development build
mise run dev
```

## How It Works

Tofusort applies consistent sorting rules:
- **Block types**: terraform → provider → variable → locals → data → resource → module → output
- **Attributes**: Alphabetical with meta-argument priorities
- **Special handling**: Validation and dynamic blocks have custom sort logic
- **Spacing**: Automatic formatting with proper blank lines

> **📋 Complete sorting rules**: See [ARCHITECTURE.md](ARCHITECTURE.md#sorting-algorithm) for detailed specifications

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

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)**: Technical design, components, and algorithms
- **[CLAUDE.md](CLAUDE.md)**: Development standards and contribution guidelines

## Development

Quick start for contributors:

```bash
# Setup environment
mise install

# Run all checks
mise run check

# Development workflow
mise run dev
```

> **🛠️ Complete development guide**: See [CLAUDE.md](CLAUDE.md#development-workflow-standards) for detailed workflow

---

*A tool for maintaining consistent Terraform configuration organization.*
