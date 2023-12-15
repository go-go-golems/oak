# Oak: Advanced Code Analysis with Tree-Sitter and Go Templates

## Overview

Oak revolutionizes code analysis by integrating tree-sitter queries and Go template expansion. It allows users to perform sophisticated code inspections and generate insightful reports. Oak's unique approach combines the power of tree-sitter's parsing with the flexibility of Go templates, making it an invaluable tool for developers.

## Tree-Sitter Queries

Tree-sitter is a modern parser generator tool designed for code comprehension. It provides high-performance parsing of source code in various programming languages. In Oak, tree-sitter queries are used to specify the patterns and elements you want to extract from your codebase. The following examples are specifically tailored for the Go language:

### Example Queries for Go Language

1. **Function Declarations**:
   ```yaml
   (function_declaration
      name: (identifier) @functionName
      parameters: (parameter_list) @params
      body: (block) @body)
   ```

2. **Class/Struct Definitions**:
   ```yaml
   (type_declaration
      (type_spec
         name: (type_identifier) @typeName
         type: (struct_type) @typeBody))
   ```

These queries allow you to pinpoint specific syntax structures in the Go code, such as functions, classes, or specific expressions.

## Dynamic Query Templates

Oak enables users to define tree-sitter queries as dynamic Go templates. This allows for the interpolation of input flags, providing a way to customize queries based on user input.

### Flags in Queries

Flags are defined in the YAML configuration and can be used to filter and refine query results. For instance, a flag can be used to only return functions that match a specific name:

```yaml
- name: function_name
  type: string
  help: Filter functions by name
```

This flag can then be interpolated into a query template:

```yaml
{{ if .function_name }}(#eq? @functionName "{{.function_name}}"){{ end }}
```

## Result Structure and Template Expansion

After executing the tree-sitter queries, Oak structures the results, which are then passed to a Go template for final output generation.

### Result Matches Structure

Each result match in Oak is structured as a Go struct, containing detailed information about the code structure identified by the query. The struct typically includes:

- **Name**: The capture name from the query, e.g., `@functionName`.
- **Text**: The actual text content captured by the query.
- **Type**: The tree-sitter node type of the captured element.
- **Byte Range**: Start and end byte positions of the captured text.
- **Point Range**: Start and end line and column positions of the captured text.

This structured approach allows for precise and comprehensive analysis of query results.

### Final Template Expansion

The final Go template is used to format the output based on the matches. For example:

```yaml
template: |
  {{- range .functionInfo.Matches }}
    Function: {{ .functionName.Text }}
    Parameters: {{ .params.Text }}
    Body: {{ .body.Text }}
  {{- end }}
```

This template iterates over the matches from the `functionInfo` query, neatly formatting the function name, parameters, and body.

## Conclusion

Oak provides a powerful and flexible way to analyze and report on code structures. By leveraging tree-sitter queries and dynamic Go templates, it offers an advanced solution for developers looking to gain deeper insights into their codebases, especially in the Go language. Whether you are refactoring, documenting, or simply exploring your code, Oak serves as an essential tool in your development workflow.