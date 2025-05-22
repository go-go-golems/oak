# Design Ideas for an Elegant Oak API

This document explores different approaches to create an elegant, builder-like programmatic API for Oak's tree-sitter query functionality, providing alternatives to the current YAML-based configuration.

## Current State

Currently, Oak uses YAML files to define tree-sitter queries and templates. Users must create YAML files like `functions.yaml` that specify:

1. Command metadata (name, description)
2. Flags for configuration
3. The programming language to parse
4. Tree-sitter queries with capture groups
5. Go templates to format the results

This approach works well for static queries but has limitations for programmatic use cases:
- Requires writing/reading external YAML files
- Limited composition and reuse of queries
- No easy way to build queries programmatically
- Difficult to integrate with other Go code

## Design Goals

An elegant programmatic API for Oak should:

1. Provide a fluent, builder-style interface
2. Allow composing and reusing query components
3. Support both string-based and programmatic query construction
4. Maintain compatibility with existing YAML files
5. Follow Go idioms and best practices
6. Make common operations simple and complex operations possible
7. Support programmatic processing of results with Go functions

## Design Approaches

### Design 1: Fluent Builder API with Method Chaining

This design uses method chaining to create a fluent API that builds up a query step by step.

```go
// Example program using Design 1
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
)

func main() {
    // Create a new query builder
    query := api.NewQueryBuilder().
        Language("tsx").
        AddQuery("functionDeclarations", `
            (
             (comment)* @comment .
             (function_declaration
              name: (identifier) @functionName
              parameters: (formal_parameters)? @parameters
              body: (statement_block)? @body)
            )
        `).
        AddQuery("arrowFunctionDeclarations", `
            (
             (comment)* @comment .
             (export_statement
              (lexical_declaration
                (variable_declarator
                  name: (identifier) @functionName
                  value: (arrow_function
                    parameters: (formal_parameters)? @parameters
                    body: (statement_block)? @body))))
            )
        `).
        WithTemplate(`
            {{ range $file, $results := .ResultsByFile }}
            File: {{ $file }}
            {{ range .functionDeclarations.Matches }}
            function {{.functionName.Text}} {{ .parameters.Text }}
            {{ end }}
            {{ range .arrowFunctionDeclarations.Matches }}
            export const {{ .functionName.Text }} = {{ .parameters.Text }}
            {{ end }}
            {{ end }}
        `)
    
    // Run the query on a file
    results, err := query.RunOnFile(context.Background(), "src/example.ts")
    if err != nil {
        panic(err)
    }
    
    // Print the results
    fmt.Println(results)
}
```

**Pros:**
- Clean, readable method chaining
- Familiar builder pattern for Go developers
- Good for simple, linear query construction
- Easy to understand for beginners

**Cons:**
- Not as flexible for complex compositions
- Can become unwieldy with many methods
- Limited ability to create reusable components

### Design 2: Functional Options Pattern

This design uses the functional options pattern, which is common in Go libraries.

```go
// Example program using Design 2
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
)

func main() {
    // Create a new query with functional options
    query := api.NewQuery(
        api.WithLanguage("tsx"),
        api.WithQuery("functionDeclarations", `
            (
             (comment)* @comment .
             (function_declaration
              name: (identifier) @functionName
              parameters: (formal_parameters)? @parameters
              body: (statement_block)? @body)
            )
        `),
        api.WithQuery("arrowFunctionDeclarations", `
            (
             (comment)* @comment .
             (export_statement
              (lexical_declaration
                (variable_declarator
                  name: (identifier) @functionName
                  value: (arrow_function
                    parameters: (formal_parameters)? @parameters
                    body: (statement_block)? @body))))
            )
        `),
        api.WithTemplate(`
            {{ range $file, $results := .ResultsByFile }}
            File: {{ $file }}
            {{ range .functionDeclarations.Matches }}
            function {{.functionName.Text}} {{ .parameters.Text }}
            {{ end }}
            {{ range .arrowFunctionDeclarations.Matches }}
            export const {{ .functionName.Text }} = {{ .parameters.Text }}
            {{ end }}
            {{ end }}
        `),
    )
    
    // Run the query on a file
    results, err := query.RunOnFile(context.Background(), "src/example.ts")
    if err != nil {
        panic(err)
    }
    
    // Print the results
    fmt.Println(results)
}
```

