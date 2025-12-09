package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	"gopkg.in/yaml.v3"
)

// GuruCommand runs guru queries using symbol names instead of byte offsets.
type GuruCommand struct {
	*cmds.CommandDescription
}

// GuruSettings maps CLI parameters to Go fields.
type GuruSettings struct {
	Mode          string   `glazed.parameter:"mode"`
	Symbols       []string `glazed.parameter:"symbol"`
	SymbolFiles   []string `glazed.parameter:"symbol-file"`
	SymbolsFile   string   `glazed.parameter:"symbols-file"`
	File          string   `glazed.parameter:"file"`
	JSON          bool     `glazed.parameter:"json"`
	Graph         string   `glazed.parameter:"graph"`
	GraphOutput   string   `glazed.parameter:"graph-output"`
	GraphMaxNodes int64    `glazed.parameter:"graph-max-nodes"`
}

type symbolRequest struct {
	Mode   string `yaml:"mode" json:"mode"`
	Symbol string `yaml:"symbol" json:"symbol"`
	File   string `yaml:"file" json:"file"`
}

type renderedGraph struct {
	Format        string
	Data          string
	File          string
	Nodes         int
	Edges         int
	SkippedReason string
}

type executionResult struct {
	Request  symbolRequest
	Position *guru.SymbolPosition
	Rows     []guru.GuruResult
	Graph    *renderedGraph
	Err      error
}

var (
	_ cmds.GlazeCommand = &GuruCommand{}
	_ cmds.BareCommand  = &GuruCommand{}
)

// RunIntoGlazeProcessor implements the structured output mode.
func (c *GuruCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	_, execResults, err := c.runBatch(ctx, parsedLayers)
	if err != nil && len(execResults) == 0 {
		return err
	}

	for idx, exec := range execResults {
		if exec.Err != nil {
			row := types.NewRow(
				types.MRP("symbol", exec.Request.Symbol),
				types.MRP("mode", exec.Request.Mode),
				types.MRP("symbol_file", exec.Request.File),
				types.MRP("error", exec.Err.Error()),
				types.MRP("batch_index", idx),
			)
			if exec.Graph != nil && exec.Graph.SkippedReason != "" {
				row.Set("graph_skipped_reason", exec.Graph.SkippedReason)
			}
			if err := gp.AddRow(ctx, row); err != nil {
				return errors.Wrap(err, "failed to add error row")
			}
			continue
		}

		graphBlob := ""
		graphFormat := strings.ToLower(exec.Graph.Format)
		graphNodes := 0
		graphEdges := 0
		graphFile := ""
		graphReason := ""
		if exec.Graph != nil {
			graphBlob = exec.Graph.Data
			graphNodes = exec.Graph.Nodes
			graphEdges = exec.Graph.Edges
			graphFile = exec.Graph.File
			graphReason = exec.Graph.SkippedReason
		}

		if len(exec.Rows) == 0 {
			row := types.NewRow(
				types.MRP("symbol", exec.Request.Symbol),
				types.MRP("mode", exec.Request.Mode),
				types.MRP("symbol_file", exec.Request.File),
				types.MRP("symbol_type", exec.Position.Type),
				types.MRP("summary", "no results"),
				types.MRP("batch_index", idx),
			)
			if exec.Graph != nil {
				row.Set("graph_format", graphFormat)
				row.Set("graph_data", graphBlob)
				row.Set("graph_nodes", graphNodes)
				row.Set("graph_edges", graphEdges)
				row.Set("graph_file", graphFile)
				if graphReason != "" {
					row.Set("graph_skipped_reason", graphReason)
				}
			}
			if err := gp.AddRow(ctx, row); err != nil {
				return errors.Wrap(err, "failed to add summary row")
			}
			continue
		}

		graphInjected := false
		for _, result := range exec.Rows {
			row := types.NewRow(
				types.MRP("file", result.File),
				types.MRP("line", result.Line),
				types.MRP("column", result.Column),
				types.MRP("text", result.Text),
				types.MRP("kind", result.Kind),
				types.MRP("symbol", exec.Request.Symbol),
				types.MRP("mode", exec.Request.Mode),
				types.MRP("symbol_file", exec.Request.File),
				types.MRP("symbol_type", exec.Position.Type),
			)
			if !graphInjected && exec.Graph != nil {
				row.Set("graph_format", graphFormat)
				row.Set("graph_data", graphBlob)
				row.Set("graph_nodes", graphNodes)
				row.Set("graph_edges", graphEdges)
				row.Set("graph_file", graphFile)
				if graphReason != "" {
					row.Set("graph_skipped_reason", graphReason)
				}
				graphInjected = true
			}
			if err := gp.AddRow(ctx, row); err != nil {
				return errors.Wrap(err, "failed to add row")
			}
		}
	}

	return err
}

