# Oak Programmatic API

## Introduction

The Oak Programmatic API provides a clean, type-safe interface for working with tree-sitter queries in Go applications. Instead of using YAML files for queries, this API allows you to construct and execute tree-sitter queries directly in your Go code, with typed results processing.

The API is designed around three core concepts:

1. **Query Building**: Constructing tree-sitter queries with a fluent interface
2. **Template Processing**: Formatting results with Go templates
3. **Programmatic Processing**: Processing results with typed Go functions

This guide covers everything you need to know to use the Oak API effectively in your applications.

## Installation

The Oak API is part of the Oak package. To use it in your project:

```bash
go get github.com/go-go-golems/oak@latest
```

Then import the API package in your code:

```go
import "github.com/go-go-golems/oak/pkg/api"
```

## Core Concepts

### Query Building

The central type in the API is the `QueryBuilder`, which allows you to construct tree-sitter queries programmatically using a functional options pattern.

```go
// Create a query builder for Go code
query := api.NewQueryBuilder(
    api.WithLanguage("go"),
    api.WithQuery("functions", `
        (function_declaration
         name: (identifier) @functionName
         parameters: (parameter_list) @parameters
         body: (block)? @body)
    `),
    api.WithQuery("methods", `
        (method_declaration
         receiver: (parameter_list) @receiver
         name: (field_identifier) @methodName
         parameters: (parameter_list) @parameters
         body: (block)? @body)
    `),
)
```

You can build a query from multiple sources:

- Inline query strings with `WithQuery`
- Files with `WithQueryFromFile`
- Existing Oak YAML files with `FromYAML`

### Query Execution

Once you've built a query, you can execute it on one or more files:

```go
// Run the query on specific files
results, err := query.Run(
    context.Background(),
    api.WithFiles([]string{"main.go", "utils.go"}),
)

// Or use glob patterns
results, err := query.Run(
    context.Background(),
    api.WithGlob("src/**/*.go"),
)

// Or scan directories recursively
results, err := query.Run(
    context.Background(),
    api.WithDirectory("pkg", api.WithRecursive(true)),
)
```

### Result Processing

The API provides two ways to process results:

#### 1. Template-Based Processing

For simple cases, you can format results using Go templates:

```go
output, err := query.RunWithTemplate(
    context.Background(),
    `
    # Functions in {{ .Language }} Files

    {{ range $file, $results := .ResultsByFile }}
    ## {{ $file }}

    {{ if index $results "functions" }}
    ### Functions
    {{ range (index $results "functions").Matches }}
    - func {{ (index . "functionName").Text }}{{ (index . "parameters").Text }}
    {{ end }}
    {{ end }}
    {{ end }}
    `,
    api.WithFiles([]string{"main.go"}),
)
```

#### 2. Programmatic Processing

For more complex cases, you can process results with a Go function that converts the raw results into typed structures:

```go
// Define a custom result type
type Function struct {
    Name       string
    Parameters string
    SourceFile string
    LineNumber int
}

// Process results into typed structures
result, err := query.RunWithProcessor(
    context.Background(),
    func(results api.QueryResults) (any, error) {
        var functions []Function

        for fileName, fileResults := range results {
            // Process functions
            if funcResults, ok := fileResults["functions"]; ok {
                for _, match := range funcResults.Matches {
                    functions = append(functions, Function{
                        Name:       match["functionName"].Text,
                        Parameters: match["parameters"].Text,
                        SourceFile: fileName,
                        LineNumber: int(match["functionName"].StartPoint.Row) + 1,
                    })
                }
            }
        }

        return functions, nil
    },
    api.WithFiles([]string{"main.go"}),
)

// Type assertion to use the result
functions, ok := result.([]Function)
if !ok {
    // Handle error
}
```

## API Reference

### QueryBuilder

```go
type QueryBuilder struct {}

// Create a new query builder
func NewQueryBuilder(options ...QueryOption) *QueryBuilder

// Run queries and return raw results
func (qb *QueryBuilder) Run(ctx context.Context, options ...RunOption) (QueryResults, error)

// Run queries and process results with a template
func (qb *QueryBuilder) RunWithTemplate(ctx context.Context, templateText string, options ...RunOption) (string, error)

// Run queries and process results with a template file
func (qb *QueryBuilder) RunWithTemplateFile(ctx context.Context, templatePath string, options ...RunOption) (string, error)

// Run queries and process results with a processor function
func (qb *QueryBuilder) RunWithProcessor(ctx context.Context, processor any, options ...RunOption) (any, error)
```

