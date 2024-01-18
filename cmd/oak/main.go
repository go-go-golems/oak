package main

import (
	"context"
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	clay_cmds "github.com/go-go-golems/clay/pkg/cmds/locations"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/oak/pkg"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
)

var rootCmd = &cobra.Command{
	Use:   "oak",
	Short: "oak runs tree-sitter queries against your source code",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		// from the command line flag
		err := clay.InitLogger()
		cobra.CheckErr(err)
	},
}

func main() {
	// first, check if the are "run-command file.yaml",
	// because we need to load the file and then run the command itself.
	// we need to do this before cobra, because we don't know which flags to load yet
	if len(os.Args) >= 3 && os.Args[1] == "run" && os.Args[2] != "--help" {
		// load the command
		loader := &pkg.OakCommandLoader{}

		filePath, err := filepath.Abs(os.Args[2])
		if err != nil {
			fmt.Printf("Could not get absolute path: %v\n", err)
			os.Exit(1)
		}
		fs_, filePath, err := loaders.FileNameToFsFilePath(filePath)
		if err != nil {
			fmt.Printf("Could not get absolute path: %v\n", err)
			os.Exit(1)
		}
		cmds, err := loader.LoadCommands(
			fs_, filePath,
			[]glazed_cmds.CommandDescriptionOption{}, []alias.Option{},
		)
		if err != nil {
			fmt.Printf("Could not load command: %v\n", err)
			os.Exit(1)
		}
		if len(cmds) != 1 {
			fmt.Printf("Expected exactly one command, got %d", len(cmds))
		}

		writerCommand, ok := cmds[0].(glazed_cmds.WriterCommand)
		if !ok {
			fmt.Printf("Expected GlazeCommand, got %T", cmds[0])
			os.Exit(1)
		}

		cobraCommand, err := cli.BuildCobraCommandFromWriterCommand(writerCommand)
		if err != nil {
			fmt.Printf("Could not build cobra command: %v\n", err)
			os.Exit(1)
		}

		_, err = initRootCmd()
		cobra.CheckErr(err)

		rootCmd.AddCommand(cobraCommand)
		restArgs := os.Args[3:]
		os.Args = append([]string{os.Args[0], cobraCommand.Use}, restArgs...)
	} else {
		helpSystem, err := initRootCmd()
		cobra.CheckErr(err)

		err = initAllCommands(helpSystem)
		cobra.CheckErr(err)
	}

	registerLegacyCommands()

	err := rootCmd.Execute()
	cobra.CheckErr(err)
}

var runCommandCmd = &cobra.Command{
	Use:   "run-command",
	Short: "Run a command from a file",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		queryFile, err := filepath.Abs(args[0])
		if err != nil {
			cobra.CheckErr(err)
		}

		loader := &pkg.OakCommandLoader{}
		fs_, queryFile, err := loaders.FileNameToFsFilePath(queryFile)
		if err != nil {
			cobra.CheckErr(err)
		}
		cmds_, err := loader.LoadCommands(fs_, queryFile, []glazed_cmds.CommandDescriptionOption{}, []alias.Option{})
		cobra.CheckErr(err)
		if len(cmds_) != 1 {
			cobra.CheckErr(fmt.Errorf("expected exactly one command"))
		}
		oak, ok := cmds_[0].(*pkg.OakWriterCommand)
		if !ok {
			cobra.CheckErr(fmt.Errorf("expected OakWriterCommand"))
		}

		for _, inputFile := range args[1:] {
			sourceCode, err := readFileOrStdin(inputFile)
			cobra.CheckErr(err)

			ctx := context.Background()
			tree, err := oak.Parse(ctx, nil, sourceCode)
			cobra.CheckErr(err)

			lang, err := oak.GetLanguage()
			cobra.CheckErr(err)

			results, err := pkg.ExecuteQueries(lang, tree.RootNode(), oak.Queries, sourceCode)
			cobra.CheckErr(err)

			s, err := oak.Render(results)
			cobra.CheckErr(err)

			fmt.Println(s)
		}
	},
}

//go:embed doc/*
var docFS embed.FS

//go:embed queries/*
var queriesFS embed.FS

func initRootCmd() (*help.HelpSystem, error) {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	cobra.CheckErr(err)

	helpSystem.SetupCobraRootCommand(rootCmd)

	err = clay.InitViper("oak", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	rootCmd.AddCommand(runCommandCmd)
	return helpSystem, nil
}

func initAllCommands(helpSystem *help.HelpSystem) error {
	repositories := viper.GetStringSlice("repositories")

	defaultDirectory := "$HOME/.oak/queries"
	_, err := os.Stat(os.ExpandEnv(defaultDirectory))
	if err == nil {
		repositories = append(repositories, os.ExpandEnv(defaultDirectory))
	}

	locations := clay_cmds.CommandLocations{
		Embedded: []clay_cmds.EmbeddedCommandLocation{
			{
				FS:      queriesFS,
				Name:    "oak",
				Root:    "queries",
				DocRoot: "queries/doc",
			},
		},
		Repositories: repositories,
	}

	oakLoader := &pkg.OakCommandLoader{}
	commandLoader := clay_cmds.NewCommandLoader[glazed_cmds.Command](&locations)
	commands, aliases, err := commandLoader.LoadCommands(oakLoader, helpSystem)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	err = cli.AddCommandsToRootCommand(rootCmd, commands, aliases)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	glazeCmd := &cobra.Command{
		Use:   "glaze",
		Short: "Run commands and output results as structured data",
	}
	rootCmd.AddCommand(glazeCmd)

	oakGlazedLoader := &pkg.OakGlazedCommandLoader{}
	commands, aliases, err = commandLoader.LoadCommands(oakGlazedLoader, helpSystem)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	err = cli.AddCommandsToRootCommand(glazeCmd, commands, aliases)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	return nil
}

func registerLegacyCommands() {
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

				description := cmds.NewCommandDescription("query")

				oak := pkg.NewOakWriterCommand(description,
					pkg.WithQueries(pkg.SitterQuery{
						Name:  queryName,
						Query: string(query),
					}),
					pkg.WithSitterLanguage(lang),
					pkg.WithTemplate(templateFile))

				sourceCode, err := readFileOrStdin(inputFile)
				cobra.CheckErr(err)

				ctx := context.Background()
				tree, err := oak.Parse(ctx, nil, sourceCode)
				cobra.CheckErr(err)

				if lang == nil {
					lang, err = oak.GetLanguage()
					cobra.CheckErr(err)
				}

				results, err := pkg.ExecuteQueries(lang, tree.RootNode(), oak.Queries, sourceCode)
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

				oak := pkg.NewOakWriterCommand(
					description,
					pkg.WithSitterLanguage(lang),
					pkg.WithTemplate(templateFile))

				sourceCode, err := readFileOrStdin(inputFile)
				cobra.CheckErr(err)

				ctx := context.Background()
				tree, err := oak.Parse(ctx, nil, sourceCode)
				cobra.CheckErr(err)

				oak.DumpTree(tree)
			}
		},
	}

	parseCmd.Flags().String("language", "", "Language name")

	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(queryCmd)

}

func readFileOrStdin(filename string) ([]byte, error) {
	if filename == "-" {
		return io.ReadAll(os.Stdin)
	}
	b, err := os.ReadFile(filename)
	return b, err
}
