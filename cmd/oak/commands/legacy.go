package commands

import (
	"context"
	"fmt"
	"os"

	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/oak/pkg"
	cmds2 "github.com/go-go-golems/oak/pkg/cmds"
	tree_sitter "github.com/go-go-golems/oak/pkg/tree-sitter"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func RegisterLegacyCommands(rootCmd *cobra.Command) {
	var queryFile string
	var templateFile string

	queryCmd := &cobra.Command{
		Use:   "query",
		Short: "SitterQuery a source code file with a plain sitter query",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			query, err := readFileOrStdin(queryFile)
			cobra.CheckErr(err)

			language, err := cmd.Flags().GetString("language")
			cobra.CheckErr(err)
			queryName, err := cmd.Flags().GetString("query-name")
			cobra.CheckErr(err)

			for _, inputFile := range args {
				var lang *sitter.Language
				if language != "" {
					lang, err = pkg.LanguageNameToSitterLanguage(language)
					cobra.CheckErr(err)
				} else {
					lang, err = pkg.FileNameToSitterLanguage(inputFile)
					cobra.CheckErr(err)
				}

				if queryName == "" {
					queryName = "main"
				}

				description := glazed_cmds.NewCommandDescription("query")

				oak := cmds2.NewOakWriterCommand(description,
					cmds2.WithQueries(tree_sitter.SitterQuery{
						Name:  queryName,
						Query: string(query),
					}),
					cmds2.WithSitterLanguage(lang),
					cmds2.WithTemplate(templateFile))

				sourceCode, err := readFileOrStdin(inputFile)
				cobra.CheckErr(err)

				ctx := context.Background()
				tree, err := oak.Parse(ctx, nil, sourceCode)
				cobra.CheckErr(err)

				if lang == nil {
					lang, err = oak.GetLanguage()
					cobra.CheckErr(err)
				}

				results, err := tree_sitter.ExecuteQueries(lang, tree.RootNode(), oak.Queries, sourceCode)
				cobra.CheckErr(err)

				// render template if provided
				if templateFile != "" {
					s, err := oak.RenderWithTemplateFile(results, templateFile)
					println(s)
					cobra.CheckErr(err)
				} else {
					matches := []map[string]string{}
					for _, result := range results {
						for _, match := range result.Matches {
							match_ := map[string]string{}
							for k, v := range match {
								// this really should be glazed output
								match_[k] = fmt.Sprintf("%s (%s)", v.Text, v.Type)
							}
							matches = append(matches, match_)
						}
					}
					err = yaml.NewEncoder(os.Stdout).Encode(matches)
					cobra.CheckErr(err)
				}
			}
		},
	}
	queryCmd.Flags().StringVarP(&queryFile, "query-file", "q", "", "SitterQuery file path")
	err := queryCmd.MarkFlagRequired("query-file")
	cobra.CheckErr(err)

	queryCmd.Flags().String("query-name", "", "SitterQuery name")
	queryCmd.Flags().String("language", "", "Language name")

	queryCmd.Flags().StringVarP(&templateFile, "template", "t", "", "Template file path")

	parseCmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse a source code file",
		Run: func(cmd *cobra.Command, args []string) {
			language, err := cmd.Flags().GetString("language")
			cobra.CheckErr(err)

			// Get dump format options
			dumpFormat, err := cmd.Flags().GetString("dump-format")
			cobra.CheckErr(err)
			showBytes, err := cmd.Flags().GetBool("show-bytes")
			cobra.CheckErr(err)
			showContent, err := cmd.Flags().GetBool("show-content")
			cobra.CheckErr(err)
			showAttributes, err := cmd.Flags().GetBool("show-attributes")
			cobra.CheckErr(err)
			skipWhitespace, err := cmd.Flags().GetBool("skip-whitespace")
			cobra.CheckErr(err)

			for _, inputFile := range args {
				var lang *sitter.Language
				if language != "" {
					lang, err = pkg.LanguageNameToSitterLanguage(language)
					cobra.CheckErr(err)
				} else {
					lang, err = pkg.FileNameToSitterLanguage(inputFile)
					cobra.CheckErr(err)
				}

				description := glazed_cmds.NewCommandDescription("parse")

				oak := cmds2.NewOakWriterCommand(
					description,
					cmds2.WithSitterLanguage(lang),
					cmds2.WithTemplate(templateFile))

				sourceCode, err := readFileOrStdin(inputFile)
				cobra.CheckErr(err)

				ctx := context.Background()
				tree, err := oak.Parse(ctx, nil, sourceCode)
				cobra.CheckErr(err)

				// Use the enhanced dumping with custom format if specified
				if dumpFormat != "" {
					var format tree_sitter.DumpFormat
					switch dumpFormat {
					case "text":
						format = tree_sitter.FormatText
					case "xml":
						format = tree_sitter.FormatXML
					case "json":
						format = tree_sitter.FormatJSON
					case "yaml":
						format = tree_sitter.FormatYAML
					default:
						format = tree_sitter.FormatText
					}

					options := tree_sitter.DumpOptions{
						ShowBytes:      showBytes,
						ShowContent:    showContent,
						ShowAttributes: showAttributes,
						SkipWhitespace: skipWhitespace,
					}

					err = oak.DumpTreeToWriter(tree, sourceCode, os.Stdout, format, options)
					cobra.CheckErr(err)
				} else {
					// Use the original DumpTree for backward compatibility
					oak.DumpTree(tree)
				}
			}
		},
	}

	parseCmd.Flags().String("language", "", "Language name")
	// Add dump format flags
	parseCmd.Flags().String("dump-format", "", "Output format for the tree dump (text, xml, json, yaml)")
	parseCmd.Flags().Bool("show-bytes", false, "Show byte offsets in the tree dump")
	parseCmd.Flags().Bool("show-content", true, "Show node content in the tree dump")
	parseCmd.Flags().Bool("show-attributes", true, "Show node attributes in the tree dump")
	parseCmd.Flags().Bool("skip-whitespace", true, "Skip whitespace-only nodes in the tree dump")

	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(queryCmd)
}
