# ARCHITECTURE.md - Technical Design

## Overview

Command-line tool for sorting Terraform/OpenTofu configuration files alphabetically using native HCL v2 parser integration.

## Core Components

### CLI Layer
- **Entry Point**: Cobra-based command-line interface
- **Commands**: Main, sort, and check commands
- **File Discovery**: Single file, directory, and recursive processing
- **Output**: Dry-run mode and formatted output

### Parser Layer
- **HCL Integration**: Native `hclwrite` package for AST manipulation
- **File Support**: `.tf` and `.tfvars` files (HCL and JSON syntax)
- **Comment Preservation**: Maintains all comments and expressions
- **Format Cleanup**: Removes excessive blank lines, standardizes formatting

### Sorter Engine
- **Block Sorting**: Terraform → provider → variable → locals → data → resource → module → output
- **Attribute Sorting**: Alphabetical with meta-argument priorities
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
- **Validation Blocks**: Sorted by `error_message` content
- **Dynamic Blocks**: Sorted by label name, then `for_each` expression
- **Multi-line Attributes**: Proper spacing with blank lines

## Technology Stack

### Core
- **Language**: Go with native HCL v2 parser
- **CLI**: Cobra framework for command-line interface
- **Parser**: `github.com/hashicorp/hcl/v2` for AST manipulation
- **Testing**: Go unit tests and integration tests

### Dependencies
- **HCL Parser**: Native integration with HashiCorp HCL v2
- **CLI Framework**: Cobra for robust command-line handling
- **File Operations**: Standard Go library for file system access

---

*Technical architecture documentation for the tofusort project.*