### QueryOptions

```go
// Set the language for the query
func WithLanguage(language string) QueryOption

// Add a named query
func WithQuery(name, query string) QueryOption

// Add a query from a file
func WithQueryFromFile(name, path string) QueryOption

// Load queries from an Oak YAML file
func FromYAML(path string) QueryOption
```

### RunOptions

```go
// Specify files to run queries on
func WithFiles(files []string) RunOption

// Specify a glob pattern to find files
func WithGlob(pattern string) RunOption

// Specify a directory to scan for files
func WithDirectory(dir string) RunOption

// Enable recursive directory scanning
func WithRecursive(recursive bool) RunOption

// Set the maximum number of worker goroutines
func WithMaxWorkers(n int) RunOption
```

### Result Types

```go
// QueryResults maps filenames to query results
type QueryResults map[string]map[string]*tree_sitter.Result

// Result contains matches for a specific query
type tree_sitter.Result struct {
    QueryName string
    Matches   []tree_sitter.Match
}

// Match maps capture names to captured nodes
type tree_sitter.Match map[string]tree_sitter.Capture

// Capture represents a captured node in the syntax tree
type tree_sitter.Capture struct {
    Name       string
    Text       string
    Type       string
    StartByte  uint32
    EndByte    uint32
    StartPoint tree_sitter.Point
    EndPoint   tree_sitter.Point
}

// Point represents a position in the source code
type tree_sitter.Point struct {
    Row    uint32
    Column uint32
}
```

## Complete Examples

### Example 1: Analyzing Go Functions

This example shows how to analyze functions in Go code:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/go-go-golems/oak/pkg/api"
)

type Function struct {
    Name       string
    Parameters string
    IsExported bool
    SourceFile string
    LineNumber int
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: program <file.go>")
        os.Exit(1)
    }

    filePath := os.Args[1]

    // Create a query builder for Go code
    query := api.NewQueryBuilder(
        api.WithLanguage("go"),
        api.WithQuery("functions", `
            (function_declaration
             name: (identifier) @functionName
             parameters: (parameter_list) @parameters
             body: (block)? @body)
        `),
        api.WithQuery("methods", `
            (method_declaration
             receiver: (parameter_list) @receiver
             name: (field_identifier) @methodName
             parameters: (parameter_list) @parameters
             body: (block)? @body)
        `),
    )

    // 1. Template-based output
    templateResult, err := query.RunWithTemplate(
        context.Background(),
        `
        # Functions in {{ .Language }} Files

        {{ range $file, $results := .ResultsByFile }}
        ## {{ $file }}

        {{ if index $results "functions" }}
        ### Functions
        {{ range (index $results "functions").Matches }}
        - func {{ (index . "functionName").Text }}{{ (index . "parameters").Text }}
        {{ end }}
        {{ end }}

        {{ if index $results "methods" }}
        ### Methods
        {{ range (index $results "methods").Matches }}
        - func {{ (index . "receiver").Text }} {{ (index . "methodName").Text }}{{ (index . "parameters").Text }}
        {{ end }}
        {{ end }}
        {{ end }}
        `,
        api.WithFiles([]string{filePath}),
    )
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        os.Exit(1)
    }

    fmt.Println("=== Template Output ===")
    fmt.Println(templateResult)

    // 2. Programmatic processing
    functionsResult, err := query.RunWithProcessor(
        context.Background(),
        func(results api.QueryResults) (any, error) {
            var functions []Function

            for fileName, fileResults := range results {
                // Process regular functions
                if funcResults, ok := fileResults["functions"]; ok {
                    for _, match := range funcResults.Matches {
                        fnName := match["functionName"].Text
                        params := match["parameters"].Text
                        functions = append(functions, Function{
                            Name:       fnName,
                            Parameters: params,
                            IsExported: isExported(fnName),
                            SourceFile: fileName,
                            LineNumber: int(match["functionName"].StartPoint.Row) + 1,
                        })
                    }
                }

                // Process methods
                if methodResults, ok := fileResults["methods"]; ok {
                    for _, match := range methodResults.Matches {
                        methName := match["methodName"].Text
                        params := match["parameters"].Text
                        functions = append(functions, Function{
                            Name:       methName,
                            Parameters: params,
                            IsExported: isExported(methName),
                            SourceFile: fileName,
                            LineNumber: int(match["methodName"].StartPoint.Row) + 1,
                        })
                    }
                }
            }

            return functions, nil
        },
        api.WithFiles([]string{filePath}),
    )
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        os.Exit(1)
    }

    // Type assertion for the result
    functions, ok := functionsResult.([]Function)
    if !ok {
        fmt.Println("Error: could not convert result to []Function")
        os.Exit(1)
    }

    fmt.Println("\n=== Programmatic Output ===")
    fmt.Printf("Found %d functions/methods:\n", len(functions))

    // Count exported vs non-exported functions
    exportedCount := 0
    for _, fn := range functions {
        if fn.IsExported {
            exportedCount++
        }
    }

    fmt.Printf("- Exported: %d\n", exportedCount)
    fmt.Printf("- Unexported: %d\n", len(functions) - exportedCount)

    // Print details
    fmt.Println("\nFunction Details:")
    for i, fn := range functions {
        exportedStr := "unexported"
        if fn.IsExported {
            exportedStr = "exported"
        }

        fmt.Printf("%d. %s (%s) at %s:%d\n",
            i+1,
            fn.Name + fn.Parameters,
            exportedStr,
            fn.SourceFile,
            fn.LineNumber,
        )
    }
}

