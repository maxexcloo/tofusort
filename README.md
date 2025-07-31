# tofusort

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-active-success)](https://img.shields.io/badge/status-active-success)
[![Go](https://img.shields.io/badge/go-blue.svg)](https://go.dev/)

Tool to sort Terraform/OpenTofu configuration files alphabetically using native HCL v2 parser integration.

## Features

- **Attribute sorting**: Alphabetical within blocks, with meta-argument ordering
- **Block sorting**: Alphabetical by type (terraform → provider → variable → locals → data → resource → module → output)
- **Comment preservation**: Maintains all comments in their relative positions
- **File support**: Handles `.tf` and `.tfvars` files (HCL and JSON syntax)
- **Nested sorting**: Recursive alphabetical sorting of all nested structures
- **Spacing management**: Automatic formatting with proper blank line handling

### Advanced Features

- **Dynamic blocks**: Sorted by label name, then by `for_each` expression
- **Meta-arguments**: `count`/`for_each` first, `lifecycle`/`depends_on` last
- **Multi-line attributes**: Proper spacing with blank lines
- **Validation blocks**: Sorted by `error_message` content

## Quick Start

Get up and running in under 5 minutes:

```bash
git clone <repository-url>
cd tofusort
mise install
mise run build
```

Visit `./tofusort --help` and start sorting your Terraform files.

## Installation

### Local Development

```bash
# Clone the repository
git clone <repository-url>
cd tofusort

# Install dependencies
mise install

# Build the binary
mise run build
```

### Docker

```bash
# Using Docker (when available)
docker build -t tofusort .
docker run -v $(pwd):/workspace -w /workspace tofusort sort main.tf
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

# Run all checks
mise run check

# Test development build
mise run dev

# Format and lint
mise run fmt
mise run lint
```

## How It Works

tofusort applies consistent sorting rules:
- **Attributes**: Alphabetical with meta-argument priorities
- **Block types**: terraform → provider → variable → locals → data → resource → module → output
- **Spacing**: Automatic formatting with proper blank lines
- **Special handling**: Validation and dynamic blocks have custom sort logic

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes following the code standards in CLAUDE.md
4. Build and test: `mise run check`
5. Submit a pull request

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)**: Technical design, components, and algorithms
- **[CLAUDE.md](CLAUDE.md)**: Development standards and contribution guidelines

---

*A tool for maintaining consistent Terraform configuration organization.*
