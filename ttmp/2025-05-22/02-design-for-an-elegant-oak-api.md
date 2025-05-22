# Streamlined Design for an Elegant Oak API

This document outlines a focused, no-nonsense approach to creating a programmatic API for Oak's tree-sitter query functionality. The design emphasizes a clean separation between three core aspects:

1. **Query Building**: Constructing tree-sitter queries
2. **Template Processing**: Formatting results with templates
3. **Programmatic Processing**: Processing results with typed Go functions

## Core Principles

- Separate concerns: building queries vs processing results
- Type safety through Go generics
- Follow Go idioms (functional options pattern)
- Keep the API surface small but powerful
- Support common use cases without excessive complexity

## 1. Query Building API

The query builder provides a clean way to construct tree-sitter queries without dealing with YAML files.

```go
// Basic query builder example
package main

import (
    "context"
    "github.com/go-go-golems/oak/pkg/api"
)

func main() {
    // Create a new query builder
    query := api.NewQueryBuilder(
        api.WithLanguage("tsx"),
        api.WithQuery("functionDeclarations", `
            (function_declaration
             name: (identifier) @functionName
             parameters: (formal_parameters)? @parameters
             body: (statement_block)? @body)
        `),
        api.WithQuery("arrowFunctions", `
            (export_statement
             (lexical_declaration
               (variable_declarator
                 name: (identifier) @functionName
                 value: (arrow_function
                   parameters: (formal_parameters)? @parameters
                   body: (statement_block)? @body))))
        `),
    )

    // The query can now be used with either a template processor
    // or a programmatic processor
}
```

### Key Components

```go
// Core types for query building
type QueryBuilder struct {
    language string
    queries  []Query
}

type Query struct {
    Name  string
    Query string
}

// Constructor and options
func NewQueryBuilder(options ...QueryOption) *QueryBuilder

// QueryOption is a functional option for configuring the QueryBuilder
type QueryOption func(*QueryBuilder)

// Options
func WithLanguage(language string) QueryOption
func WithQuery(name, query string) QueryOption
func WithQueryFromFile(name, path string) QueryOption
func FromYAML(path string) QueryOption
```

## 2. Template Processing

For simpler use cases, template processing allows formatting results using Go templates, similar to the existing YAML approach.

```go
// Template processing example
package main

import (
    "context"
    "fmt"
    "github.com/go-go-golems/oak/pkg/api"
)

func main() {
    // Create a query
    query := api.NewQueryBuilder(
        api.WithLanguage("tsx"),
        api.WithQuery("functions", `
            (function_declaration
             name: (identifier) @functionName
             parameters: (formal_parameters)? @parameters)
        `),
    )
    
    // Process results with a template
    template := `
    {{ range $file, $results := .ResultsByFile }}
    File: {{ $file }}
    {{ range .functions.Matches }}
    - Function: {{ .functionName.Text }}{{ .parameters.Text }}
    {{ end }}
    {{ end }}
    `
    
    // Run the query and process results with the template
    results, err := query.RunWithTemplate(
        context.Background(),
        template,
        api.WithFiles([]string{"src/example.ts"}),
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println(results)
}
```

### Key Components

```go
// Template processing methods
func (q *QueryBuilder) RunWithTemplate(
    ctx context.Context,
    template string,
    options ...RunOption,
) (string, error)

func (q *QueryBuilder) RunWithTemplateFile(
    ctx context.Context,
    templatePath string,
    options ...RunOption,
) (string, error)
```

## 3. Programmatic Processing with Go Functions

For more complex use cases, programmatic processing allows transforming results into typed Go structures.

```go
// Programmatic processing example
package main

import (
    "context"
    "fmt"
    "github.com/go-go-golems/oak/pkg/api"
)

// Define a custom result type
type Function struct {
    Name       string
    Parameters string
    SourceFile string
    LineNumber int
}

func main() {
    // Create a query
    query := api.NewQueryBuilder(
        api.WithLanguage("tsx"),
        api.WithQuery("functions", `
            (function_declaration
             name: (identifier) @functionName
             parameters: (formal_parameters)? @parameters)
        `),
    )
    
    // Run the query with a processor function that returns a typed result
    functions, err := query.RunWithProcessor(
        context.Background(),
        func(results api.QueryResults) ([]Function, error) {
            var functions []Function
            
            for fileName, fileResults := range results {
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
        api.WithFiles([]string{"src/example.ts"}),
    )
    if err != nil {
        panic(err)
    }
    
    // Use the typed results
    for _, fn := range functions {
        fmt.Printf("Function %s%s in %s:%d\n", 
            fn.Name, 
            fn.Parameters, 
            fn.SourceFile,
            fn.LineNumber,
        )
    }
}
```

### Key Components

