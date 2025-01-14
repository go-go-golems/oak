package tree_sitter

import (
	"context"
	"github.com/go-go-golems/oak/pkg/cmds"
	"github.com/pkg/errors"
	"github.com/smacker/go-tree-sitter"
	"os"
)

type Capture struct {
	// Name if the capture name from the query
	Name string
	// Text is the actual text that was captured
	Text string
	// Type is the Treesitter type of the captured node
	Type string

	// TODO(manuel, 2023-04-23): Add more information about the capture
	// for example: offset, line number, filename, query name, ...
	StartByte  uint32
	EndByte    uint32
	StartPoint sitter.Point
	EndPoint   sitter.Point
}

type Match map[string]Capture

type Result struct {
	QueryName string
	Matches   []Match
}

func (r *Result) Clone() *Result {
	clone := &Result{
		QueryName: r.QueryName,
		Matches:   make([]Match, len(r.Matches)),
	}
	copy(clone.Matches, r.Matches)
	return clone
}

type SitterQuery struct {
	// Name of the resulting variable after parsing
	Name string `yaml:"name"`
	// Query contains the tree-sitter query that will be applied to the source code
	Query string `yaml:"query"`
	// Rendered keeps track if the Query was Rendered with RenderQueries.
	// This is an ugly way of doing things, but at least we'll signal at runtime
	// if the code tries to render a query multiple times.
	// See the NOTEs in RenderQueries.
	Rendered bool
}

type QueryResults map[string]*Result

// GetResultsByFile is a helper function that parses the given fileNames and
// returns a map of results by fileName.
func GetResultsByFile(
	ctx context.Context,
	fileNames []string,
	oc *cmds.OakCommand,
) (
	map[string]QueryResults, error) {
	resultsByFile := map[string]QueryResults{}

	lang, err := oc.GetLanguage()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get language for oak command")
	}

	for _, fileName := range fileNames {
		source, err := os.ReadFile(fileName)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read file %s", fileName)
		}

		tree, err := oc.Parse(ctx, nil, []byte(source))
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse file %s", fileName)
		}

		results, err := ExecuteQueries(lang, tree.RootNode(), oc.Queries, source)
		if err != nil {
			return nil, errors.Wrapf(err, "could not execute queries for file %s", fileName)
		}

		resultsByFile[fileName] = results
	}

	return resultsByFile, nil
}

// ExecuteQueries runs the given queries on the given tree and returns the
// results. Individual names are resolved using the sourceCode string, so as
// to provide full identifier names when matched.
//
// TODO(manuel, 2023-06-19) We only need the language from oc here, right?
func ExecuteQueries(
	lang *sitter.Language,
	tree *sitter.Node,
	queries []SitterQuery,
	sourceCode []byte,
) (QueryResults, error) {
	results := make(map[string]*Result)
	for _, query := range queries {
		matches := []Match{}

		// could parse queries up front and return an error if necessary
		q, err := sitter.NewQuery([]byte(query.Query), lang)
		if err != nil {
			switch err := err.(type) {
			case *sitter.QueryError:
				return nil, errors.Wrap(err, "error parsing query %s")
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
				name := q.CaptureNameForId(c.Index)
				match[name] = Capture{
					Name:       name,
					Text:       c.Node.Content(sourceCode),
					StartByte:  c.Node.StartByte(),
					EndByte:    c.Node.EndByte(),
					StartPoint: c.Node.StartPoint(),
					EndPoint:   c.Node.EndPoint(),
				}
			}

			m = qc.FilterPredicates(m, sourceCode)

			if len(m.Captures) == 0 {
				continue
			}

			match = Match{}
			for _, c := range m.Captures {
				name := q.CaptureNameForId(c.Index)
				content := string(sourceCode[c.Node.StartByte():c.Node.EndByte()])
				if m, ok := match[name]; ok {
					match[name] = Capture{
						Name:       name,
						Text:       m.Text + "\n" + content,
						StartByte:  m.StartByte,
						EndByte:    c.Node.EndByte(),
						StartPoint: m.StartPoint,
						EndPoint:   c.Node.EndPoint(),
					}
					continue
				}
				match[name] = Capture{
					Name:       name,
					Text:       content,
					Type:       c.Node.Type(),
					StartByte:  c.Node.StartByte(),
					EndByte:    c.Node.EndByte(),
					StartPoint: c.Node.StartPoint(),
					EndPoint:   c.Node.EndPoint(),
				}
			}
			matches = append(matches, match)
		}

		results[query.Name] = &Result{
			QueryName: query.Name,
			Matches:   matches,
		}
	}

	return results, nil
}
