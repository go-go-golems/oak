package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

func executeQuery(tree *sitter.Node, query string, sourceCode []byte) []*sitter.Node {
	lang := golang.GetLanguage()
	q, err := sitter.NewQuery([]byte(query), lang)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create query")
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, tree)
	var results []*sitter.Node
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		m = qc.FilterPredicates(m, sourceCode)
		for _, capture := range m.Captures {
			fmt.Printf("string %s\n", q.CaptureNameForId(capture.Index))
			fmt.Printf("%d: %s\n", capture.Index, capture.Node.String())
		}
		results = append(results, m.Captures[0].Node)
	}
	return results
}

func main() {
	// Load the Go grammar
	lang := golang.GetLanguage()

	// Create a parser
	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	// Parse a Go file
	sourceCode := []byte(`
        package main

        import "fmt"

        func main() {
            fmt.Println("Hello, world!")
        }
    `)
	ctx := context.Background()
	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		panic(err)
	}

	// Execute a query
	query := `
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list) @parameters
  body: (block)
)
`
	results := executeQuery(tree.RootNode(), query, sourceCode)

	// Print the results
	for _, node := range results {
		fmt.Printf("%s: %s - %s\n", node.Type(), node.String(), sourceCode[node.StartByte():node.EndByte()])
	}

}