**Pros:**
- Very idiomatic Go pattern
- Allows for default values and optional parameters
- Good for creating reusable option sets
- Extensible without breaking changes

**Cons:**
- Less intuitive for complex query composition
- Can be verbose for simple queries
- Not as fluent as method chaining

### Design 3: Hybrid Approach with Composable Query Components

This design treats queries as composable components that can be combined and reused.

```go
// Example program using Design 3
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/typescript"
)

func main() {
    // Use predefined query components for TypeScript
    functionQuery := typescript.FunctionDeclaration().
        WithCapture("functionName").
        WithCapture("parameters").
        WithCapture("body").
        WithComments()
    
    arrowFunctionQuery := typescript.ArrowFunction().
        OnlyExported().
        WithCapture("functionName").
        WithCapture("parameters").
        WithCapture("body").
        WithComments()
    
    // Combine queries into a single query set
    query := api.NewQuery().
        Language("tsx").
        AddNamedQuery("functionDeclarations", functionQuery).
        AddNamedQuery("arrowFunctionDeclarations", arrowFunctionQuery).
        WithTemplate(`
            {{ range $file, $results := .ResultsByFile }}
            File: {{ $file }}
            {{ range .functionDeclarations.Matches }}
            function {{.functionName.Text}} {{ .parameters.Text }}
            {{ end }}
            {{ range .arrowFunctionDeclarations.Matches }}
            export const {{ .functionName.Text }} = {{ .parameters.Text }}
            {{ end }}
            {{ end }}
        `)
    
    // Run the query on a file
    results, err := query.RunOnFiles(context.Background(), []string{"src/example.ts"})
    if err != nil {
        panic(err)
    }
    
    // Print the results
    fmt.Println(results)
}
```

**Pros:**
- Highly composable and reusable components
- Language-specific helpers for common patterns
- Most powerful for complex queries
- Balances fluency with composition

**Cons:**
- More complex implementation
- Steeper learning curve
- Requires more upfront design for language-specific components

### Design 4: Query DSL with Chainable Conditions

This design creates a domain-specific language (DSL) for building tree-sitter queries programmatically.

```go
// Example program using Design 4
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/dsl"
)

func main() {
    // Create a query using the DSL
    query := api.NewQuery().
        Language("tsx").
        AddQuery("functionDeclarations", 
            dsl.Capture("comment", dsl.ZeroOrMore(dsl.Node("comment"))).
            Then().
            Capture("functionName", dsl.Field("name", dsl.Node("identifier"))).
            Inside(
                dsl.Node("function_declaration").
                WithOptionalField("parameters", "formal_parameters").
                WithOptionalField("body", "statement_block")
            )
        ).
        AddQuery("arrowFunctionDeclarations", 
            dsl.Capture("comment", dsl.ZeroOrMore(dsl.Node("comment"))).
            Then().
            Inside(
                dsl.Node("export_statement").
                Containing(
                    dsl.Node("lexical_declaration").
                    Containing(
                        dsl.Node("variable_declarator").
                        WithField("name", dsl.Capture("functionName", dsl.Node("identifier"))).
                        WithField("value", 
                            dsl.Node("arrow_function").
                            WithOptionalField("parameters", "formal_parameters").
                            WithOptionalField("body", "statement_block")
                        )
                    )
                )
            )
        ).
        WithTemplate(`
            {{ range $file, $results := .ResultsByFile }}
            File: {{ $file }}
            {{ range .functionDeclarations.Matches }}
            function {{.functionName.Text}} {{ .parameters.Text }}
            {{ end }}
            {{ range .arrowFunctionDeclarations.Matches }}
            export const {{ .functionName.Text }} = {{ .parameters.Text }}
            {{ end }}
            {{ end }}
        `)
    
    // Run the query on a file
    results, err := query.RunOnGlob(context.Background(), "src/**/*.ts")
    if err != nil {
        panic(err)
    }
    
    // Print the results
    fmt.Println(results)
}
```

**Pros:**
- Most expressive approach for complex queries
- Eliminates string-based query syntax errors at compile time
- Very powerful composition capabilities
- Provides type safety for query construction

**Cons:**
- Most complex implementation
- Steepest learning curve
- May be overkill for simple queries
- Most different from current approach

## Processing Results with Go Functions

