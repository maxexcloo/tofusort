# Architecture Documentation

## Overview

Tofusort is a command-line tool designed to sort Terraform/OpenTofu configuration files alphabetically. The architecture follows Go best practices with a clean separation of concerns between CLI handling, parsing, and sorting logic.

## System Architecture

```mermaid
graph TD
    A[CLI Entry Point] --> B[Command Parser]
    B --> C[File Discovery]
    C --> D[HCL Parser]
    D --> E[Sorter Engine]
    E --> F[File Writer]
    F --> G[Output]
    
    subgraph "Core Components"
        D
        E
    end
    
    subgraph "CLI Layer"
        A
        B
        C
        F
        G
    end
    
    H[Sample Files] --> C
    I[Configuration] --> E
```

## Component Overview

### 1. CLI Layer (`cmd/tofusort/`)

**Purpose**: Handle command-line interface, file operations, and user interaction.

**Components**:
- `check.go`: Check command for CI/CD validation
- `main.go`: Application entry point and root command setup
- `sort.go`: Sort command implementation with file discovery and processing

**Responsibilities**:
- Coordinate parsing and sorting operations
- Discover files (single file, directory, recursive)
- Handle dry-run mode and output formatting
- Manage error reporting and exit codes
- Parse command-line arguments using Cobra

### 2. Parser Layer (`internal/parser/`)

**Purpose**: Handle HCL file parsing and formatting integration.

**Components**:
- `parser.go`: HCL parsing and formatting logic

**Responsibilities**:
- Clean up excessive blank lines
- Handle both `.tf` and `.tfvars` files
- Integrate with `tofu fmt` for consistent formatting
- Parse HCL files using `github.com/hashicorp/hcl/v2/hclwrite`
- Preserve comments and expressions

**Key Features**:
- AST-based parsing for reliable handling
- Comment preservation
- Expression integrity maintenance
- Format cleanup and standardization

### 3. Sorter Engine (`internal/sorter/`)

**Purpose**: Core sorting logic for Terraform constructs.

**Components**:
- `sorter.go`: Main sorting implementation
- `sorter_test.go`: Comprehensive test suite

**Responsibilities**:
- Handle meta-arguments (count, for_each, lifecycle, depends_on)
- Manage spacing and formatting rules
- Sort attributes within blocks
- Sort nested blocks recursively
- Sort top-level blocks by type and name

**Advanced Features**:
- **Attribute Categorization**: Early, regular, and late attributes
- **Dynamic Block Sorting**: Sort by label name, then `for_each` expression
- **Spacing Management**: Single-line vs multi-line attribute handling
- **Validation Block Sorting**: Sort by `error_message` content

## Data Flow

```mermaid
sequenceDiagram
    participant CLI as CLI Command
    participant Parser as HCL Parser
    participant Sorter as Sorter Engine
    participant Writer as File Writer
    
    CLI->>Parser: Parse HCL file
    Parser->>Parser: Build AST
    Parser->>Sorter: Pass file AST
    Sorter->>Sorter: Sort top-level blocks
    Sorter->>Sorter: Sort block attributes
    Sorter->>Sorter: Sort nested blocks
    Sorter->>Parser: Return sorted AST
    Parser->>Parser: Format and cleanup
    Parser->>Writer: Write formatted content
    Writer->>CLI: Return status
```

## Sorting Algorithm

### Block Type Priority System
```go
var blockTypeOrder = map[string]int{
    "terraform": 0, "provider": 1, "variable": 2, "locals": 3,
    "data": 4, "resource": 5, "module": 6, "output": 7,
}
```

### Meta-Argument Priority System
```go
var metaArgumentOrder = map[string]int{
    "count": 0, "for_each": 1,
    "lifecycle": 998, "depends_on": 999,
}
```

### Special Block Sorting Logic

**Validation Blocks**:
```go
func (s *Sorter) getValidationErrorMessage(block *hclwrite.Block) string {
    // Extracts error_message content for comparison
}
```

**Dynamic Blocks**:
```go
func (s *Sorter) getDynamicBlockLabel(block *hclwrite.Block) string {
    // Extracts label for primary sort
}
func (s *Sorter) getDynamicForEachContent(block *hclwrite.Block) string {
    // Extracts for_each expression for secondary sort  
}
```

## Key Design Decisions

### 1. AST-Based Parsing
- Uses `hclwrite` package for reliable AST manipulation
- Preserves all formatting, comments, and expressions
- Enables precise sorting without breaking Terraform syntax

### 2. Recursive Sorting
- Sorts nested blocks using the same rules as top-level blocks
- Maintains consistency across all nesting levels
- Special handling for validation and dynamic blocks

### 3. Spacing Management
- Groups single-line attributes together
- Adds blank lines before multi-line attributes and nested blocks
- Preserves original formatting intentions while enforcing consistency

### 4. Error Recovery
- Continues processing other files if one fails
- Provides detailed error reporting with file and line context
- Graceful handling of parsing errors

## Performance Considerations

- **Memory Efficiency**: Processes files individually to minimize memory usage
- **Concurrent Processing**: Ready for parallel file processing (future enhancement)
- **AST Reuse**: Minimal AST manipulation for better performance

## Dependencies

- `github.com/hashicorp/hcl/v2`: HCL parsing and manipulation
- `github.com/spf13/cobra`: CLI framework
- Standard Go library for file operations and string manipulation

## Testing Strategy

- **Integration Tests**: Sample files validate end-to-end functionality
- **Regression Tests**: Ensure changes don't break existing functionality
- **Unit Tests**: Comprehensive coverage for sorting logic