// Run implements the BareCommand interface for human-friendly output.
func (c *GuruCommand) Run(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
) error {
	_, execResults, err := c.runBatch(ctx, parsedLayers)
	if err != nil && len(execResults) == 0 {
		return err
	}

	for _, exec := range execResults {
		fmt.Printf("Symbol %s (%s) in %s\n",
			exec.Request.Symbol, safeType(exec.Position), exec.Request.File)
		fmt.Printf("  Mode: %s\n", exec.Request.Mode)

		if exec.Err != nil {
			fmt.Printf("  Error: %s\n\n", exec.Err)
			continue
		}

		if len(exec.Rows) == 0 {
			fmt.Println("  No results returned.\n")
		} else {
			for i, result := range exec.Rows {
				fmt.Printf("  %d. %s:%d:%d -> %s\n",
					i+1, result.File, result.Line, result.Column, result.Text)
			}
			fmt.Println()
		}

		if exec.Graph != nil {
			if exec.Graph.SkippedReason != "" {
				fmt.Printf("  Graph skipped: %s\n\n", exec.Graph.SkippedReason)
				continue
			}
			fmt.Printf("  Graph format: %s (%d nodes / %d edges)\n",
				exec.Graph.Format, exec.Graph.Nodes, exec.Graph.Edges)
			if exec.Graph.File != "" {
				fmt.Printf("  Graph written to: %s\n", exec.Graph.File)
			} else if exec.Graph.Data != "" && len(exec.Graph.Data) < 1024 {
				fmt.Printf("  Graph preview:\n%s\n", exec.Graph.Data)
			}
			fmt.Println()
		}
	}

	return err
}

func (c *GuruCommand) runBatch(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
) (*GuruSettings, []executionResult, error) {
	settings := &GuruSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse settings")
	}

	requests, err := c.resolveRequests(settings)
	if err != nil {
		return nil, nil, err
	}

	graphFormat := strings.ToLower(strings.TrimSpace(settings.Graph))
	graphWriter := strings.TrimSpace(settings.GraphOutput)

	finder := guru.NewSymbolFinder()
	results := make([]executionResult, 0, len(requests))
	var errs []error

	for _, req := range requests {
		exec := executionResult{
			Request: req,
		}

		pos, findErr := finder.FindSymbol(ctx, req.File, req.Symbol)
		if findErr != nil {
			exec.Err = errors.Wrapf(findErr, "failed to locate %s in %s", req.Symbol, req.File)
			results = append(results, exec)
			errs = append(errs, exec.Err)
			continue
		}
		exec.Position = pos

		guruRows, guruErr := guru.RunGuruQuery(ctx, req.Mode, pos.ToGuruPosition(), settings.JSON)
		if guruErr != nil {
			exec.Err = errors.Wrapf(guruErr, "guru %s query failed for %s", req.Mode, req.Symbol)
			results = append(results, exec)
			errs = append(errs, exec.Err)
			continue
		}
		exec.Rows = guruRows

		if graphFormat != "" {
			graphData := guru.BuildGraph(req.Mode, req.Symbol, pos, guruRows)
			exec.Graph = renderGraph(graphData, graphFormat, settings.GraphMaxNodes)
			if exec.Graph != nil && exec.Graph.Data != "" && graphWriter != "" {
				outPath, writeErr := writeGraphToDisk(exec.Graph, graphWriter, req.Symbol, len(requests))
				if writeErr != nil {
					errs = append(errs, writeErr)
					if exec.Graph != nil {
						exec.Graph.File = ""
					}
				} else {
					exec.Graph.File = outPath
				}
			}
		}

		results = append(results, exec)
	}

	if len(errs) > 0 {
		return settings, results, errors.Errorf("batch completed with %d error(s)", len(errs))
	}

	return settings, results, nil
}