While templates are great for formatting output, they have limitations when it comes to complex data processing. A powerful addition to the API would be the ability to process results using Go functions that can transform and aggregate data across files.

### Design 5: Result Processors and Collectors

This design adds the concept of "result processors" - functions that can be applied to query results to transform them into more useful data structures.

```go
// Example of using result processors
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/processors"
)

// Define a custom structure to collect function data
type FunctionInfo struct {
    Name       string
    Parameters string
    Body       string
    IsExported bool
    SourceFile string
    LineNumber int
}

// Group functions by file
type FunctionsByFile map[string][]FunctionInfo

func main() {
    // Create a query with a custom processor
    query := api.NewQuery(
        api.WithLanguage("tsx"),
        api.WithQuery("functionDeclarations", `
            (
             (comment)* @comment .
             (function_declaration
              name: (identifier) @functionName
              parameters: (formal_parameters)? @parameters
              body: (statement_block)? @body)
            )
        `),
        api.WithQuery("arrowFunctionDeclarations", `
            (
             (comment)* @comment .
             (export_statement
              (lexical_declaration
                (variable_declarator
                  name: (identifier) @functionName
                  value: (arrow_function
                    parameters: (formal_parameters)? @parameters
                    body: (statement_block)? @body))))
            )
        `),
        // Process results with a custom function instead of a template
        api.WithProcessor(func(ctx context.Context, results api.QueryResultsByFile) (FunctionsByFile, error) {
            functionsByFile := make(FunctionsByFile)
            
            for fileName, fileResults := range results {
                functions := []FunctionInfo{}
                
                // Process function declarations
                if funcResults, ok := fileResults["functionDeclarations"]; ok {
                    for _, match := range funcResults.Matches {
                        functions = append(functions, FunctionInfo{
                            Name:       match["functionName"].Text,
                            Parameters: match["parameters"].Text,
                            Body:       match["body"].Text,
                            IsExported: false,
                            SourceFile: fileName,
                            LineNumber: int(match["functionName"].StartPoint.Row) + 1,
                        })
                    }
                }
                
                // Process arrow function declarations
                if arrowResults, ok := fileResults["arrowFunctionDeclarations"]; ok {
                    for _, match := range arrowResults.Matches {
                        functions = append(functions, FunctionInfo{
                            Name:       match["functionName"].Text,
                            Parameters: match["parameters"].Text,
                            Body:       match["body"].Text,
                            IsExported: true,
                            SourceFile: fileName,
                            LineNumber: int(match["functionName"].StartPoint.Row) + 1,
                        })
                    }
                }
                
                functionsByFile[fileName] = functions
            }
            
            return functionsByFile, nil
        }),
    )
    
    // Run the query and get structured results
    functionsByFile, err := query.RunAndProcess(context.Background(), 
        api.WithGlob("src/**/*.ts"),
    )
    if err != nil {
        panic(err)
    }
    
    // Now we have a structured Go object that we can use for anything
    for file, functions := range functionsByFile {
        fmt.Printf("File: %s (%d functions)\n", file, len(functions))
        for _, fn := range functions {
            fmt.Printf("  - %s%s at line %d\n", 
                fn.Name, 
                fn.Parameters, 
                fn.LineNumber,
            )
        }
    }
    
    // We could also filter, transform, or save to a database
    exportedFunctions := []FunctionInfo{}
    for _, functions := range functionsByFile {
        for _, fn := range functions {
            if fn.IsExported {
                exportedFunctions = append(exportedFunctions, fn)
            }
        }
    }
    
    fmt.Printf("\nTotal exported functions: %d\n", len(exportedFunctions))
}
```

### Built-in Collectors

The API could also provide common collector functions for typical use cases:

```go
// Using built-in collectors
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/collectors"
)

func main() {
    // Create a query using built-in collectors
    query := api.NewQuery(
        api.WithLanguage("tsx"),
        api.WithQuery("functionDeclarations", `
            (
             (comment)* @comment .
             (function_declaration
              name: (identifier) @functionName
              parameters: (formal_parameters)? @parameters
              body: (statement_block)? @body)
            )
        `),
        // Use a built-in collector to gather all function names
        api.WithCollector(collectors.GatherTexts("functionName")),
    )
    
    // Run the query and get just function names
    functionNames, err := query.RunAndCollect(context.Background(), 
        api.WithGlob("src/**/*.ts"),
    )
    if err != nil {
        panic(err)
    }
    
    // Now we have a slice of all function names
    fmt.Printf("Found %d functions:\n", len(functionNames))
    for i, name := range functionNames {
        fmt.Printf("%d. %s\n", i+1, name)
    }
}
```

