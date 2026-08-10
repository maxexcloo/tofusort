# tofusort

[![Licence](https://img.shields.io/badge/licence-AGPL--3.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-active-success)](https://img.shields.io/badge/status-active-success)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](Dockerfile)
[![Go](https://img.shields.io/badge/go-blue.svg)](https://go.dev/)

Sort OpenTofu and Terraform HCL configuration files using the native HCL v2
parser.

## Quick Start

```bash
mise install
mise run build
```

## Features

- **Attribute sorting**: Alphabetical within blocks, with meta-argument ordering
- **Block sorting**: Alphabetical by type (terraform → provider → variable → locals → data → resource → module → output)
- **Comment preservation**: Maintains all comments in their relative positions
- **File support**: Handles HCL-format `.tf` and `.tfvars` files
- **Nested sorting**: Recursive alphabetical sorting of all nested structures
- **Spacing management**: Automatic formatting with proper blank line handling

### Advanced Features

- **Dynamic blocks**: Sorted by label name, then by `for_each` expression
- **Meta-arguments**: `count`/`for_each` first; dependency and lifecycle fields last
- **Multi-line attributes**: Proper spacing with blank lines
- **Validation blocks**: Sorted by `error_message` content

Visit `./tofusort --help` and start sorting your OpenTofu/Terraform files.

## Installation

### Local Development

```bash
git clone https://github.com/maxexcloo/tofusort.git
cd tofusort

# Install dependencies
mise install

# Build the binary
mise run build
```

### Docker

```bash
docker build -t tofusort .
docker run --rm -v "$(pwd):/workspace" -w /workspace tofusort sort main.tf
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

# Sort a directory recursively
tofusort sort -r ./modules
```

### Development Commands

```bash
# Build binary
mise run build

# Run all checks
mise run check

# Development validation cycle
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
3. Make changes following the code standards in `AGENTS.md`
4. Build and test: `mise run check`
5. Submit a pull request

## Documentation

- **[AGENTS.md](AGENTS.md)**: Development standards and contribution guidelines
- **[Architecture](docs/architecture.md)**: Technical design, components, and algorithms

---

_A tool for maintaining consistently organised OpenTofu and Terraform configuration._
