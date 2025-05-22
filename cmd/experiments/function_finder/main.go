package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/oak/pkg/api"
)

// Function represents a function found in the source code
type Function struct {
	Name       string
	Parameters string
	IsExported bool
	SourceFile string
	LineNumber int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: function_finder <file.go>")
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
	fmt.Println("\n=== Template Output ===")
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

	fmt.Println(templateResult)

	// 2. Programmatic processing
	fmt.Println("\n=== Programmatic Output ===")
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

	// Print statistics
	fmt.Printf("Found %d functions/methods:\n", len(functions))

	// Count exported vs non-exported functions
	exportedCount := 0
	for _, fn := range functions {
		if fn.IsExported {
			exportedCount++
		}
	}

	fmt.Printf("- Exported: %d\n", exportedCount)
	fmt.Printf("- Unexported: %d\n", len(functions)-exportedCount)

	// Print details
	fmt.Println("\nFunction Details:")
	for i, fn := range functions {
		exportedStr := "unexported"
		if fn.IsExported {
			exportedStr = "exported"
		}

		fmt.Printf("%d. %s (%s) at %s:%d\n",
			i+1,
			fn.Name+fn.Parameters,
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
