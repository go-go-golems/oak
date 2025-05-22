package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/oak/pkg/api"
)

// Component represents a React component in TypeScript
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
		fmt.Println("Usage: typescript_analyzer <directory or file.tsx>")
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