### Chaining Processors

Processors could be chained to create processing pipelines:

```go
// Chaining processors
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/processors"
)

func main() {
    // Create a query with a processing pipeline
    query := api.NewQuery(
        api.WithLanguage("go"),
        api.WithQuery("functions", `
            (function_declaration
             name: (identifier) @functionName
             body: (block) @body)
        `),
        // Apply a pipeline of processors
        api.WithProcessingPipeline(
            // First extract function information
            processors.ExtractFunctions(),
            // Then analyze complexity
            processors.AnalyzeCyclomaticComplexity(),
            // Then filter by complexity threshold
            processors.FilterByComplexity(10),
            // Finally, sort by complexity
            processors.SortByComplexity(),
        ),
    )
    
    // Run the query with the processing pipeline
    complexFunctions, err := query.RunAndProcess(context.Background(), 
        api.WithGlob("pkg/**/*.go"),
    )
    if err != nil {
        panic(err)
    }
    
    // Display functions that exceed complexity threshold
    fmt.Println("Functions with high cyclomatic complexity:")
    for _, fn := range complexFunctions {
        fmt.Printf("- %s (complexity: %d)\n", fn.Name, fn.Complexity)
    }
}
```

### Generic Result Types

For maximum flexibility, the API could use generics to specify the expected result type:

```go
// Using generics for result types
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/typescript"
)

// Custom result type
type ComponentInfo struct {
    Name      string
    Props     string
    FileName  string
    IsClass   bool
    UsesHooks bool
}

func main() {
    // Create a query with generic result type
    query := api.NewQueryWithResult[[]ComponentInfo](
        api.WithLanguage("tsx"),
        api.WithQuery("components", typescript.ReactComponents()),
        api.WithResultProcessor(func(results api.QueryResultsByFile) ([]ComponentInfo, error) {
            var components []ComponentInfo
            
            for fileName, fileResults := range results {
                if componentResults, ok := fileResults["components"]; ok {
                    for _, match := range componentResults.Matches {
                        // Extract component information
                        component := ComponentInfo{
                            Name:     match["componentName"].Text,
                            Props:    match["props"].Text,
                            FileName: fileName,
                            // Additional processing logic...
                        }
                        
                        // Check if component uses hooks
                        if body, ok := match["body"]; ok {
                            component.UsesHooks = strings.Contains(body.Text, "useState") || 
                                                 strings.Contains(body.Text, "useEffect")
                        }
                        
                        components = append(components, component)
                    }
                }
            }
            
            return components, nil
        }),
    )
    
    // Run the query and get strongly-typed results
    components, err := query.Run(context.Background(), 
        api.WithGlob("src/components/**/*.tsx"),
    )
    if err != nil {
        panic(err)
    }
    
    // Work with strongly-typed results
    for _, component := range components {
        fmt.Printf("Component: %s\n", component.Name)
        fmt.Printf("  File: %s\n", component.FileName)
        fmt.Printf("  Props: %s\n", component.Props)
        fmt.Printf("  Uses Hooks: %v\n\n", component.UsesHooks)
    }
}
```

### Streaming Results

For large codebases, the API could support streaming results to process them incrementally:

```go
// Streaming results
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
)

func main() {
    // Create a query that streams results
    query := api.NewQuery(
        api.WithLanguage("go"),
        api.WithQuery("functions", `
            (function_declaration
             name: (identifier) @functionName
             body: (block) @body)
        `),
    )
    
    // Create a result stream
    stream, err := query.StreamResults(context.Background(), 
        api.WithGlob("pkg/**/*.go"),
    )
    if err != nil {
        panic(err)
    }
    
    // Process results as they arrive
    count := 0
    for result := range stream.Results() {
        fileName := result.FileName
        for _, match := range result.Matches["functions"] {
            functionName := match["functionName"].Text
            fmt.Printf("Found function %s in %s\n", functionName, fileName)
            count++
        }
    }
    
    // Check for any errors during streaming
    if err := stream.Error(); err != nil {
        fmt.Printf("Error during streaming: %v\n", err)
    }
    
    fmt.Printf("Processed %d functions\n", count)
}
```

