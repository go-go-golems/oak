---
Title: Guru Symbol Queries
Slug: guru
Short: Run guru queries using symbol names instead of byte offsets, with automatic symbol resolution via Tree-sitter
Topics:
  - guru
  - static-analysis
  - go
  - symbol-resolution
  - code-analysis
Commands:
  - guru
Flags:
  - --mode
  - --symbol
  - --file
  - --json
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# Guru Symbol Queries

## Overview

The `guru` command enables semantic code analysis by running Go guru queries using symbol names instead of manual byte offsets. Unlike traditional guru usage that requires finding exact byte positions, this command automatically locates symbols in your codebase using Tree-sitter parsing, then executes guru queries to find references, callers, implementers, and other semantic relationships. This integration combines Tree-sitter's fast structural parsing with guru's deep semantic analysis capabilities, making code exploration and refactoring significantly more accessible.

The command outputs structured data that can be formatted as JSON, YAML, CSV, or tables, enabling both interactive exploration and programmatic processing of code relationships.

## Basic Usage

The `guru` command requires three parameters: the query mode, the symbol name, and the file containing that symbol. Tree-sitter automatically finds the symbol's position, eliminating the need to manually calculate byte offsets.

```bash
oak guru <mode> <symbol> --file <filepath>
```

**Required parameters:**
- `mode`: The type of guru query to execute (referrers, callees, implements, etc.)
- `symbol`: The name of the symbol to query (function, type, method, variable, or constant)
- `--file`: Path to the file containing the symbol

**Example:**
```bash
# Find all references to ProcessData function
oak guru referrers ProcessData --file pkg/processor/data.go
```

## Query Modes

Guru provides several query modes, each answering different questions about your codebase. The `guru` command supports all standard guru modes, enabling comprehensive code analysis workflows.

### referrers

Finds all references to a symbol across your codebase. This is essential for understanding where a function, type, or variable is used before refactoring or renaming.

```bash
oak guru referrers ProcessData --file pkg/processor/data.go
```

**Use cases:**
- Impact analysis before renaming symbols
- Finding all usages of a function or type
- Understanding symbol dependencies

### callees

Shows all possible callers of a function. This helps identify which functions call a specific function, enabling call graph analysis and impact assessment.

```bash
oak guru callees Query --file pkg/database/db.go
```

**Use cases:**
- Finding all callers of a function
- Understanding function call relationships
- Identifying code that needs updates when a function signature changes

### implements

Finds all types that implement a specific interface. This is crucial for understanding interface adoption and finding all implementations when modifying interface contracts.

```bash
oak guru implements Closer --file io/io.go
```

**Use cases:**
- Finding all implementations of an interface
- Impact analysis when changing interface definitions
- Discovering types that satisfy an interface contract

### definition

Shows the declaration of a selected identifier. This quickly locates where a symbol is defined, useful for navigating large codebases.

```bash
oak guru definition ProcessData --file pkg/processor/data.go
```

### describe

Provides comprehensive information about a symbol, including its definition, methods, and related types. This gives a complete picture of a symbol's structure and relationships.

```bash
oak guru describe ProcessData --file pkg/processor/data.go
```

### freevars

Shows free variables in a selection. This helps identify variables that are referenced but not defined within a specific code block.

```bash
oak guru freevars --file pkg/processor/data.go
```

### peers

Shows send/receive operations corresponding to a channel operation. This is essential for understanding channel communication patterns and finding related goroutines.

```bash
oak guru peers --file pkg/concurrency/channels.go
```

### what

Shows basic information about the selected syntax node. This provides quick context about what a symbol represents.

```bash
oak guru what ProcessData --file pkg/processor/data.go
```

### callstack

Shows the path from the callgraph root to a selected function. This enables understanding the call chain that leads to a function execution.

```bash
oak guru callstack ProcessData --file pkg/processor/data.go
```

## Output Formats

The `guru` command outputs structured data that integrates with Glazed's formatting system. By default, results are displayed as a formatted table, but you can request JSON output for programmatic processing.

### Table Output (Default)

```bash
oak guru referrers ProcessData --file pkg/processor/data.go
```

**Output:**
```
+----------------------+------+--------+------------------+--------+
| file                 | line | column | text              | symbol |
+----------------------+------+--------+------------------+--------+
| pkg/processor/main.go| 42   | 15     | ProcessData(...) | ProcessData |
| pkg/processor/util.go| 18   | 8      | ProcessData(...) | ProcessData |
+----------------------+------+--------+------------------+--------+
```

### JSON Output

Use the `--json` flag to output results as JSON, enabling easy integration with scripts and tools:

