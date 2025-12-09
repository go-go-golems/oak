package guru

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// SymbolFinder finds symbols in Go files using Tree-sitter
type SymbolFinder struct {
	parser   *sitter.Parser
	language *sitter.Language
}

// NewSymbolFinder creates a new symbol finder
func NewSymbolFinder() *SymbolFinder {
	parser := sitter.NewParser()
	language := golang.GetLanguage()
	parser.SetLanguage(language)

	return &SymbolFinder{
		parser:   parser,
		language: language,
	}
}

// SymbolPosition represents a symbol's location
type SymbolPosition struct {
	File      string
	StartByte uint32
	EndByte   uint32
	Line      int
	Column    int
	Type      string // function, type, method, variable, const
}

// ToGuruPosition converts to guru position format
func (sp *SymbolPosition) ToGuruPosition() string {
	return fmt.Sprintf("%s:#%d", sp.File, sp.StartByte)
}

// FindSymbol finds a symbol in a file and returns its position
func (sf *SymbolFinder) FindSymbol(ctx context.Context, filePath, symbolName string) (*SymbolPosition, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", filePath)
	}

	var tree *sitter.Tree
	tree, err = sf.parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse file %s", filePath)
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	return sf.findSymbolInTree(rootNode, content, filePath, symbolName)
}

// findSymbolInTree searches for a symbol in the parsed tree
func (sf *SymbolFinder) findSymbolInTree(tree *sitter.Node, content []byte, filePath, symbolName string) (*SymbolPosition, error) {
	queries := []struct {
		name  string
		query string
	}{
		{"function", `(function_declaration name: (identifier) @name)`},
		{"type", `(type_declaration name: (type_identifier) @name)`},
		{"method", `(method_declaration name: (field_identifier) @name)`},
		{"variable", `(var_declaration (var_spec name: (identifier) @name))`},
		{"const", `(const_declaration (const_spec name: (identifier) @name))`},
	}

	for _, q := range queries {
		query, err := sitter.NewQuery([]byte(q.query), sf.language)
		if err != nil {
			continue
		}

		cursor := sitter.NewQueryCursor()
		cursor.Exec(query, tree)

		for {
			match, ok := cursor.NextMatch()
			if !ok {
				break
			}

			for _, capture := range match.Captures {
				name := query.CaptureNameForId(capture.Index)
				if name == "name" {
					text := string(content[capture.Node.StartByte():capture.Node.EndByte()])
					if text == symbolName {
						startPoint := capture.Node.StartPoint()
						return &SymbolPosition{
							File:      filePath,
							StartByte: capture.Node.StartByte(),
							EndByte:   capture.Node.EndByte(),
							Line:      int(startPoint.Row) + 1,
							Column:    int(startPoint.Column) + 1,
							Type:      q.name,
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("symbol %s not found in %s", symbolName, filePath)
}

// StandaloneFindSymbol is a convenience function that doesn't require
// creating a SymbolFinder instance
func StandaloneFindSymbol(ctx context.Context, filePath, symbolName string) (string, error) {
	finder := NewSymbolFinder()
	pos, err := finder.FindSymbol(ctx, filePath, symbolName)
	if err != nil {
		return "", err
	}
	return pos.ToGuruPosition(), nil
}
