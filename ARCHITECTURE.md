# ARCHITECTURE.md - Technical Design

## Overview

Command-line tool for sorting OpenTofu/Terraform configuration files alphabetically using native HCL v2 parser integration.

## Core Components

### CLI Layer
- **Commands**: Main, sort, and check commands
- **Entry Point**: Cobra-based command-line interface
- **File Discovery**: Single file, directory, and recursive processing
- **Output**: Dry-run mode and formatted output

### Parser Layer
- **Comment Preservation**: Maintains all comments and expressions
- **File Support**: `.tf` and `.tfvars` files (HCL and JSON syntax)
- **Format Cleanup**: Removes excessive blank lines, standardizes formatting
- **HCL Integration**: Native `hclwrite` package for AST manipulation

### Sorter Engine
- **Attribute Sorting**: Alphabetical with meta-argument priorities
- **Block Sorting**: terraform → provider → variable → locals → data → resource → module → output
- **Nested Sorting**: Recursive sorting of all nested structures
- **Special Cases**: Validation and dynamic blocks with custom logic

## Data Flow

1. **Processing**: CLI command → File discovery → HCL parse → Sort → Format → Write output
2. **Sorting**: Parse AST → Sort top-level blocks → Sort attributes → Sort nested blocks → Format → Return
3. **Output**: Sorted AST → Cleanup formatting → Generate content → Write to file/stdout

## Key Algorithms

### Block Type Priority
```go
var blockTypeOrder = map[string]int{
    "terraform": 0, "provider": 1, "variable": 2, "locals": 3,
    "data": 4, "resource": 5, "module": 6, "output": 7,
}
```

### Meta-Argument Priority
```go
var metaArgumentOrder = map[string]int{
    "count": 0, "for_each": 1,
    "lifecycle": 998, "depends_on": 999,
}
```

### Special Block Handling
- **Dynamic Blocks**: Sorted by label name, then `for_each` expression
- **Multi-line Attributes**: Proper spacing with blank lines
- **Validation Blocks**: Sorted by `error_message` content

## Technology Stack

### Core
- **CLI**: Cobra framework for command-line interface
- **Language**: Go with native HCL v2 parser
- **Parser**: `github.com/hashicorp/hcl/v2` for AST manipulation
- **Testing**: Go unit tests and integration tests

### Dependencies
- **CLI Framework**: Cobra for robust command-line handling
- **File Operations**: Standard Go library for file system access
- **HCL Parser**: Native integration with HashiCorp HCL v2

---

*Technical architecture documentation for the tofusort project.*