// isExported checks if a function/method name is exported (starts with uppercase)
func isExported(name string) bool {
    if len(name) == 0 {
        return false
    }

    firstChar := name[0]
    return firstChar >= 'A' && firstChar <= 'Z'
}
```

### Example 2: Analyzing React Components in TypeScript

This example shows how to analyze React components in TypeScript code:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/go-go-golems/oak/pkg/api"
)

type Component struct {
    Name        string
    Type        string // 'function', 'arrow', 'class'
    Props       string
    IsExported  bool
    HasChildren bool
    SourceFile  string
    LineNumber  int
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: program <file.tsx or directory>")
        os.Exit(1)
    }

    path := os.Args[1]

    var runOption api.RunOption
    fileInfo, err := os.Stat(path)
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        os.Exit(1)
    }

    if fileInfo.IsDir() {
        // Use glob for directory
        runOption = api.WithGlob(filepath.Join(path, "**/*.{ts,tsx}"))
    } else {
        // Use specific file
        runOption = api.WithFiles([]string{path})
    }

    // Create a query builder for TypeScript
    query := api.NewQueryBuilder(
        api.WithLanguage("tsx"),
        api.WithQuery("functionComponents", `
            (function_declaration
             name: (identifier) @componentName
             parameters: (formal_parameters
               (required_parameter
                 pattern: (object_pattern)? @props)) @parameters
             body: (statement_block)? @body)
        `),
        api.WithQuery("arrowComponents", `
            (export_statement
             (lexical_declaration
               (variable_declarator
                 name: (identifier) @componentName
                 value: (arrow_function
                   parameters: (formal_parameters
                     (required_parameter
                       pattern: (object_pattern)? @props)) @parameters
                   body: (statement_block)? @body))))
        `),
        api.WithQuery("constArrowComponents", `
            (lexical_declaration
             (variable_declarator
               name: (identifier) @componentName
               value: (arrow_function
                 parameters: (formal_parameters
                   (required_parameter
                     pattern: (object_pattern)? @props)) @parameters
                 body: (statement_block)? @body)))
        `),
    )

    // Template-based output
    fmt.Println("=== React Components Analysis ===")
    templateResult, err := query.RunWithTemplate(
        context.Background(),
        `
        # React Components in TypeScript Files

        {{ range $file, $results := .ResultsByFile }}
        ## {{ $file }}

        {{ if index $results "functionComponents" }}
        ### Function Components
        {{ range (index $results "functionComponents").Matches }}
        - function {{ (index . "componentName").Text }}{{ (index . "parameters").Text }}
        {{ end }}
        {{ end }}

        {{ if index $results "arrowComponents" }}
        ### Exported Arrow Components
        {{ range (index $results "arrowComponents").Matches }}
        - export const {{ (index . "componentName").Text }} = {{ (index . "parameters").Text }} => {...}
        {{ end }}
        {{ end }}

        {{ if index $results "constArrowComponents" }}
        ### Const Arrow Components
        {{ range (index $results "constArrowComponents").Matches }}
        - const {{ (index . "componentName").Text }} = {{ (index . "parameters").Text }} => {...}
        {{ end }}
        {{ end }}
        {{ end }}
        `,
        runOption,
    )
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        os.Exit(1)
    }

    fmt.Println(templateResult)

    // Programmatic processing
    fmt.Println("\n=== Component Statistics ===")
    componentsResult, err := query.RunWithProcessor(
        context.Background(),
        func(results api.QueryResults) (any, error) {
            var components []Component

            for fileName, fileResults := range results {
                // Process function components
                if funcResults, ok := fileResults["functionComponents"]; ok {
                    for _, match := range funcResults.Matches {
                        compName := match["componentName"].Text
                        props := match["props"].Text
                        components = append(components, Component{
                            Name:        compName,
                            Type:        "function",
                            Props:       props,
                            IsExported:  isExported(compName),
                            HasChildren: hasChildren(props),
                            SourceFile:  fileName,
                            LineNumber:  int(match["componentName"].StartPoint.Row) + 1,
                        })
                    }
                }

                // Process exported arrow components
                if arrowResults, ok := fileResults["arrowComponents"]; ok {
                    for _, match := range arrowResults.Matches {
                        compName := match["componentName"].Text
                        props := match["props"].Text
                        components = append(components, Component{
                            Name:        compName,
                            Type:        "arrow",
                            Props:       props,
                            IsExported:  true, // These are always exported
                            HasChildren: hasChildren(props),
                            SourceFile:  fileName,
                            LineNumber:  int(match["componentName"].StartPoint.Row) + 1,
                        })
                    }
                }

                // Process const arrow components
                if constResults, ok := fileResults["constArrowComponents"]; ok {
                    for _, match := range constResults.Matches {
                        compName := match["componentName"].Text
                        props := match["props"].Text
                        components = append(components, Component{
                            Name:        compName,
                            Type:        "arrow",
                            Props:       props,
                            IsExported:  false, // These are not exported
                            HasChildren: hasChildren(props),
                            SourceFile:  fileName,
                            LineNumber:  int(match["componentName"].StartPoint.Row) + 1,
                        })
                    }
                }
            }

            return components, nil
        },
        runOption,
    )
    if err != nil {
        fmt.Printf("Error: %s\n", err)
        os.Exit(1)
    }

    // Type assertion for the result
    components, ok := componentsResult.([]Component)
    if !ok {
        fmt.Println("Error: could not convert result to []Component")
        os.Exit(1)
    }

    // Print statistics
    fmt.Printf("Found %d React components:\n", len(components))

    // Count by type
    funcCount := 0
    arrowCount := 0
    exportedCount := 0
    childrenCount := 0

    for _, comp := range components {
        if comp.Type == "function" {
            funcCount++
        } else if comp.Type == "arrow" {
            arrowCount++
        }

        if comp.IsExported {
            exportedCount++
        }

        if comp.HasChildren {
            childrenCount++
        }
    }

    fmt.Printf("- Function components: %d\n", funcCount)
    fmt.Printf("- Arrow function components: %d\n", arrowCount)
    fmt.Printf("- Exported components: %d\n", exportedCount)
    fmt.Printf("- Components with children: %d\n", childrenCount)

    // Print details of components with children
    if childrenCount > 0 {
        fmt.Println("\nComponents that accept children:")
        for _, comp := range components {
            if comp.HasChildren {
                exportedStr := ""
                if comp.IsExported {
                    exportedStr = "exported "
                }

                fmt.Printf("- %s (%s%s component) at %s:%d\n",
                    comp.Name,
                    exportedStr,
                    comp.Type,
                    comp.SourceFile,
                    comp.LineNumber,
                )
            }
        }
    }
}

// isExported checks if a component name is exported (starts with uppercase)
func isExported(name string) bool {
    if len(name) == 0 {
        return false
    }

    firstChar := name[0]
    return firstChar >= 'A' && firstChar <= 'Z'
}

// hasChildren checks if props include children
func hasChildren(props string) bool {
    return strings.Contains(props, "children") || strings.Contains(props, "props")
}
```

