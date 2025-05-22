# TypeScript/JavaScript API Documentation Generator

This tool uses the Oak API to automatically generate API reference documentation in Markdown format for TypeScript and JavaScript files.

## Features

- Detects function declarations, arrow functions, and class methods
- Extracts docstrings and parameter information
- Formats JSDoc comments (@param, @returns) as Markdown
- Handles TypeScript type annotations
- Detects exported vs non-exported functions
- Creates a table of contents with links to each function

## Usage

```bash
# Generate documentation for a single file
go run main.go path/to/file.ts

# Generate documentation for all TS/JS files in a directory
go run main.go path/to/directory
```

## Output

The tool generates Markdown output with the following structure:

```
# API Reference

## Table of Contents

- [File1](#file1)
  - [function1](#function1)
  - [function2](#function2)

## File1

### function1

_Exported_

Function description from docstring

```typescript
function1(param1: type1, param2: type2): returnType
```

**Parameters:**

- `param1` - _type1_
- `param2` - _type2_

**Returns:** _returnType_

_Defined in [file.ts:10]_
```

## Implementation Details

The tool uses Oak's tree-sitter integration to parse TypeScript/JavaScript code and extract:

1. Function declarations
2. Arrow function expressions (both exported and non-exported)
3. Class method definitions
4. Comments associated with functions
5. Parameter types and return types

For each function, it generates properly formatted Markdown documentation including:

- Function signature with parameters and return type
- Parameter list with types
- Return type information
- Export status
- Source location

## Dependencies

- Oak API for tree-sitter parsing
- Cobra for command-line interface
