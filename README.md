# tofusort

A tool to sort Terraform/OpenTofu configuration files alphabetically for better organization and consistency.

## Features

- **Block sorting**: Sorts blocks by type (terraform → provider → variable → locals → data → resource → module → output)
- **Attribute sorting**: Sorts attributes within blocks alphabetically, with special handling for meta-arguments
- **Comment preservation**: Maintains all comments in their relative positions
- **File support**: Handles both `.tf` and `.tfvars` files
- **CLI interface**: Command-line tool with sort and check commands
- **Recursive operation**: Process entire directory trees

## Installation

Clone the repository and build:

```bash
git clone <repository-url>
cd tofusort
go build -o tofusort ./cmd/tofusort
```

## Usage

### Sort files

Sort a single file:
```bash
tofusort sort main.tf
```

Sort all files in a directory:
```bash
tofusort sort ./
```

Sort recursively:
```bash
tofusort sort -r ./modules
```

Preview changes without modifying files:
```bash
tofusort sort --dry-run main.tf
```

### Check if files are sorted

Check if files are already sorted (useful for CI):
```bash
tofusort check main.tf
```

Returns exit code 0 if sorted, 1 if any files need sorting.

## Sorting Rules

1. **Block types** are sorted in this order:
   - `terraform`
   - `provider`
   - `variable`
   - `locals`
   - `data`
   - `resource`
   - `module`
   - `output`

2. **Within each block type**, blocks are sorted alphabetically by name

3. **Attributes within blocks** are sorted alphabetically, except:
   - `count` and `for_each` appear first
   - `lifecycle` and `depends_on` appear last

4. **Nested blocks** are recursively sorted using the same rules

## Development

This project uses [mise](https://mise.jdx.dev/) for development environment management:

```bash
# Install dependencies
mise install

# Run tests
go test ./...

# Build
go build -o tofusort ./cmd/tofusort

# Format code
go fmt ./...
```

## Example

Before:
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

After:
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

## License

[Add your license here]