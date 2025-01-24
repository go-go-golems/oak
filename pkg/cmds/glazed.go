package cmds

import (
	"context"
	"io"
	"io/fs"
	"strings"

	"github.com/go-go-golems/oak/pkg"
	"gopkg.in/yaml.v3"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
)

type OakGlazeCommand struct {
	*OakCommand
}

var _ cmds.GlazeCommand = (*OakGlazeCommand)(nil)

type RunSettings struct {
	Sources []string `glazed.parameter:"sources"`
}

func (oc *OakGlazeCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	s := &RunSettings{}
	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return err
	}
	ss := &OakSettings{}
	err = parsedLayers.InitializeStruct(OakSlug, ss)
	if err != nil {
		return err
	}

	err = oc.RenderQueries(parsedLayers)
	if err != nil {
		return err
	}

	if ss.PrintQueries {
		for _, q := range oc.Queries {
			v := types.NewRow(
				types.MRP("query", q.Query),
				types.MRP("name", q.Name),
			)
			err := gp.AddRow(ctx, v)
			if err != nil {
				return err
			}
		}

		return nil
	}

	glob_ := ss.Glob
	if ss.Recurse && len(glob_) == 0 {
		// use standard globs for the language of the command
		glob_, err = pkg.GetLanguageGlobs(oc.Language)
		if err != nil {
			return err
		}
	}
	sources_, err := collectSources(s.Sources, glob_)
	if err != nil {
		return err
	}

	resultsByFile, err := oc.GetResultsByFile(ctx, sources_)
	if err != nil {
		return err
	}

	for fileName, fileResults := range resultsByFile {
		for _, result := range fileResults {
			for _, match := range result.Matches {
				for _, capture := range match {
					row := types.NewRow(
						types.MRP("file", fileName),
						types.MRP("query", result.QueryName),
						types.MRP("capture", capture.Name),

						types.MRP("startRow", capture.StartPoint.Row),
						types.MRP("startColumn", capture.StartPoint.Column),
						types.MRP("endRow", capture.EndPoint.Row),
						types.MRP("endColumn", capture.EndPoint.Column),

						types.MRP("startByte", capture.StartByte),
						types.MRP("endByte", capture.EndByte),

						types.MRP("type", capture.Type),
						types.MRP("text", capture.Text),
					)
					err = gp.AddRow(ctx, row)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func NewOakGlazedCommand(d *cmds.CommandDescription, options ...OakCommandOption) *OakGlazeCommand {
	cmd := OakGlazeCommand{
		OakCommand: &OakCommand{
			CommandDescription: d,
		},
	}
	for _, option := range options {
		option(cmd.OakCommand)
	}
	return &cmd
}

type OakGlazedCommandLoader struct{}

func (o *OakGlazedCommandLoader) IsFileSupported(f fs.FS, fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}

var _ loaders.CommandLoader = (*OakGlazedCommandLoader)(nil)

func (o *OakGlazedCommandLoader) LoadCommands(
	f fs.FS, entryName string,
	options []cmds.CommandDescriptionOption,
	aliasOptions []alias.Option,
) ([]cmds.Command, error) {
	r, err := f.Open(entryName)
	if err != nil {
		return nil, err
	}
	defer func(r fs.File) {
		_ = r.Close()
	}(r)

	return loaders.LoadCommandOrAliasFromReader(
		r,
		o.loadCommandFromReader,
		options,
		aliasOptions)
}

func (o *OakGlazedCommandLoader) loadCommandFromReader(
	s io.Reader,
	options []cmds.CommandDescriptionOption,
	_ []alias.Option,
) ([]cmds.Command, error) {
	ocd := &OakCommandDescription{}
	err := yaml.NewDecoder(s).Decode(ocd)
	if err != nil {
		return nil, err
	}

	oakLayer, err := NewOakParameterLayer()
	if err != nil {
		return nil, err
	}

	glazeLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	layers := append(ocd.Layers, glazeLayer, oakLayer)

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithName(ocd.Name),
		cmds.WithShort(ocd.Short),
		cmds.WithLong(ocd.Long),
		cmds.WithFlags(ocd.Flags...),
		cmds.WithLayersList(layers...),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"sources",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Files (or directories if recursing) to parse"),
				parameters.WithRequired(false),
			),
		),
		cmds.WithLayout(&layout.Layout{
			Sections: ocd.Layout,
		}),
	}
	options_ = append(options_, options...)

	oakCommand := NewOakGlazedCommand(
		cmds.NewCommandDescription(ocd.Name, options_...),
		WithQueries(ocd.Queries...),
		WithTemplate(ocd.Template),
		WithLanguage(ocd.Language),
	)

	return []cmds.Command{oakCommand}, nil
}

func (o *OakGlazedCommandLoader) LoadCommandAliasFromYAML(
	s io.Reader,
	options ...alias.Option,
) ([]*alias.CommandAlias, error) {
	return loaders.LoadCommandAliasFromYAML(s, options...)
}
