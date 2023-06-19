package pkg

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/go-go-golems/glazed/pkg/processor"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
	"io"
	"regexp"
	"strings"
	"text/template"
)

//go:embed "layers/oak.yaml"
var oakLayerYaml string

type OakParameterLayer struct {
	layers.ParameterLayerImpl
}

func NewOakParameterLayer(
	options ...layers.ParameterLayerOptions,
) (*OakParameterLayer, error) {
	layer, err := layers.NewParameterLayerFromYAML([]byte(oakLayerYaml), options...)
	if err != nil {
		return nil, err
	}
	return &OakParameterLayer{
		ParameterLayerImpl: *layer,
	}, nil
}

type SitterQuery struct {
	// Name of the resulting variable after parsing
	Name string
	// Query contains the tree-sitter query that will be applied to the source code
	Query string
	// rendered keeps track if the Query was rendered with RenderQueries.
	// This is an ugly way of doing things, but at least we'll signal at runtime
	// if the code tries to render a query multiple times.
	// See the NOTEs in RenderQueries.
	rendered bool
}

type OakCommandDescription struct {
	Language string        `yaml:"language,omitempty"`
	Queries  []SitterQuery `yaml:"queries"`
	Template string        `yaml:"template,omitempty"`

	Name   string                            `yaml:"name"`
	Short  string                            `yaml:"short"`
	Long   string                            `yaml:"long,omitempty"`
	Layout []*layout.Section                 `yaml:"layout,omitempty"`
	Flags  []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
	Layers []layers.ParameterLayer           `yaml:"layers,omitempty"`

	Parents []string `yaml:",omitempty"`
	Source  string   `yaml:",omitempty"`
}

type OakCommandLoader struct {
}

func (o *OakCommandLoader) LoadCommandFromYAML(
	s io.Reader,
	options ...cmds.CommandDescriptionOption,
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

	layers := append(ocd.Layers, oakLayer)

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithName(ocd.Name),
		cmds.WithShort(ocd.Short),
		cmds.WithLong(ocd.Long),
		cmds.WithFlags(ocd.Flags...),
		cmds.WithLayers(layers...),
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

	oakCommand := NewOakCommand(
		cmds.NewCommandDescription(ocd.Name, options_...),
		WithQueries(ocd.Queries...),
		WithTemplate(ocd.Template),
		WithLanguage(ocd.Language),
	)

	return []cmds.Command{oakCommand}, nil
}

func (o *OakCommandLoader) LoadCommandAliasFromYAML(
	s io.Reader,
	options ...alias.Option,
) ([]*alias.CommandAlias, error) {
	return loaders.LoadCommandAliasFromYAML(s, options...)
}

func (oc *OakCommand) Description() *cmds.CommandDescription {
	return oc.description

}

func NewOakCommand(d *cmds.CommandDescription, options ...OakCommandOption) *OakCommand {
	cmd := OakCommand{
		description: d,
	}
	for _, option := range options {
		option(&cmd)
	}
	return &cmd
}

type OakCommandOption func(*OakCommand)

func WithLanguage(lang string) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.Language = lang
	}
}

func WithSitterLanguage(lang *sitter.Language) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.SitterLanguage = lang
	}
}

func WithQueries(queries ...SitterQuery) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.Queries = append(cmd.Queries, queries...)
	}
}

func WithTemplate(template string) OakCommandOption {
	return func(cmd *OakCommand) {
		cmd.Template = template
	}
}

func (oc *OakCommand) Render(results QueryResults) (string, error) {
	tmpl, err := templating.CreateTemplate("oak").Parse(oc.Template)
	if err != nil {
		return "", err
	}

	return oc.RenderWithTemplate(results, tmpl)
}