## Pros and Cons of Function-Based Processing

**Pros:**
- Full power of Go for data processing and transformation
- Can build complex data structures that aren't possible with templates
- Type safety and IDE completion for result structures
- Can implement complex aggregation and analysis
- Reusable processor components for common tasks
- Easy integration with other Go code and libraries

**Cons:**
- More verbose than templates for simple formatting
- Higher learning curve for basic usage
- More complex implementation in the library

## Design Considerations for Result Processing

1. **Type Safety**: Use Go generics to provide type safety for result processors
2. **Composability**: Allow processors to be composed and chained
3. **Reusability**: Provide common processors for typical tasks
4. **Performance**: Support streaming for large codebases
5. **Flexibility**: Allow mixing templates and processors when appropriate

## Example Use Cases

### Use Case 1: Finding TypeScript React Components

```go
// Using Design 3 (Hybrid Approach)
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/typescript"
    "github.com/go-go-golems/oak/pkg/api/jsx"
)

func main() {
    // Create a query for React components
    query := api.NewQuery().
        Language("tsx").
        AddNamedQuery("functionComponents", 
            typescript.FunctionDeclaration().
            WithCapture("componentName").
            WithCapture("props").
            WithCapture("returnValue").
            Where(jsx.ReturnsJSXElement())
        ).
        AddNamedQuery("arrowComponents", 
            typescript.ArrowFunction().
            OnlyExported().
            WithCapture("componentName").
            WithCapture("props").
            Where(jsx.ReturnsJSXElement())
        ).
        WithTemplate(`
            {{ range $file, $results := .ResultsByFile }}
            # React Components in {{ $file }}
            
            ## Function Components
            {{ range .functionComponents.Matches }}
            - {{ .componentName.Text }} (Props: {{ .props.Text }})
            {{ end }}
            
            ## Arrow Function Components
            {{ range .arrowComponents.Matches }}
            - {{ .componentName.Text }} (Props: {{ .props.Text }})
            {{ end }}
            {{ end }}
        `)
    
    // Run the query on a directory
    results, err := query.RunOnGlob(context.Background(), "src/components/**/*.tsx")
    if err != nil {
        panic(err)
    }
    
    // Output the results in Markdown format
    fmt.Println(results)
}
```

### Use Case 2: Finding SQL Injections in Go Code

```go
// Using Design 4 (Query DSL)
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/dsl"
    "github.com/go-go-golems/oak/pkg/api/golang"
)

func main() {
    // Create a query to find potential SQL injections
    query := api.NewQuery().
        Language("go").
        AddQuery("sqlQueries", 
            dsl.Capture("sqlCall", 
                dsl.Or(
                    golang.FunctionCall("db.Query"),
                    golang.FunctionCall("db.Exec"),
                    golang.FunctionCall("db.QueryRow")
                )
            ).
            Where(
                dsl.HasDescendant(
                    dsl.Capture("sqlString", 
                        dsl.And(
                            dsl.Node("interpreted_string_literal"),
                            dsl.Contains("+")
                        )
                    )
                )
            )
        ).
        WithTemplate(`
            # Potential SQL Injection Vulnerabilities
            
            {{ range $file, $results := .ResultsByFile }}
            ## {{ $file }}
            
            {{ range .sqlQueries.Matches }}
            - Line {{ .sqlCall.StartPoint.Row }}: {{ .sqlString.Text }}
            {{ end }}
            {{ end }}
        `)
    
    // Run the query on Go files
    results, err := query.RunOnGlob(context.Background(), "src/**/*.go")
    if err != nil {
        panic(err)
    }
    
    // Output the results
    fmt.Println(results)
}
```

### Use Case 3: Converting Existing YAML Queries

