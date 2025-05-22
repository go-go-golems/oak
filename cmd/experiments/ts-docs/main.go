package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/oak/pkg/api"
	"github.com/go-go-golems/oak/pkg/tree-sitter"
	"github.com/spf13/cobra"
)

type FunctionInfo struct {
	Name       string
	Docstring  string
	Params     []ParameterInfo
	ReturnType string
	SourceFile string
	LineNumber int
	IsExported bool
}

type ParameterInfo struct {
	Name string
	Type string
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "ts-docs [file/directory]",
		Short: "Generate API documentation for TypeScript/JavaScript files",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]
			var runOption api.RunOption

			fileInfo, err := os.Stat(path)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				os.Exit(1)
			}

			if fileInfo.IsDir() {
				// Use glob for directory
				runOption = api.WithGlob(filepath.Join(path, "**/*.{js,ts,jsx,tsx}"))
			} else {
				// Use specific file
				runOption = api.WithFiles([]string{path})
			}

			// Create a query builder for TypeScript/JavaScript
			query := api.NewQueryBuilder(
				api.WithLanguage("typescript"), // Works for both TS and JS
				api.WithQuery("functionDeclarations", `
					(function_declaration
					 name: (identifier) @name
					 parameters: (formal_parameters) @parameters
					 return_type: (_)? @returnType
					) @function
				`),
				api.WithQuery("arrowFunctions", `
					(lexical_declaration
					 (variable_declarator
					  name: (identifier) @name
					  value: (arrow_function
					   parameters: (formal_parameters) @parameters
					   return_type: (_)? @returnType
					  )) @function)
				`),
				// Note: We'll filter out exported arrow functions in the processor

				api.WithQuery("exportedArrowFunctions", `
					(export_statement
					 (lexical_declaration
					  (variable_declarator
					   name: (identifier) @name
					   value: (arrow_function
					    parameters: (formal_parameters) @parameters
					    return_type: (_)? @returnType
					   ))) @function)
				`),
				api.WithQuery("methodDefinitions", `
					(method_definition
					 name: (_) @name
					 parameters: (formal_parameters) @parameters
					 return_type: (_)? @returnType
					) @function
				`),
				api.WithQuery("comments", `
					(comment) @comment
				`),
			)

			// Process the results
			functionsResult, err := query.RunWithProcessor(
				context.Background(),
				func(results api.QueryResults) (any, error) {
					var allFunctions []FunctionInfo
					fileComments := make(map[string][]tree_sitter.Capture)

					// First, collect all comments to match with functions
					for fileName, fileResults := range results {
						if commentResults, ok := fileResults["comments"]; ok {
							for _, match := range commentResults.Matches {
								fileComments[fileName] = append(fileComments[fileName], match["comment"])
							}
						}
					}

					// Helper function to extract parameter info
					extractParams := func(paramsText string) []ParameterInfo {
						var params []ParameterInfo

						// Remove parentheses and split by comma
						cleanParams := strings.Trim(paramsText, "()")
						if cleanParams == "" {
							return params
						}

						// Handle object types with nested commas
						// This is a simplified approach - a proper parser would be better
						var paramItems []string
						braceLevel := 0
						currentParam := ""

						for i := 0; i < len(cleanParams); i++ {
							ch := cleanParams[i]
							switch ch {
							case '{', '(':
								braceLevel++
								currentParam += string(ch)
							case '}', ')':
								braceLevel--
								currentParam += string(ch)
							case ',':
								if braceLevel > 0 {
									currentParam += string(ch)
								} else {
									paramItems = append(paramItems, currentParam)
									currentParam = ""
								}
							default:
								currentParam += string(ch)
							}
						}

						// Add the last parameter
						if currentParam != "" {
							paramItems = append(paramItems, currentParam)
						}

						for _, param := range paramItems {
							param = strings.TrimSpace(param)

							// Split on the first colon to separate name and type
							parts := strings.SplitN(param, ":", 2)

							pInfo := ParameterInfo{Name: parts[0]}
							if len(parts) > 1 {
								pInfo.Type = strings.TrimSpace(parts[1])
							}
							params = append(params, pInfo)
						}
						return params
					}

					// Helper to find the nearest comment above a function
					findDocComment := func(fileName string, fnStartRow uint32) string {
						var nearestComment tree_sitter.Capture
						nearestDistance := uint32(10) // Only look for comments within 10 lines

						for _, comment := range fileComments[fileName] {
							if comment.EndPoint.Row < fnStartRow &&
								fnStartRow-comment.EndPoint.Row <= nearestDistance {
								nearestDistance = fnStartRow - comment.EndPoint.Row
								nearestComment = comment
							}
						}

						if nearestDistance <= 3 { // Only use comments within 3 lines
							// Clean the comment text
							text := nearestComment.Text
							text = strings.TrimSpace(text)

							// Remove comment markers
							text = strings.TrimPrefix(text, "//")
							text = strings.TrimPrefix(text, "/*")
							text = strings.TrimSuffix(text, "*/")

							// Process JSDoc style comments
							lines := strings.Split(text, "\n")
							for i, line := range lines {
								lines[i] = strings.TrimSpace(line)
								lines[i] = strings.TrimPrefix(lines[i], "*")
								lines[i] = strings.TrimSpace(lines[i])
							}

							// Convert JSDoc @param and @returns tags to markdown
							text = strings.Join(lines, "\n")
							text = strings.TrimSpace(text)

							// Convert @param tags to markdown list items
							text = strings.ReplaceAll(text, "@param ", "- **")
							text = strings.ReplaceAll(text, "@returns ", "**Returns:** ")

							// Add closing bold tags for parameter names
							lines = strings.Split(text, "\n")
							for i, line := range lines {
								if strings.HasPrefix(line, "- **") {
									// Extract parameter name (first word after the prefix)
									parts := strings.SplitN(line[4:], " ", 2) // Skip the "- **" prefix
									if len(parts) == 2 {
										lines[i] = "- **" + parts[0] + ":** " + parts[1]
									}
								}
							}

							return strings.Join(lines, "\n")
						}
						return ""
					}

					// Create a map to track seen functions by location to avoid duplicates
					seenFunctions := make(map[string]bool)

					// Process all function types
					for fileName, fileResults := range results {
						// Process function declarations
						processFunctions := func(queryName string) {
							if funcResults, ok := fileResults[queryName]; ok {
								for _, match := range funcResults.Matches {
									fnName := match["name"].Text
									fnStartRow := match["function"].StartPoint.Row
									fnStartCol := match["function"].StartPoint.Column

									// Skip invalid function names
									if strings.TrimSpace(fnName) == "" {
										continue
									}

									// Create a unique ID for this function based on its location
									// to avoid duplicates from multiple queries
									functionID := fmt.Sprintf("%s:%d:%d", fileName, fnStartRow, fnStartCol)
									if _, exists := seenFunctions[functionID]; exists {
										// Skip this function as we've already processed it
										continue
									}
									seenFunctions[functionID] = true

									// Find docstring comment
									docstring := findDocComment(fileName, fnStartRow)

									// Extract return type if available
									returnType := ""
									if rtCapture, ok := match["returnType"]; ok && rtCapture.Text != "" {
										returnType = rtCapture.Text
										// Clean up return type (remove colon prefix if present)
										returnType = strings.TrimPrefix(returnType, ":")
										returnType = strings.TrimSpace(returnType)
									}

									// Extract parameters
									params := extractParams(match["parameters"].Text)

									allFunctions = append(allFunctions, FunctionInfo{
										Name:       fnName,
										Docstring:  docstring,
										Params:     params,
										ReturnType: returnType,
										SourceFile: fileName,
										LineNumber: int(fnStartRow) + 1,
										IsExported: isExported(fnName) || queryName == "exportedArrowFunctions",
									})
								}
							}
						}

						// Process all function types - order is important for correctly detecting exports
						processFunctions("exportedArrowFunctions") // Process exported functions first
						processFunctions("functionDeclarations")
						processFunctions("arrowFunctions")
						processFunctions("methodDefinitions")
					}

					return allFunctions, nil
				},
				runOption,
			)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				os.Exit(1)
			}

			// Type assertion for the result
			functions, ok := functionsResult.([]FunctionInfo)
			if !ok {
				fmt.Println("Error: could not convert result to []FunctionInfo")
				os.Exit(1)
			}

			// Final deduplication by location
			seenLocations := make(map[string]int)
			var uniqueFunctions []FunctionInfo
			for _, fn := range functions {
				// Create a location key from file + line number + name
				locationKey := fmt.Sprintf("%s:%d:%s", fn.SourceFile, fn.LineNumber, fn.Name)
				if _, exists := seenLocations[locationKey]; !exists {
					uniqueFunctions = append(uniqueFunctions, fn)
					seenLocations[locationKey] = len(uniqueFunctions) - 1
				}
			}

			// Generate markdown output
			fmt.Println(generateMarkdown(uniqueFunctions, path))
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func generateMarkdown(functions []FunctionInfo, srcPath string) string {
	var output strings.Builder

	// Get the title from the source path
	base := filepath.Base(srcPath)
	extension := filepath.Ext(srcPath)
	title := base
	if extension != "" {
		title = strings.TrimSuffix(base, extension)
	}

	// Write header
	output.WriteString(fmt.Sprintf("# %s API Reference\n\n", title))

	// Group functions by file
	fileMap := make(map[string][]FunctionInfo)
	for _, fn := range functions {
		fileMap[fn.SourceFile] = append(fileMap[fn.SourceFile], fn)
	}

	// Generate table of contents
	output.WriteString("## Table of Contents\n\n")
	for file, fileFunctions := range fileMap {
		relPath, _ := filepath.Rel(srcPath, file)
		if relPath == "" {
			relPath = filepath.Base(file)
		}

		// Create section heading
		output.WriteString(fmt.Sprintf("- [%s](#%s)\n", relPath, strings.ReplaceAll(relPath, ".", "")))

		// Add function links
		for _, fn := range fileFunctions {
			anchor := strings.ToLower(fn.Name)
			anchor = strings.ReplaceAll(anchor, " ", "-")
			output.WriteString(fmt.Sprintf("  - [%s](#%s)\n", fn.Name, anchor))
		}
	}
	output.WriteString("\n")

	// Generate function documentation for each file
	for file, fileFunctions := range fileMap {
		relPath, _ := filepath.Rel(srcPath, file)
		if relPath == "" {
			relPath = filepath.Base(file)
		}

		// File header
		output.WriteString(fmt.Sprintf("## %s\n\n", relPath))

		// Document each function
		for _, fn := range fileFunctions {
			// Function heading
			output.WriteString(fmt.Sprintf("### %s\n\n", fn.Name))

			// Export status
			if fn.IsExported {
				output.WriteString("*Exported*\n\n")
			}

			// Description from docstring
			if fn.Docstring != "" {
				output.WriteString(fn.Docstring + "\n\n")
			}

			// Function signature
			output.WriteString("```typescript\n")
			signature := fn.Name + "("

			// Format parameters
			for i, param := range fn.Params {
				if i > 0 {
					signature += ", "
				}
				signature += param.Name
				if param.Type != "" {
					signature += ": " + param.Type
				}
			}
			signature += ")"

			// Add return type if available
			if fn.ReturnType != "" {
				signature += ": " + fn.ReturnType
			}

			output.WriteString(signature + "\n```\n\n")

			// Parameters section if we have any
			if len(fn.Params) > 0 {
				output.WriteString("**Parameters:**\n\n")
				for _, param := range fn.Params {
					typeInfo := ""
					if param.Type != "" {
						typeInfo = fmt.Sprintf(" - *%s*", param.Type)
					}
					output.WriteString(fmt.Sprintf("- `%s`%s\n", param.Name, typeInfo))
				}
				output.WriteString("\n")
			}

			// Return type section if available
			if fn.ReturnType != "" {
				output.WriteString(fmt.Sprintf("**Returns:** *%s*\n\n", fn.ReturnType))
			}

			// Source location
			output.WriteString(fmt.Sprintf("*Defined in [%s:%d]*\n\n", relPath, fn.LineNumber))
		}
	}

	return output.String()
}

// isExported checks if a function/method name is exported (starts with uppercase)
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}

	firstChar := name[0]
	return firstChar >= 'A' && firstChar <= 'Z'
}
