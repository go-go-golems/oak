package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	cmds2 "github.com/go-go-golems/oak/pkg/cmds"
	tree_sitter "github.com/go-go-golems/oak/pkg/tree-sitter"
	"github.com/spf13/cobra"
)

var RunCommandCmd = &cobra.Command{
	Use:   "run-command",
	Short: "Run a command from a file",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		queryFile, err := filepath.Abs(args[0])
		if err != nil {
			cobra.CheckErr(err)
		}

		loader := &cmds2.OakCommandLoader{}
		fs_, queryFile, err := loaders.FileNameToFsFilePath(queryFile)
		if err != nil {
			cobra.CheckErr(err)
		}
		cmds_, err := loader.LoadCommands(fs_, queryFile, []glazed_cmds.CommandDescriptionOption{}, []alias.Option{})
		cobra.CheckErr(err)
		if len(cmds_) != 1 {
			cobra.CheckErr(fmt.Errorf("expected exactly one command"))
		}
		oak, ok := cmds_[0].(*cmds2.OakWriterCommand)
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

			results, err := tree_sitter.ExecuteQueries(lang, tree.RootNode(), oak.Queries, sourceCode)
			cobra.CheckErr(err)

			s, err := oak.Render(results)
			cobra.CheckErr(err)

			fmt.Println(s)
		}
	},
}

func readFileOrStdin(filename string) ([]byte, error) {
	if filename == "-" {
		return io.ReadAll(os.Stdin)
	}
	b, err := os.ReadFile(filename)
	return b, err
}
