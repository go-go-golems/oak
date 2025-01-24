package cmds

import (
	"bytes"
	"context"
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/go-go-golems/oak/pkg"
	tree_sitter "github.com/go-go-golems/oak/pkg/tree-sitter"
	"io"
	"strings"
)

type OakWriterCommand struct {
	*OakCommand
}

var _ cmds.WriterCommand = (*OakWriterCommand)(nil)

func NewOakWriterCommand(d *cmds.CommandDescription, options ...OakCommandOption) *OakWriterCommand {
	cmd := OakWriterCommand{
		OakCommand: &OakCommand{
			CommandDescription: d,
		},
	}
	for _, option := range options {
		option(cmd.OakCommand)
	}
	return &cmd
}

func (oc *OakWriterCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
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
		err := oc.PrintQueries(w)
		if err != nil {
			return err
		}
		return nil
	}

	sources_ := s.Sources

	glob_ := ss.Glob

	if ss.Recurse && len(glob_) == 0 {
		// use standard globs for the language of the command
		glob_, err = pkg.GetLanguageGlobs(oc.Language)
		if err != nil {
			return err
		}
	}
	sources_, err = collectSources(sources_, glob_)
	if err != nil {
		return err
	}

	resultsByFile, err := oc.GetResultsByFile(ctx, sources_)
	if err != nil {
		return err
	}

	tmpl, err := templating.CreateTemplate("oak").Parse(oc.Template)
	if err != nil {
		return err
	}

	allResults := tree_sitter.QueryResults{}

	for _, fileResults := range resultsByFile {
		for k, v := range fileResults {
			result, ok := allResults[k]
			if !ok {
				// store copy of v in allResults
				allResults[k] = v.Clone()
				continue
			}
			result.Matches = append(result.Matches, v.Matches...)
		}
	}

	data := parsedLayers.GetDataMap()
	data["ResultsByFile"] = resultsByFile
	data["Results"] = allResults

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return err
	}

	s_ := buf.String()
	// trim left and right
	s_ = strings.TrimSpace(s_) + "\n"

	_, err = w.Write(([]byte)(s_))
	if err != nil {
		return err
	}

	return nil
}
