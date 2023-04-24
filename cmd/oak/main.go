package main

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/oak/pkg"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

func main() {
	var queryFile string
	var templateFile string

	rootCmd := &cobra.Command{
		Use:   "oak",
		Short: "Oak is a wrapper around tree-sitter",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run an oak command against an input file",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// load queries
			f, err := os.Open(queryFile)
			cobra.CheckErr(err)

			loader := &pkg.OakCommandLoader{}
			cmds_, err := loader.LoadCommandFromYAML(f)
			cobra.CheckErr(err)
			if len(cmds_) != 1 {
				cobra.CheckErr(fmt.Errorf("expected exactly one command"))
			}
			oak := cmds_[0].(*pkg.OakCommand)

			for _, inputFile := range args {
				sourceCode, err := readFileOrStdin(inputFile)
				cobra.CheckErr(err)

				ctx := context.Background()
				tree, err := oak.Parse(ctx, sourceCode)
				cobra.CheckErr(err)

				results, err := oak.ExecuteQueries(tree.RootNode(), oak.Queries, sourceCode)
				cobra.CheckErr(err)

				// render template if provided
				var s string
				if templateFile != "" {
					s, err = oak.RenderWithTemplateFile(results, templateFile)
				} else {
					s, err = oak.Render(results)
				}
				cobra.CheckErr(err)

				fmt.Println(s)
			}
		},
	}

	runCmd.Flags().StringVarP(&queryFile, "query-file", "q", "", "SitterQuery file path")
	err := runCmd.MarkFlagRequired("query-file")
	cobra.CheckErr(err)

	runCmd.Flags().StringVarP(&templateFile, "template", "t", "", "Template file path")

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

				description := cmds.NewCommandDescription("query")

				oak := pkg.NewOakCommand(description,
					pkg.WithQueries(pkg.SitterQuery{
						Name:  queryName,
						Query: string(query),
					}),
					pkg.WithSitterLanguage(lang),
					pkg.WithTemplate(templateFile))

				sourceCode, err := readFileOrStdin(inputFile)
				cobra.CheckErr(err)

				ctx := context.Background()
				tree, err := oak.Parse(ctx, sourceCode)
				cobra.CheckErr(err)

				results, err := oak.ExecuteQueries(tree.RootNode(), oak.Queries, sourceCode)
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
	err = queryCmd.MarkFlagRequired("query-file")
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

			for _, inputFile := range args {

				var lang *sitter.Language
				if language != "" {
					lang, err = pkg.LanguageNameToSitterLanguage(language)
					cobra.CheckErr(err)
				} else {
					lang, err = pkg.FileNameToSitterLanguage(inputFile)
					cobra.CheckErr(err)
				}

				description := cmds.NewCommandDescription("parse")

				oak := pkg.NewOakCommand(
					description,
					pkg.WithSitterLanguage(lang),
					pkg.WithTemplate(templateFile))

				sourceCode, err := readFileOrStdin(inputFile)
				cobra.CheckErr(err)

				ctx := context.Background()
				tree, err := oak.Parse(ctx, sourceCode)
				cobra.CheckErr(err)

				oak.DumpTree(tree)
			}
		},
	}

	parseCmd.Flags().String("language", "", "Language name")

	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func readFileOrStdin(filename string) ([]byte, error) {
	if filename == "-" {
		return io.ReadAll(os.Stdin)
	}
	b, err := os.ReadFile(filename)
	return b, err
}
