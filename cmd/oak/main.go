package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/oak/cmd/oak/commands"
	cmds2 "github.com/go-go-golems/oak/pkg/cmds"
	"github.com/spf13/cobra"
)

//go:embed doc/*
var docFS embed.FS

//go:embed queries/*
var queriesFS embed.FS

func main() {
	// first, check if the are "run-command file.yaml",
	// because we need to load the file and then run the command itself.
	// we need to do this before cobra, because we don't know which flags to load yet
	if len(os.Args) >= 3 && os.Args[1] == "run" && os.Args[2] != "--help" {
		// load the command
		loader := &cmds2.OakCommandLoader{}

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

		_, err = commands.InitRootCmd(docFS)
		cobra.CheckErr(err)

		commands.RootCmd.AddCommand(cobraCommand)
		restArgs := os.Args[3:]
		os.Args = append([]string{os.Args[0], cobraCommand.Use}, restArgs...)
	} else {
		helpSystem, err := commands.InitRootCmd(docFS)
		cobra.CheckErr(err)

		err = commands.InitAllCommands(helpSystem, queriesFS)
		cobra.CheckErr(err)
	}

	commands.RegisterLegacyCommands(commands.RootCmd)

	err := commands.RootCmd.Execute()
	cobra.CheckErr(err)
}