func (oc *OakCommand) RenderWithTemplate(results QueryResults, tmpl *template.Template) (string, error) {
	data := map[string]interface{}{
		"Results": results,
	}

	for k, v := range results {
		data[k] = v
	}

	// TODO(manuel, 2023-04-23): add a way to pass in additional data

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (oc *OakCommand) RenderWithTemplateFile(results QueryResults, file string) (string, error) {
	tmpl, err := templating.CreateTemplate("oak").ParseFiles(file)
	if err != nil {
		return "", err
	}

	return oc.RenderWithTemplate(results, tmpl)
}

func (oc *OakCommand) ResultsToJSON(results QueryResults, f io.Writer) error {
	enc := json.NewEncoder(f)
	return enc.Encode(results)
}

func (oc *OakCommand) ResultsToYAML(results QueryResults, f io.Writer) error {
	enc := yaml.NewEncoder(f)
	return enc.Encode(results)
}

func (oc *OakCommand) GetLanguage() (*sitter.Language, error) {
	if oc.SitterLanguage == nil {
		lang, err := LanguageNameToSitterLanguage(oc.Language)
		if err != nil {
			return nil, err
		}
		oc.SitterLanguage = lang
	}
	return oc.SitterLanguage, nil
}

// Parse parses the given code using the language set in the command and returns
// the resulting tree.
func (oc *OakCommand) Parse(ctx context.Context, oldTree *sitter.Tree, code []byte) (*sitter.Tree, error) {
	lang, err := oc.GetLanguage()
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree, err := parser.ParseCtx(ctx, oldTree, code)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

// DumpTree prints the tree out to the console. This is lifted straight from example_test
// in the smacker/sitter repo.
//
// The output format is:
//
// source_file [0-29]
//
//	function_declaration [0-29]
//	    func [0-4]
//	    name: identifier [5-6]
//	    parameters: parameter_list [6-29]
//	        ( [6-7]
//	        parameter_declaration [7-18]
//	            name: identifier [7-8]
//	            , [8-9]
//	            name: identifier [10-11]
//	            , [11-12]
//	            name: identifier [13-14]
//	            type: type_identifier [15-18]
//
// The recursive cursor walker from the documentation didn't seem to work, at least on the hcl file.
func (oc *OakCommand) DumpTree(tree *sitter.Tree) {
	var visit2 func(n *sitter.Node, name string, depth int)
	visit2 = func(n *sitter.Node, name string, depth int) {
		printNode(n, depth, name)
		for i := 0; i < int(n.ChildCount()); i++ {
			visit2(n.Child(i), n.FieldNameForChild(i), depth+1)
		}

	}
	visit2(tree.RootNode(), "root", 0)
}

func printNode(n *sitter.Node, depth int, name string) {
	prefix := ""
	if name != "" {
		prefix = name + ": "
	}
	s := n.Type()
	// if s is whitespace, skip
	matched, err := regexp.MatchString(`^\s+$`, s)
	if err != nil {
		panic(err)
	}
	if matched {
		return
	}
	if len(s) <= 1 {
		fmt.Printf("%s%s%s\n", strings.Repeat("  ", depth), prefix, s)

	} else {
		fmt.Printf("%s%s%s [%d-%d]\n", strings.Repeat("  ", depth), prefix, s, n.StartByte(), n.EndByte())

	}
}

// RenderQueries replaces all the queries in the command with their "rendered" (using go templates)
// version.
//
// WARNING: This is destructive and should only be called once.
// NOTE(manuel, 2023-06-19) This is not a great API, but it will do for now.
func (oc *OakCommand) RenderQueries(ps map[string]interface{}) error {
	for name, _ := range oc.Queries {
		if oc.Queries[name].rendered {
			return errors.Errorf("query %s has already been rendered", name)
		}
		tmpl, err := templating.CreateTemplate("oak").Parse(oc.Queries[name].Query)
		if err != nil {
			return errors.Wrapf(err, "failed to parse query %s", name)
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, ps)
		if err != nil {
			return errors.Wrapf(err, "failed to render query %s", name)
		}

		query := buf.String()

		oc.Queries[name].Query = query
		oc.Queries[name].rendered = true
	}

	// remove queries that only consists of whitespace
	queries := []SitterQuery{}
	for _, query := range oc.Queries {
		if strings.TrimSpace(query.Query) != "" {
			queries = append(queries, query)
		}
	}

	oc.Queries = queries

	return nil
}

func (oc *OakCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp processor.Processor,
) error {
	err := oc.RenderQueries(ps)
	if err != nil {
		return err
	}

	printQueries, ok := ps["print-queries"]
	if ok && printQueries.(bool) {
		for _, q := range oc.Queries {
			v := map[string]interface{}{
				"query": q.Query,
				"name":  q.Name,
			}
			err := gp.ProcessInputObject(ctx, v)
			if err != nil {
				return err
			}
		}

		return nil
	}

	sources, ok := parsedLayers["oak"].Parameters["sources"]
	if !ok {
		return errors.New("no sources provided")
	}
	sources_, ok := cast.CastList2[string, interface{}](sources)
	if !ok {
		return errors.New("sources must be a list of strings")
	}

	resultsByFile, err := getResultsByFile(ctx, sources_, oc)
	if err != nil {
		return err
	}

	for _, fileResults := range resultsByFile {
		for _, result := range fileResults {
			for _, match := range result.Matches {
				for _, capture := range match {
					row := map[string]interface{}{
						"file":    fileResults,
						"query":   result.Name,
						"capture": capture.Name,
						"type":    capture.Type,
						"text":    capture.Text,
					}
					err = gp.ProcessInputObject(ctx, row)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// indentLines is a helper function that will prepend the given prefix in front of each line
// in s. This is useful when outputting things as a literal string in YAML.
func indentLines(s string, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}

func (oc *OakCommand) PrintQueries(w io.Writer) error {
	for _, query := range oc.Queries {
		_, err := fmt.Fprintf(
			w, "- name: %s\n  query: |\n%s",
			query.Name,
			indentLines(query.Query, "    "))
		if err != nil {
			return err
		}
	}

	return nil
}

func (oc *OakCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	w io.Writer,
) error {
	err := oc.RenderQueries(ps)
	if err != nil {
		return err
	}

	printQueries, ok := ps["print-queries"]
	if ok && printQueries.(bool) {
		err := oc.PrintQueries(w)
		if err != nil {
			return err
		}
		return nil
	}

	sources, ok := ps["sources"]
	if !ok {
		return errors.New("no sources provided")
	}
	sources_, ok := cast.CastList2[string, interface{}](sources)
	if !ok {
		return errors.New("sources must be a list of strings")
	}

	// TODO(manuel, 2023-04-23) Here we need to expand the query templates
	// probably also need to remove empty queries (?)

	resultsByFile, err := getResultsByFile(ctx, sources_, oc)
	if err != nil {
		return err
	}

	tmpl, err := templating.CreateTemplate("oak").Parse(oc.Template)
	if err != nil {
		return err
	}

	results := QueryResults{}

	for _, fileResults := range resultsByFile {
		for k, v := range fileResults {
			result, ok := results[k]
			if !ok {
				results[k] = v
				continue
			}
			result.Matches = append(result.Matches, v.Matches...)
		}
	}

	data := map[string]interface{}{
		"ResultsByFile": resultsByFile,
		"Results":       results,
	}

	for _, pd := range oc.description.Flags {
		v, ok := ps[pd.Name]
		if !ok {
			continue
		}
		data[pd.Name] = v
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return err
	}

	_, err = w.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}
