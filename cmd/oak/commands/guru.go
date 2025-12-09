package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/oak/pkg/guru"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// GuruCommand runs guru queries using symbol names instead of byte offsets
type GuruCommand struct {
	*cmds.CommandDescription
}

// GuruSettings maps command-line flags to Go fields
type GuruSettings struct {
	Mode   string `glazed.parameter:"mode"`
	Symbol string `glazed.parameter:"symbol"`
	File   string `glazed.parameter:"file"`
	JSON   bool   `glazed.parameter:"json"`
}

// Ensure interface compliance
var _ cmds.GlazeCommand = &GuruCommand{}

// RunIntoGlazeProcessor implements the GlazeCommand interface
func (c *GuruCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	// Parse settings from command line
	settings := &GuruSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
		return errors.Wrap(err, "failed to parse settings")
	}

	// Validate required fields
	if settings.Mode == "" {
		return fmt.Errorf("mode is required")
	}
	if settings.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if settings.File == "" {
		return fmt.Errorf("file is required")
	}

	// Find symbol position using Tree-sitter
	finder := guru.NewSymbolFinder()
	pos, err := finder.FindSymbol(ctx, settings.File, settings.Symbol)
	if err != nil {
		return errors.Wrapf(err, "failed to find symbol %s in %s", settings.Symbol, settings.File)
	}

	// Run guru query
	guruPosition := pos.ToGuruPosition()
	results, err := guru.RunGuruQuery(settings.Mode, guruPosition, settings.JSON)
	if err != nil {
		return errors.Wrapf(err, "failed to run guru query")
	}

	// Output structured data as rows
	for _, result := range results {
		row := types.NewRow(
			types.MRP("file", result.File),
			types.MRP("line", result.Line),
			types.MRP("column", result.Column),
			types.MRP("text", result.Text),
			types.MRP("kind", result.Kind),
			types.MRP("symbol", settings.Symbol),
			types.MRP("mode", settings.Mode),
			types.MRP("symbol_file", settings.File),
			types.MRP("symbol_type", pos.Type),
		)

		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add row")
		}
	}

	return nil
}

// NewGuruCommand creates a new guru command
func NewGuruCommand() (*GuruCommand, error) {
	// Create glazed layer for output formatting options
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create glazed layer")
	}

	// Create command settings layer for debugging features
	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command settings layer")
	}

	// Define command with parameters
	cmdDesc := cmds.NewCommandDescription(
		"guru",
		cmds.WithShort("Run guru queries using symbol names"),
		cmds.WithLong(`
Run guru queries using symbol names instead of byte offsets.

This command uses Tree-sitter to find symbol positions automatically, then
executes guru queries. No need to manually find byte offsets!

Modes:
  - referrers: Find all references to a symbol
  - callees: Find all callers of a function
  - implements: Find types implementing an interface
  - definition: Show symbol definition
  - describe: Describe symbol and its methods
  - freevars: Show free variables
  - peers: Show channel send/receive peers
  - what: Show basic symbol information
  - callstack: Show path from callgraph root to selected function

Examples:
  oak guru referrers ProcessData --file pkg/processor/data.go
  oak guru callees Query --file pkg/database/db.go --json
  oak guru implements Closer --file io/io.go
  oak guru describe ProcessData --file pkg/processor/data.go
		`),
		cmds.WithFlags(
			parameters.NewParameterDefinition(
				"mode",
				parameters.ParameterTypeString,
				parameters.WithRequired(true),
				parameters.WithHelp("Guru query mode: referrers, callees, implements, definition, describe, freevars, peers, what, callstack"),
			),
			parameters.NewParameterDefinition(
				"symbol",
				parameters.ParameterTypeString,
				parameters.WithRequired(true),
				parameters.WithHelp("Symbol name to query (function, type, method, variable, or const)"),
			),
			parameters.NewParameterDefinition(
				"file",
				parameters.ParameterTypeString,
				parameters.WithRequired(true),
				parameters.WithHelp("File containing the symbol"),
				parameters.WithShortFlag("f"),
			),
			parameters.NewParameterDefinition(
				"json",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Output guru results as JSON"),
			),
		),
		cmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &GuruCommand{
		CommandDescription: cmdDesc,
	}, nil
}

// GuruCmd is the cobra command wrapper
var GuruCmd *cobra.Command

func init() {
	guruCmd, err := NewGuruCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating guru command: %v\n", err)
		os.Exit(1)
	}

	// Convert to Cobra command
	cobraGuruCmd, err := cli.BuildCobraCommand(guruCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building guru command: %v\n", err)
		os.Exit(1)
	}

	GuruCmd = cobraGuruCmd
}