```bash
oak guru referrers ProcessData --file pkg/processor/data.go --json
```

**Output:**
```json
[
  {
    "file": "pkg/processor/main.go",
    "line": 42,
    "column": 15,
    "text": "ProcessData(...)",
    "symbol": "ProcessData",
    "mode": "referrers",
    "symbol_file": "pkg/processor/data.go",
    "symbol_type": "function"
  }
]
```

### Using Glazed Output Options

Since `guru` is a Glazed command, you can use all standard Glazed output formatting options:

```bash
# Output as YAML
oak guru referrers ProcessData --file pkg/processor/data.go --output yaml

# Output as CSV
oak guru referrers ProcessData --file pkg/processor/data.go --output csv

# Select specific fields
oak guru referrers ProcessData --file pkg/processor/data.go --fields file,line,text

# Sort by line number
oak guru referrers ProcessData --file pkg/processor/data.go --sort-columns line
```

## Symbol Resolution

The `guru` command uses Tree-sitter to automatically locate symbols in your code, eliminating the manual step of finding byte offsets. The command searches for symbols across multiple declaration types to ensure comprehensive symbol resolution.

**Supported symbol types:**
- Functions (`function_declaration`)
- Types (`type_declaration`)
- Methods (`method_declaration`)
- Variables (`var_declaration`)
- Constants (`const_declaration`)

When you provide a symbol name and file path, Tree-sitter parses the file, searches for the symbol across all supported declaration types, and extracts the byte offset needed for guru queries. This process happens automatically, making the command much more user-friendly than traditional guru usage.

**Example workflow:**
1. You specify `ProcessData` as the symbol and `pkg/processor/data.go` as the file
2. Tree-sitter parses the file and finds the `function_declaration` node named `ProcessData`
3. The command extracts the byte offset (e.g., `1234`)
4. Guru is called with `pkg/processor/data.go:#1234`
5. Results are parsed and formatted according to your output preferences

## Common Workflows

### Finding All Usages Before Refactoring

Before renaming or modifying a function, find all its references:

```bash
oak guru referrers ProcessData --file pkg/processor/data.go --output json > references.json
```

This creates a JSON file listing all files and locations that reference `ProcessData`, enabling systematic refactoring.

### Discovering Interface Implementations

When modifying an interface, find all types that implement it:

```bash
oak guru implements Closer --file io/io.go
```

This shows all types that implement `io.Closer`, helping you understand the impact of interface changes.

### Building Call Graphs

Trace function call relationships to understand code flow:

```bash
# Find all callers of a function
oak guru callees Query --file pkg/database/db.go

# Find the call stack leading to a function
oak guru callstack ProcessData --file pkg/processor/data.go
```

### Code Review Assistance

During code reviews, quickly understand symbol relationships:

```bash
# Check what a function calls
oak guru callees ProcessData --file pkg/processor/data.go

# See where a function is used
oak guru referrers ProcessData --file pkg/processor/data.go

# Get complete symbol information
oak guru describe ProcessData --file pkg/processor/data.go
```

## Integration with Other Oak Commands

The `guru` command complements other oak commands for comprehensive code analysis:

- **Tree-sitter queries** (`oak run`): Find structural patterns in code
- **PAIP patterns** (`oak pattern`): Match complex logical patterns in ASTs
- **Guru queries** (`oak guru`): Analyze semantic relationships and references

Together, these tools provide a complete code analysis toolkit that combines structural parsing, pattern matching, and semantic analysis.

## Troubleshooting

### Symbol Not Found

If the command reports that a symbol is not found, verify:

1. **Correct symbol name**: Ensure the symbol name matches exactly (case-sensitive)
2. **Correct file path**: The file must contain the symbol definition
3. **Symbol type**: The symbol must be a function, type, method, variable, or constant

```bash
# Verify the symbol exists using Tree-sitter
oak ast pkg/processor/data.go --language go --format lisp | grep ProcessData
```

### Guru Command Not Found

Ensure the `guru` tool is installed and available in your PATH:

```bash
which guru
# Should output: /path/to/guru

# If not found, install guru:
go install golang.org/x/tools/cmd/guru@latest
```

### Empty Results

Some guru modes may return empty results if:
- The symbol has no references (for `referrers`)
- No types implement the interface (for `implements`)
- The function has no callers (for `callees`)

This is expected behavior and indicates the symbol has no relationships of the queried type.

## Related Documentation

For more information about related oak capabilities:

```
oak help pattern
```

Learn about PAIP pattern matching for complex AST queries.

```
oak help create-query
```

Understand how to create custom Tree-sitter queries for structural analysis.

