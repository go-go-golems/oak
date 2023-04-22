package pkg

import (
	"bytes"
	"encoding/json"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"gopkg.in/yaml.v3"
	"io"
)

type OakCommand struct {
	Language string
	Queries  []Query
	Template string
}

type Query struct {
	// Name of the resulting variable after parsing
	Name string
	// Query contains the tree-sitter query that will be applied to the source code
	Query string
}

type Capture struct {
	// Name if the capture name from the query
	Name string
	// Text is the actual text that was captured
	Text string
}

type Match map[string]Capture

type Result struct {
	// Name is the name of the query
	Name string
	// Matches are the matches for the query
	Matches []*Match
}

type QueryResults map[string]*Result

func NewOakCommandFromReader(r io.Reader) (*OakCommand, error) {
	var cmd OakCommand
	err := yaml.NewDecoder(r).Decode(&cmd)
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (cmd *OakCommand) ExecuteQueries(tree *sitter.Node, sourceCode []byte) QueryResults {
	results := make(map[string]*Result)
	for _, query := range cmd.Queries {
		matches := []*Match{}
		q, err := sitter.NewQuery([]byte(query.Query),
			golang.GetLanguage())
		if err != nil {
			continue
		}
		qc := sitter.NewQueryCursor()
		qc.Exec(q, tree)
		for {
			m, ok := qc.NextMatch()
			if !ok {
				break
			}
			m = qc.FilterPredicates(m, sourceCode)

			match := Match{}
			for _, c := range m.Captures {
				match[q.CaptureNameForId(c.Index)] = Capture{
					Name: q.CaptureNameForId(c.Index),
					Text: c.Node.Content(sourceCode),
				}
			}
			matches = append(matches, &match)
		}

		results[query.Name] = &Result{
			Name:    query.Name,
			Matches: matches,
		}
	}

	return results
}

func (cmd *OakCommand) RenderTemplate(results QueryResults) (string, error) {
	tmpl, err := templating.CreateTemplate("oak").Parse(cmd.Template)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, results)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (cmd *OakCommand) ResultsToJSON(results QueryResults, f io.Writer) error {
	enc := json.NewEncoder(f)
	return enc.Encode(results)
}

func (cmd *OakCommand) ResultsToYAML(results QueryResults, f io.Writer) error {
	enc := yaml.NewEncoder(f)
	return enc.Encode(results)
}
