# CLAUDE.md - Development Guide

## Project Overview
**Purpose**: Infrastructure as Code using OpenTofu for personal cloud infrastructure  
**Status**: Active

## Commands
```bash
# Development
tofu fmt         # Format configuration
tofu validate    # Validate configuration
tofu plan        # Plan changes

# Build
tofu apply       # Apply configuration
```

## Tech Stack
- **Language**: HCL (HashiCorp Configuration Language)
- **Framework**: OpenTofu
- **Testing**: tofu validate and tofu plan

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
- **Formatting**: Run formatter before commits
- **KISS principle**: Keep it simple - prefer readable code over clever code
- **Naming**: snake_case for all resources and variables
- **Trailing newlines**: Required in all files

## Project Structure
- **data.tf**: All data sources
- **locals_*.tf**: All locals (prefixed by filename)
- **outputs.tf**: Output definitions
- **providers.tf**: Provider configurations
- **terraform.tf**: Terraform configuration
- **variables.tf**: Variable definitions
- ***.tf**: Resource files
- **terraform.tfvars**: Instance values

## Project Specs
- **Consolidate defaults**: Use `var.default` structure for default values
- **Locals prefix**: Locals in `locals_*.tf` files must start with filename prefix
- **No comments**: Code is self-explanatory
- **Sorting order**: Key order within blocks: 1) count/for_each (with blank line after), 2) Simple values (strings, numbers, bools, null), 3) Complex values (arrays, objects, maps)
- **Type definitions**: Use `type = any` for complex nested structures

## README Guidelines
- **Structure**: Title → Description → Quick Start → Features → Installation → Usage → Contributing
- **Badges**: Include relevant status badges (build, version, license)
- **Code examples**: Always include working examples in code blocks
- **Installation**: Provide copy-paste commands that work
- **Quick Start**: Get users running in under 5 minutes

## Git Workflow
```bash
# After every change
tofu fmt && tofu validate && tofu plan
git add . && git commit -m "type: description"

# Always commit after verified working changes
# Keep commits small and focused
```

---

*Simple context for AI assistants working on this open source project.*