func (c *GuruCommand) resolveRequests(settings *GuruSettings) ([]symbolRequest, error) {
	var requests []symbolRequest

	// Manifest entries
	if settings.SymbolsFile != "" {
		manifestReqs, err := loadSymbolManifest(settings.SymbolsFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse symbols manifest")
		}
		requests = append(requests, manifestReqs...)
	}

	// CLI-provided symbols
	for idx, symbol := range settings.Symbols {
		if strings.TrimSpace(symbol) == "" {
			continue
		}
		entry := symbolRequest{Symbol: symbol}
		if idx < len(settings.SymbolFiles) {
			entry.File = strings.TrimSpace(settings.SymbolFiles[idx])
		}
		requests = append(requests, entry)
	}

	for idx := range requests {
		if requests[idx].Mode == "" {
			requests[idx].Mode = settings.Mode
		}
		if requests[idx].File == "" {
			requests[idx].File = settings.File
		}

		if requests[idx].Mode == "" {
			return nil, fmt.Errorf("mode required for symbol %s", requests[idx].Symbol)
		}
		if requests[idx].Symbol == "" {
			return nil, fmt.Errorf("symbol missing in request #%d", idx+1)
		}
		if requests[idx].File == "" {
			return nil, fmt.Errorf("file required for symbol %s", requests[idx].Symbol)
		}
	}

	if len(requests) == 0 {
		return nil, errors.New("no symbols provided via --symbol or --symbols-file")
	}

	return requests, nil
}