## Understanding the Implementation

### Overall Architecture

The Oak API is built around several key components:

1. **QueryBuilder**: The entry point that manages queries and execution options
2. **Query**: A simple struct representing a named tree-sitter query
3. **tree_sitter.SitterQuery**: The internal representation of a query in the tree-sitter subsystem
4. **tree_sitter.Result**: Contains matches for a specific query in a file
5. **tree_sitter.Match**: Maps capture names to captured syntax nodes
6. **tree_sitter.Capture**: Contains information about a captured syntax node

Here's a diagram of the main components and their relationships:

```
┌─────────────┐          ┌───────────────┐          ┌────────────────┐
│ QueryBuilder│ builds   │ tree_sitter   │ executes │ QueryResults   │
│             │─────────▶│ queries       │─────────▶│                │
└─────────────┘          └───────────────┘          └────────────────┘
       │                                                    │
       │                                                    │
       ▼                                                    ▼
┌─────────────┐                                    ┌────────────────┐
│ Template    │                                    │ Processor      │
│ processing  │                                    │ function       │
└─────────────┘                                    └────────────────┘
       │                                                    │
       ▼                                                    ▼
┌─────────────┐                                    ┌────────────────┐
│ Formatted   │                                    │ Typed Go       │
│ text output │                                    │ structures     │
└─────────────┘                                    └────────────────┘
```

