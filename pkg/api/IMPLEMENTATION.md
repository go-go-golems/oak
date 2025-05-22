# Oak API Implementation Guide

## Overview

The Oak API provides a clean, type-safe interface for working with tree-sitter queries. This document describes the key implementation details and design decisions.

## Core Components

### QueryBuilder

The `QueryBuilder` is the central type in the API. It holds:

- The language to parse files with
- A list of queries with names

It provides methods for:

- Building queries with functional options
- Running queries on files
- Processing results with templates or processor functions

### Query Execution

Query execution is performed in parallel with a configurable number of worker goroutines. The implementation:

1. Resolves the list of files to process (from globs, directories, etc.)
2. Creates tree-sitter queries from the query strings
3. Processes each file in parallel with a worker pool
4. Collects results in a thread-safe manner

### Template Processing

Template processing converts query results into text using Go's template engine. The implementation:

1. Creates a template data structure with language and results by file
2. Parses and executes the template
3. Returns the formatted output

Templates can access query results with the following structure:

```
.Language        - The language used for parsing
.ResultsByFile   - A map of filename to query results
```

For example, to loop through results by file:

```go
{{ range $file, $results := .ResultsByFile }}
  File: {{ $file }}
  {{ range (index $results "queryName").Matches }}
    {{ index . "captureName" "Text" }}
  {{ end }}
{{ end }}
```

### Programmatic Processing

Programmatic processing allows transforming results into typed Go structures using a processor function. The implementation:

1. Runs the queries to get raw results
2. Calls the processor function with the results
3. Returns the processed output

The processor function must match one of the supported signatures, such as:

```go
func(api.QueryResults) (any, error)
```

## Key Design Decisions

1. **Functional Options Pattern**: The API uses the functional options pattern for both query building and execution, making it flexible and easy to extend.

2. **Concurrency Model**: Query execution uses a worker pool with a configurable number of workers, allowing for efficient processing of large codebases.

3. **Error Handling**: Errors are wrapped with descriptive messages to help with debugging.

4. **Extensibility**: The API is designed to be extensible with new processors and query sources.

## Limitations and Future Work

1. **Generic Programming**: The current API doesn't use Go's generics for the processor function, limiting type safety. Future versions could add generics support for better type inference.

2. **Language Support**: The API relies on Oak's existing language support. New languages would need to be added to Oak's core.

3. **Query Caching**: Query compilation is not currently cached between runs. This could be an optimization for repeated queries.

4. **Incremental Parsing**: For large codebases, incremental parsing could improve performance for frequently changing files.

5. **Extensible Capture Processing**: Advanced capture processing could be added, such as automatic type inference or conversion.