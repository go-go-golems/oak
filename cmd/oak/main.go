package main

import (
	"context"
	"github.com/go-go-golems/oak/pkg"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"os"
)

func main() {
	// load queries/example1.yaml

	f, err := os.Open("queries/example1.yaml")
	if err != nil {
		panic(err)
	}

	oak, err := pkg.NewOakCommandFromReader(f)
	if err != nil {
		panic(err)
	}

	// read pkg/queries.go
	sourceCode, err := os.ReadFile("pkg/queries.go")
	if err != nil {
		panic(err)
	}

	lang := golang.GetLanguage()

	// Create a parser
	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	ctx := context.Background()
	tree, err := parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		panic(err)
	}

	results := oak.ExecuteQueries(tree.RootNode(), sourceCode)
	s, err := oak.RenderTemplate(results)
	if err != nil {
		panic(err)
	}

	println(s)
}
