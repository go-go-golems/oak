# Oak API

This package provides a clean, programmatic API for working with tree-sitter queries in Oak. The API is designed around three core aspects:

1. **Query Building**: Constructing tree-sitter queries
2. **Template Processing**: Formatting results with templates
3. **Programmatic Processing**: Processing results with typed Go functions

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/go-go-golems/oak/pkg/api"
)

func main() {
    // Create a query builder for Go code
    query := api.NewQueryBuilder(
        api.WithLanguage("go"),
        api.WithQuery("functions", `
            (function_declaration
             name: (identifier) @functionName
             parameters: (parameter_list) @parameters
             body: (block)? @body)
        `),
    )

    // Run the query and process results with a template
    templateResult, err := query.RunWithTemplate(
        context.Background(),
        `
        # Functions in {{ .Language }} Files
        
        {{ range $file, $results := .ResultsByFile }}
        ## {{ $file }}
        
        {{ range (index $results "functions").Matches }}
        - func {{ index . "functionName" "Text" }}{{ index . "parameters" "Text" }}
        {{ end }}
        {{ end }}
        `,
        api.WithFiles([]string{"main.go"}),
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println(templateResult)
}
```

## Core Concepts

### Query Building

The `QueryBuilder` struct provides methods for constructing tree-sitter queries:

```go
query := api.NewQueryBuilder(
    api.WithLanguage("go"),
    api.WithQuery("functions", "(function_declaration...)"),
    api.WithQueryFromFile("methods", "queries/methods.txt"),
    api.FromYAML("queries/spec.yaml"),
)
```

### Query Execution

Queries can be executed on files using various options:

```go
results, err := query.Run(
    context.Background(),
    api.WithFiles([]string{"main.go"}),
    api.WithGlob("src/**/*.go"),
    api.WithDirectory("pkg", api.WithRecursive(true)),
)
```

### Template Processing

For simple use cases, template processing allows formatting results using Go templates:

```go
result, err := query.RunWithTemplate(
    context.Background(),
    templateString,
    api.WithFiles([]string{"main.go"}),
)
```

### Programmatic Processing

For more complex use cases, programmatic processing allows transforming results into typed Go structures:

```go
type Function struct {
    Name       string
    Parameters string
    SourceFile string
    LineNumber int
}

functionsResult, err := query.RunWithProcessor(
    context.Background(),
    func(results api.QueryResults) (any, error) {
        var functions []Function
        // Process results into functions slice
        return functions, nil
    },
    api.WithFiles([]string{"main.go"}),
)

functions, ok := functionsResult.([]Function)
```

## Examples

See the `cmd/experiments` directory for example applications using this API.