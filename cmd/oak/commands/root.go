package commands

import (
	"embed"
	"fmt"
	"os"

	clay "github.com/go-go-golems/clay/pkg"
	clay_commandmeta "github.com/go-go-golems/clay/pkg/cmds/commandmeta"
	clay_repositories "github.com/go-go-golems/clay/pkg/cmds/repositories"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/types"
	cmds2 "github.com/go-go-golems/oak/pkg/cmds"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use:   "oak",
	Short: "oak runs tree-sitter queries against your source code",
}

func InitRootCmd(docFS embed.FS) (*help.HelpSystem, error) {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	if err != nil {
		return nil, err
	}

	helpSystem.SetupCobraRootCommand(RootCmd)

	err = clay.InitViper("oak", RootCmd)
	if err != nil {
		return nil, err
	}

	RootCmd.AddCommand(RunCommandCmd)
	return helpSystem, nil
}

func InitAllCommands(helpSystem *help.HelpSystem, queriesFS embed.FS) error {
	repositoryPaths := viper.GetStringSlice("repositories")

	defaultDirectory := "$HOME/.oak/queries"
	_, err := os.Stat(os.ExpandEnv(defaultDirectory))
	if err == nil {
		repositoryPaths = append(repositoryPaths, os.ExpandEnv(defaultDirectory))
	}

	loader := &cmds2.OakCommandLoader{}
	repositories_ := createRepositories(repositoryPaths, loader, queriesFS)

	allCommands, err := repositories.LoadRepositories(
		helpSystem,
		RootCmd,
		repositories_,
		cli.WithCobraShortHelpLayers(layers.DefaultSlug, cmds2.OakSlug),
	)
	if err != nil {
		return err
	}

	glazeCmd := &cobra.Command{
		Use:   "glaze",
		Short: "Run commands and output results as structured data",
	}
	RootCmd.AddCommand(glazeCmd)

	oakGlazedLoader := &cmds2.OakGlazedCommandLoader{}
	repositories_ = createRepositories(repositoryPaths, oakGlazedLoader, queriesFS)

	_, err = repositories.LoadRepositories(
		helpSystem,
		glazeCmd,
		repositories_,
		cli.WithCobraShortHelpLayers(layers.DefaultSlug, cmds2.OakSlug),
	)
	if err != nil {
		return err
	}

	// Create and add the unified command management group
	commandManagementCmd, err := clay_commandmeta.NewCommandManagementCommandGroup(
		allCommands,
		clay_commandmeta.WithListAddCommandToRowFunc(func(
			command glazed_cmds.Command,
			row types.Row,
			parsedLayers *layers.ParsedLayers,
		) ([]types.Row, error) {
			ret := []types.Row{row}
			switch c := command.(type) {
			case *cmds2.OakCommand:
				row.Set("language", c.Language)
				row.Set("queries", c.Queries)
				row.Set("type", "oak")
			case *cmds2.OakWriterCommand:
				row.Set("language", c.Language)
				row.Set("type", "oak-writer")
			case *alias.CommandAlias:
				row.Set("type", "alias")
				row.Set("aliasFor", c.AliasFor)
			default:
				if _, ok := row.Get("type"); !ok {
					row.Set("type", "unknown")
				}
			}
			return ret, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize command management commands: %w", err)
	}
	RootCmd.AddCommand(commandManagementCmd)

	// Create and add the repositories command group
	RootCmd.AddCommand(clay_repositories.NewRepositoriesGroupCommand())

	return nil
}

func createRepositories(repositoryPaths []string, loader loaders.CommandLoader, queriesFS embed.FS) []*repositories.Repository {
	directories := []repositories.Directory{
		{
			FS:               queriesFS,
			RootDirectory:    "queries",
			RootDocDirectory: "queries/doc",
			Name:             "oak",
			SourcePrefix:     "embed",
		}}

	for _, repositoryPath := range repositoryPaths {
		dir := os.ExpandEnv(repositoryPath)
		// check if dir exists
		if fi, err := os.Stat(dir); os.IsNotExist(err) || !fi.IsDir() {
			continue
		}
		directories = append(directories, repositories.Directory{
			FS:               os.DirFS(dir),
			RootDirectory:    ".",
			RootDocDirectory: "doc",
			Name:             dir,
			WatchDirectory:   dir,
			SourcePrefix:     "file",
		})
	}

	repositories_ := []*repositories.Repository{
		repositories.NewRepository(
			repositories.WithDirectories(directories...),
			repositories.WithCommandLoader(loader),
		),
	}
	return repositories_
}
