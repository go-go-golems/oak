package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type OakCommand struct {
	Language string        `yaml:"language,omitempty"`
	Queries  []SitterQuery `yaml:"queries"`
	Template string        `yaml:"template"`

	SitterLanguage *sitter.Language
	description    *cmds.CommandDescription
}

type Capture struct {
	// Name if the capture name from the query
	Name string
	// Text is the actual text that was captured
	Text string
	Type string

	// TODO(manuel, 2023-04-23): Add more information about the capture
	// for example: offset, line number, filename, query name, ...
}

type Match map[string]Capture

type Result struct {
	// Name is the name of the query
	Name string
	// TODO(manuel, 2023-04-23): Add filename
	// Matches are the matches for the query
	Matches []Match
}

type QueryResults map[string]*Result

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

func (oc *OakCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp cmds.Processor,
) error {
	sources, ok := parsedLayers["oak"].Parameters["sources"]
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
					err = gp.ProcessInputObject(row)
					if err != nil {
						return err
					}
				}
			}
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
	sources, ok := parsedLayers["oak"].Parameters["sources"]
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

func getResultsByFile(ctx context.Context, sources_ []string, oc *OakCommand) (
	map[string]QueryResults, error) {
	resultsByFile := map[string]QueryResults{}

	for _, fileName := range sources_ {
		source, err := os.ReadFile(fileName)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read file %s", fileName)
		}

		tree, err := oc.Parse(ctx, []byte(source))
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse file %s", fileName)
		}

		// TODO(manuel, 2023-04-23) Interpolate the queries
		results, err := oc.ExecuteQueries(tree.RootNode(), oc.Queries, source)
		if err != nil {
			return nil, errors.Wrapf(err, "could not execute queries for file %s", fileName)
		}

		resultsByFile[fileName] = results
	}

	return resultsByFile, nil
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

func (oc *OakCommand) ExecuteQueries(
	tree *sitter.Node,
	queries []SitterQuery,
	sourceCode []byte,
) (QueryResults, error) {
	if oc.SitterLanguage == nil {
		lang, err := LanguageNameToSitterLanguage(oc.Language)
		if err != nil {
			return nil, err
		}
		oc.SitterLanguage = lang
	}
	results := make(map[string]*Result)
	for _, query := range queries {
		matches := []Match{}

		// could parse queries up front and return an error if necessary
		q, err := sitter.NewQuery([]byte(query.Query), oc.SitterLanguage)
		if err != nil {
			switch err := err.(type) {
			case *sitter.QueryError:
				return nil, errors.Errorf("error parsing query %s: '%v'", query.Name, err.Error())
			}
			return nil, err
		}
		qc := sitter.NewQueryCursor()
		qc.Exec(q, tree)
		for {
			m, ok := qc.NextMatch()
			if !ok {
				break
			}
			if len(m.Captures) == 0 {
				continue
			}

			// for debugging purposes
			match := Match{}
			for _, c := range m.Captures {
				match[q.CaptureNameForId(c.Index)] = Capture{
					Name: q.CaptureNameForId(c.Index),
					Text: c.Node.Content(sourceCode),
				}
			}

			m = FilterPredicates(q, m, sourceCode)

			if len(m.Captures) == 0 {
				continue
			}

			match = Match{}
			for _, c := range m.Captures {
				match[q.CaptureNameForId(c.Index)] = Capture{
					Name: q.CaptureNameForId(c.Index),
					Text: c.Node.Content(sourceCode),
					Type: c.Node.Type(),
				}
			}
			matches = append(matches, match)
		}

		results[query.Name] = &Result{
			Name:    query.Name,
			Matches: matches,
		}
	}

	return results, nil
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

func (oc *OakCommand) Parse(ctx context.Context, code []byte) (*sitter.Tree, error) {
	if oc.SitterLanguage == nil {
		lang, err := LanguageNameToSitterLanguage(oc.Language)
		if err != nil {
			return nil, err
		}

		oc.SitterLanguage = lang
	}

	parser := sitter.NewParser()
	parser.SetLanguage(oc.SitterLanguage)
	tree, err := parser.ParseCtx(ctx, nil, code)
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