func loadSymbolManifest(path string) ([]symbolRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", path)
	}

	var list []symbolRequest
	if err := yaml.Unmarshal(data, &list); err == nil && len(list) > 0 {
		return list, nil
	}

	var wrapper struct {
		Symbols []symbolRequest `yaml:"symbols" json:"symbols"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err == nil && len(wrapper.Symbols) > 0 {
		return wrapper.Symbols, nil
	}

	var single symbolRequest
	if err := yaml.Unmarshal(data, &single); err == nil && single.Symbol != "" {
		return []symbolRequest{single}, nil
	}

	return nil, fmt.Errorf("manifest %s must contain a list or symbols field", path)
}

func renderGraph(graph *guru.Graph, format string, maxNodes int64) *renderedGraph {
	if graph == nil || format == "" {
		return nil
	}

	if maxNodes > 0 && int64(len(graph.Nodes)) > maxNodes {
		return &renderedGraph{
			Format:        format,
			SkippedReason: fmt.Sprintf("graph exceeds limit (%d nodes > %d max)", len(graph.Nodes), maxNodes),
		}
	}

	var data string
	switch format {
	case "dot":
		data = graph.ToDOT()
	case "mermaid":
		data = graph.ToMermaid()
	case "json":
		jsonData, err := graph.ToJSON()
		if err != nil {
			return &renderedGraph{
				Format:        format,
				SkippedReason: fmt.Sprintf("failed to encode graph: %v", err),
			}
		}
		data = jsonData
	default:
		return &renderedGraph{
			Format:        format,
			SkippedReason: fmt.Sprintf("unsupported graph format %s", format),
		}
	}

	return &renderedGraph{
		Format: format,
		Data:   data,
		Nodes:  len(graph.Nodes),
		Edges:  len(graph.Edges),
	}
}

func writeGraphToDisk(graph *renderedGraph, output string, symbol string, total int) (string, error) {
	if graph == nil || graph.Data == "" || output == "" {
		return "", nil
	}

	path := output
	info, err := os.Stat(output)
	if err == nil && info.IsDir() {
		path = filepath.Join(output, fmt.Sprintf("%s%s",
			guru.SlugifyIdentifier(symbol), graphExtension(graph.Format)))
	} else if err == nil && !info.IsDir() && total > 1 {
		base := filepath.Base(output)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		path = filepath.Join(filepath.Dir(output),
			fmt.Sprintf("%s-%s%s", name, guru.SlugifyIdentifier(symbol), ext))
	} else if err != nil && !os.IsNotExist(err) {
		return "", errors.Wrapf(err, "failed to inspect %s", output)
	}

	if err := os.WriteFile(path, []byte(graph.Data), 0o644); err != nil {
		return "", errors.Wrapf(err, "failed to write graph to %s", path)
	}

	return path, nil
}

func graphExtension(format string) string {
	switch format {
	case "dot":
		return ".dot"
	case "mermaid":
		return ".mmd"
	case "json":
		return ".json"
	default:
		return ".txt"
	}
}

func safeType(pos *guru.SymbolPosition) string {
	if pos == nil {
		return "unknown"
	}
	return pos.Type
}

// NewGuruCommand builds the CLI command description.
func NewGuruCommand() (*GuruCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers(
		settings.WithOutputParameterLayerOptions(
			layers.WithDefaults(map[string]interface{}{"output": "yaml"}),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create glazed layer")
	}

	commandSettingsLayer, err := cli.NewCommandSettingsLayer()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command settings layer")
	}

	cmdDesc := cmds.NewCommandDescription(
		"guru",
		cmds.WithShort("Run guru queries using symbol names"),
		cmds.WithLong(`
Run guru queries using symbol names instead of byte offsets.

This command uses Tree-sitter to find symbol positions automatically, then
executes guru queries. No need to manually find byte offsets!

It also supports batch execution (multiple --symbol flags or a manifest file)
and graph exports for referrers/callees/callstack modes.`),
		cmds.WithFlags(
			parameters.NewParameterDefinition(
				"mode",
				parameters.ParameterTypeString,
				parameters.WithHelp("Guru query mode: referrers, callees, implements, definition, describe, freevars, peers, what, callstack"),
			),
			parameters.NewParameterDefinition(
				"symbol",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Symbol name(s) to query (repeatable)"),
			),
			parameters.NewParameterDefinition(
				"symbol-file",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("File path(s) for each --symbol (falls back to --file)"),
			),
			parameters.NewParameterDefinition(
				"symbols-file",
				parameters.ParameterTypeString,
				parameters.WithHelp("YAML/JSON manifest describing multiple symbols"),
			),
			parameters.NewParameterDefinition(
				"file",
				parameters.ParameterTypeString,
				parameters.WithHelp("Default file containing the symbol (used if symbol entry omits file)"),
				parameters.WithShortFlag("f"),
			),
			parameters.NewParameterDefinition(
				"json",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Output guru results as JSON"),
			),
			parameters.NewParameterDefinition(
				"graph",
				parameters.ParameterTypeChoice,
				parameters.WithChoices("dot", "mermaid", "json"),
				parameters.WithHelp("Render graph output for supported modes"),
			),
			parameters.NewParameterDefinition(
				"graph-output",
				parameters.ParameterTypeString,
				parameters.WithHelp("Optional file or directory to write graph output"),
			),
			parameters.NewParameterDefinition(
				"graph-max-nodes",
				parameters.ParameterTypeInteger,
				parameters.WithHelp("Skip graph rendering beyond this node count"),
			),
		),
		cmds.WithLayersList(glazedLayer, commandSettingsLayer),
	)

	return &GuruCommand{
		CommandDescription: cmdDesc,
	}, nil
}

// GuruCmd is the cobra command wrapper.
var GuruCmd *cobra.Command

func init() {
	guruCmd, err := NewGuruCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating guru command: %v\n", err)
		os.Exit(1)
	}

	cobraGuruCmd, err := cli.BuildCobraCommand(guruCmd,
		cli.WithDualMode(true),
		cli.WithDefaultToGlaze(),
		cli.WithGlazeToggleFlag("no-glaze-output"),
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
