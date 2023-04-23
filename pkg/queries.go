package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
	"io"
	"regexp"
	"strings"
	"text/template"
)

type OakCommand struct {
	Language string        `yaml:"language,omitempty"`
	Queries  []SitterQuery `yaml:"queries"`
	Template string        `yaml:"template"`

	SitterLanguage *sitter.Language
	description    *OakCommandDescription
}

type Capture struct {
	// Name if the capture name from the query
	Name string
	// Text is the actual text that was captured
	Text string
	Type string
}

type Match map[string]Capture

type Result struct {
	// Name is the name of the query
	Name string
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

func (cmd *OakCommand) Run(ctx context.Context,
	parsedLayers map[string]*layers.ParameterLayer,
	ps map[string]interface{},
) error {
	return nil
}

func (cmd *OakCommand) Description() *cmds.CommandDescription {
	d := cmd.description
	return &cmds.CommandDescription{
		Name:  d.Name,
		Short: d.Short,
		Long:  d.Long,
		Layout: &layout.Layout{
			Sections: d.Layout,
		},
		Flags:     d.Flags,
		Arguments: d.Arguments,
		Layers:    d.Layers,
		Parents:   d.Parents,
		Source:    d.Source,
	}
}

func NewOakCommandFromReader(r io.Reader, options ...OakCommandOption) (*OakCommand, error) {
	var cmd OakCommand
	err := yaml.NewDecoder(r).Decode(&cmd)
	if err != nil {
		return nil, err
	}

	for _, option := range options {
		option(&cmd)
	}
	return &cmd, nil
}

func NewOakCommand(options ...OakCommandOption) *OakCommand {
	cmd := OakCommand{}
	for _, option := range options {
		option(&cmd)
	}
	return &cmd
}

func SitterQueryErrorTypeToString(errorType sitter.QueryErrorType) string {
	switch errorType {
	case sitter.QueryErrorNone:
		return "none"
	case sitter.QueryErrorNodeType:
		return "node type"
	case sitter.QueryErrorField:
		return "field"
	case sitter.QueryErrorCapture:
		return "capture"
	case sitter.QueryErrorSyntax:
		return "syntax"
	default:
		return "unknown"
	}

}

func (cmd *OakCommand) ExecuteQueries(tree *sitter.Node, sourceCode []byte) (QueryResults, error) {
	if cmd.SitterLanguage == nil {
		lang, err := LanguageNameToSitterLanguage(cmd.Language)
		if err != nil {
			return nil, err
		}
		cmd.SitterLanguage = lang
	}
	results := make(map[string]*Result)
	for _, query := range cmd.Queries {
		matches := []Match{}

		// could parse queries up front and return an error if necessary
		q, err := sitter.NewQuery([]byte(query.Query), cmd.SitterLanguage)
		if err != nil {
			switch err := err.(type) {
			case *sitter.QueryError:
				line := 1
				for i := uint32(0); i < err.Offset; i++ {
					if query.Query[i] == '\n' {
						line++
					}
				}

				log.Error().
					Str("query_name", query.Name).
					Str("query", query.Query).
					Str("error", SitterQueryErrorTypeToString(err.Type)).
					Int("line", line).
					Msg("error parsing query")
				return nil, errors.Errorf("error parsing query %s: '%v' at line %d",
					query.Name, SitterQueryErrorTypeToString(err.Type), line)
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

func (cmd *OakCommand) Render(results QueryResults) (string, error) {
	tmpl, err := templating.CreateTemplate("oak").Parse(cmd.Template)
	if err != nil {
		return "", err
	}

	return cmd.RenderWithTemplate(results, tmpl)
}

func (cmd *OakCommand) RenderWithTemplate(results QueryResults, tmpl *template.Template) (string, error) {
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

func (cmd *OakCommand) RenderWithTemplateFile(results QueryResults, file string) (string, error) {
	tmpl, err := templating.CreateTemplate("oak").ParseFiles(file)
	if err != nil {
		return "", err
	}

	return cmd.RenderWithTemplate(results, tmpl)
}

func (cmd *OakCommand) ResultsToJSON(results QueryResults, f io.Writer) error {
	enc := json.NewEncoder(f)
	return enc.Encode(results)
}

func (cmd *OakCommand) ResultsToYAML(results QueryResults, f io.Writer) error {
	enc := yaml.NewEncoder(f)
	return enc.Encode(results)
}

func (cmd *OakCommand) Parse(ctx context.Context, code []byte) (*sitter.Tree, error) {
	if cmd.SitterLanguage == nil {
		lang, err := LanguageNameToSitterLanguage(cmd.Language)
		if err != nil {
			return nil, err
		}

		cmd.SitterLanguage = lang
	}

	parser := sitter.NewParser()
	parser.SetLanguage(cmd.SitterLanguage)
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
func (cmd *OakCommand) DumpTree(tree *sitter.Tree) {
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

type OakGlazeCommand struct {
	*OakCommand
}

func (c *OakGlazeCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp cmds.Processor,
) error {

	return nil

}