```go
// Programmatic processing with generics
func (q *QueryBuilder) RunWithProcessor[T any](
    ctx context.Context,
    processor func(api.QueryResults) (T, error),
    options ...RunOption,
) (T, error)

// QueryResults represents the raw query results by file
type QueryResults map[string]map[string]*Result

// Result represents the matches for a specific query
type Result struct {
    QueryName string
    Matches   []Match
}

// Match represents a single match with captured nodes
type Match map[string]Capture

// Capture represents a captured node in the tree
type Capture struct {
    Name       string
    Text       string
    Type       string
    StartByte  uint32
    EndByte    uint32
    StartPoint Point
    EndPoint   Point
}

// Point represents a position in the source code
type Point struct {
    Row    uint32
    Column uint32
}
```

## Common Run Options

Both template processing and programmatic processing share common options for specifying input files:

```go
// Run options
type RunOption func(*RunConfig)

func WithFiles(files []string) RunOption
func WithGlob(pattern string) RunOption
func WithRecursive(recursive bool) RunOption
func WithDirectory(dir string) RunOption
```

## Complete Example

Here's a complete example that shows how to use all three aspects of the API:

```go
package main

import (
    "context"
    "fmt"
    "github.com/go-go-golems/oak/pkg/api"
)

// Define a custom result type
type Function struct {
    Name       string
    Parameters string
    IsExported bool
    SourceFile string
    LineNumber int
}

func main() {
    // 1. Build the query
    query := api.NewQueryBuilder(
        api.WithLanguage("tsx"),
        api.WithQuery("functions", `
            (function_declaration
             name: (identifier) @functionName
             parameters: (formal_parameters)? @parameters)
        `),
        api.WithQuery("exportedFunctions", `
            (export_statement
             (lexical_declaration
               (variable_declarator
                 name: (identifier) @functionName
                 value: (arrow_function
                   parameters: (formal_parameters)? @parameters))))
        `),
    )
    
    // 2. Process with a template (for simple output)
    templateResult, err := query.RunWithTemplate(
        context.Background(),
        `
        # Functions in TypeScript Files
        
        {{ range $file, $results := .ResultsByFile }}
        ## {{ $file }}
        
        {{ range .functions.Matches }}
        - function {{ .functionName.Text }}{{ .parameters.Text }}
        {{ end }}
        
        {{ range .exportedFunctions.Matches }}
        - export const {{ .functionName.Text }} = {{ .parameters.Text }} => ...
        {{ end }}
        {{ end }}
        `,
        api.WithGlob("src/**/*.ts"),
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Template Output:")
    fmt.Println(templateResult)
    
    // 3. Process with a function (for structured data)
    functions, err := query.RunWithProcessor(
        context.Background(),
        func(results api.QueryResults) ([]Function, error) {
            var functions []Function
            
            for fileName, fileResults := range results {
                // Process regular functions
                if funcResults, ok := fileResults["functions"]; ok {
                    for _, match := range funcResults.Matches {
                        functions = append(functions, Function{
                            Name:       match["functionName"].Text,
                            Parameters: match["parameters"].Text,
                            IsExported: false,
                            SourceFile: fileName,
                            LineNumber: int(match["functionName"].StartPoint.Row) + 1,
                        })
                    }
                }
                
                // Process exported functions
                if exportedResults, ok := fileResults["exportedFunctions"]; ok {
                    for _, match := range exportedResults.Matches {
                        functions = append(functions, Function{
                            Name:       match["functionName"].Text,
                            Parameters: match["parameters"].Text,
                            IsExported: true,
                            SourceFile: fileName,
                            LineNumber: int(match["functionName"].StartPoint.Row) + 1,
                        })
                    }
                }
            }
            
            return functions, nil
        },
        api.WithGlob("src/**/*.ts"),
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println("\nProgrammatic Output:")
    fmt.Printf("Found %d functions:\n", len(functions))
    
    // Count exported vs non-exported functions
    exportedCount := 0
    for _, fn := range functions {
        if fn.IsExported {
            exportedCount++
        }
    }
    
    fmt.Printf("- Exported: %d\n", exportedCount)
    fmt.Printf("- Regular: %d\n", len(functions) - exportedCount)
}
```

## Implementation Considerations

1. **Separation of Concerns**: 
   - Keep query building and result processing separate
   - Allow each component to be used independently

2. **Type Safety**: 
   - Use generics for the result processor
   - Provide clear type definitions for query results

3. **Error Handling**:
   - Return meaningful errors at each step
   - Validate queries at build time when possible

4. **Performance**:
   - Compile queries once and reuse
   - Support parallel processing for multiple files

5. **Extensibility**:
   - Allow custom parsers and processors
   - Maintain compatibility with existing YAML files

## Next Steps

1. Implement the core query builder
2. Add template processing support
3. Implement programmatic processing with generics
4. Create examples for common use cases
5. Add support for loading/converting YAML files

## Conclusion

This streamlined API design provides a clean separation between query building and result processing, offering both template-based and programmatic options for handling results. By focusing on these core aspects and leveraging Go's type system, we can create an elegant, type-safe API that's both simple to use and powerful enough for complex code analysis tasks. 