### Query Execution Process

When you run a query, here's what happens internally:

1. The `QueryBuilder` resolves the list of files to process based on the provided options (files, glob patterns, directories).

2. It converts the queries to the internal `tree_sitter.SitterQuery` format.

3. It gets the appropriate tree-sitter language parser based on the language name.

4. For each file (in parallel):

   - It reads the file content
   - Creates a tree-sitter parser with the appropriate language
   - Parses the file content into a syntax tree
   - Executes each query against the syntax tree
   - Collects the results in a thread-safe manner

5. It returns the results as a `QueryResults` map, which maps filenames to query results.

6. If you're using template or programmatic processing, the results are passed to the template engine or processor function for further processing.

### Template Processing

Template processing uses Go's standard `text/template` package to format the query results. The template receives a data structure with the following fields:

- `Language`: The language used for parsing
- `ResultsByFile`: A map of filename to query results

The template can access these fields using the standard Go template syntax.

### Programmatic Processing

Programmatic processing uses a processor function that takes the raw query results and converts them into a typed Go structure. The processor function must have one of the following signatures:

```go
func(api.QueryResults) (any, error)
func(api.QueryResults) ([]any, error)
func(api.QueryResults) (map[string]any, error)
func(api.QueryResults) (string, error)
func(api.QueryResults) (int, error)
func(api.QueryResults) (bool, error)
```

The API uses type assertion to determine which function signature is being used.

## Advanced Topics

### Performance Considerations

#### Concurrency

The API processes files in parallel using a worker pool, with a default of 4 workers. You can adjust the number of workers using the `WithMaxWorkers` option.

```go
results, err := query.Run(
    context.Background(),
    api.WithGlob("src/**/*.go"),
    api.WithMaxWorkers(8), // Use 8 workers
)
```

For large codebases, increasing the number of workers can significantly improve performance, but be mindful of memory usage.

#### Query Complexity

Tree-sitter queries can be computationally expensive, especially when using complex patterns or predicates. Some tips for optimizing query performance:

1. Be specific with your captures. Capture only the nodes you need.
2. Avoid deeply nested patterns when possible.
3. Split complex queries into multiple simpler queries.

### Error Handling

The API provides detailed error messages for various failure scenarios:

- Missing language specification
- No queries specified
- No files found to process
- File read errors
- Parse errors
- Query execution errors
- Template parsing/execution errors
- Processor function errors

All errors are wrapped with descriptive messages using the `github.com/pkg/errors` package, so you can use `errors.Wrap` and `errors.Cause` to handle them appropriately.

### Language Support

The API supports all languages supported by the tree-sitter parsers included in Oak:

- Go
- JavaScript/TypeScript
- Python
- Ruby
- Rust
- C/C++
- Java
- PHP
- And more...

You can use the `pkg.LanguageNameToSitterLanguage` function to get the appropriate tree-sitter language parser for a given language name.

## Conclusion

The Oak Programmatic API provides a powerful, type-safe interface for working with tree-sitter queries in Go applications. By separating query building, execution, and result processing, it offers flexibility for a wide range of code analysis tasks.

Whether you're performing simple template-based formatting or complex programmatic processing, the API provides a clean, idiomatic Go interface that follows established patterns like functional options.

This guide covered the basics of using the API, detailed reference information, complete examples, and internal implementation details. For more information, see the API source code and examples in the Oak repository.