```go
// Using Design 2 (Functional Options)
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
)

func main() {
    // Load an existing query from a YAML file
    existingQuery, err := api.LoadFromYAML("cmd/oak/queries/typescript/functions.yaml")
    if err != nil {
        panic(err)
    }
    
    // Modify the loaded query
    query := api.NewQuery(
        api.WithExistingQuery(existingQuery),
        api.WithAdditionalQuery("classComponents", `
            (
             (class_declaration
              name: (identifier) @className
              body: (class_body
                (method_definition
                  name: (property_identifier) @methodName
                  (#eq? @methodName "render")
                  body: (statement_block) @renderBody)))
             (#match? @className "^[A-Z]")
            )
        `),
        api.WithFlag("include_classes", "bool", "Include React class components", false),
    )
    
    // Run the query on TypeScript files
    results, err := query.RunOnGlob(context.Background(), "src/**/*.tsx")
    if err != nil {
        panic(err)
    }
    
    // Output the results
    fmt.Println(results)
}
```

## API Design Comparison

| Feature | Design 1: Fluent Builder | Design 2: Functional Options | Design 3: Composable Components | Design 4: Query DSL |
|---------|-------------------------|---------------------------|------------------------------|------------------|
| Readability | High | Medium | High | Medium-High |
| Composability | Low | Medium | High | Very High |
| Reusability | Low | Medium | High | High |
| Type Safety | Medium | High | High | Very High |
| Learning Curve | Low | Low | Medium | High |
| Implementation Complexity | Low | Medium | High | Very High |
| Backward Compatibility | High | High | Medium | Medium |
| Extension Points | Few | Many | Many | Very Many |

## Recommendation

A hybrid approach combining elements of Design 2 (Functional Options), Design 3 (Composable Components), and Design 5 (Result Processors) would likely provide the best balance:

1. Use functional options for initializing queries and setting global properties
2. Provide composable query components for language-specific patterns
3. Allow both string-based queries and programmatic construction
4. Maintain YAML compatibility for existing users
5. Use result processors for complex data processing

This approach would allow for an elegant, flexible API that can grow over time while maintaining backward compatibility.

```go
// Recommended hybrid approach example
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg/api/typescript"
)

func main() {
    // Create reusable query components
    functionQuery := typescript.FunctionDeclaration().
        WithCapture("functionName").
        WithCapture("parameters").
        WithCapture("body").
        WithComments()
    
    // Create a query using functional options and components
    query := api.NewQuery(
        api.WithLanguage("tsx"),
        api.WithNamedComponent("functionDeclarations", functionQuery),
        api.WithQuery("arrowFunctionDeclarations", `
            (
             (comment)* @comment .
             (export_statement
              (lexical_declaration
                (variable_declarator
                  name: (identifier) @functionName
                  value: (arrow_function
                    parameters: (formal_parameters)? @parameters
                    body: (statement_block)? @body))))
            )
        `),
        api.WithTemplate(`
            {{ range $file, $results := .ResultsByFile }}
            File: {{ $file }}
            {{ range .functionDeclarations.Matches }}
            function {{.functionName.Text}} {{ .parameters.Text }}
            {{ end }}
            {{ range .arrowFunctionDeclarations.Matches }}
            export const {{ .functionName.Text }} = {{ .parameters.Text }}
            {{ end }}
            {{ end }}
        `),
        api.WithFlag("list", "bool", "List function names only", false),
    )
    
    // Run the query with options
    results, err := query.RunOn(
        context.Background(),
        api.WithFiles([]string{"src/example.ts"}),
        api.WithRecursive(true),
        api.WithGlob("**/*.ts"),
    )
    if err != nil {
        panic(err)
    }
    
    // Print the results
    fmt.Println(results)
}
```

## Implementation Considerations

1. **Core Types:**
   - `Query` - The main query object
   - `QueryComponent` - Composable query elements
   - `QueryOption` - Functional options for configuration
   - `Language` - Language-specific functionality

2. **Extension Points:**
   - Allow registering custom language handlers
   - Support for query transforms and middlewares
   - Hooks for pre/post processing of results

3. **Compatibility:**
   - Functions to convert between YAML and programmatic queries
   - Command-line tools that use the new API

4. **Performance:**
   - Cache compiled queries
   - Lazy loading of language definitions
   - Parallel execution for multi-file processing

## Next Steps

1. Implement a minimal version of the core API
2. Add language-specific components for TypeScript/JavaScript
3. Create converters for existing YAML queries
4. Build examples for common use cases
5. Gradually expand language support
6. Develop comprehensive documentation and tutorials

## Conclusion

An elegant builder-like API for Oak would significantly improve the developer experience and enable more complex, programmatic use cases. The hybrid approach combining functional options with composable components offers the best balance of simplicity, power, and compatibility with existing code. 