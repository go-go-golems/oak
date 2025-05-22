package dump

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
)

// YAMLDumper implements the YAML format dumper
type YAMLDumper struct{}

// NodeYAML represents a tree node in YAML format
type NodeYAML struct {
	Type     string                 `yaml:"type"`
	Position string                 `yaml:"pos"`             // Format: "startLine,startCol-endLine,endCol"
	Bytes    string                 `yaml:"bytes,omitempty"` // Format: "startByte-endByte" (optional)
	Named    bool                   `yaml:"named,omitempty"`
	Missing  bool                   `yaml:"missing,omitempty"`
	Extra    bool                   `yaml:"extra,omitempty"`
	HasError bool                   `yaml:"has_error,omitempty"`
	Content  string                 `yaml:"content,omitempty"`
	Fields   map[string][]*NodeYAML `yaml:"fields,omitempty"`
	Children []*NodeYAML            `yaml:"children,omitempty"`
}

// Dump outputs the tree in YAML format
func (d *YAMLDumper) Dump(tree *sitter.Tree, source []byte, w io.Writer, options Options) error {
	var buildYAML func(n *sitter.Node) *NodeYAML
	buildYAML = func(n *sitter.Node) *NodeYAML {
		if n.IsNull() {
			return nil
		}

		nodeType := n.Type()
		// Skip pure whitespace nodes
		if options.SkipWhitespace {
			if matched, _ := regexp.MatchString(`^\s+$`, nodeType); matched {
				return nil
			}
		}

		// Create the YAML node using compact line/column position format
		startPoint := n.StartPoint()
		endPoint := n.EndPoint()

		// Convert to 1-based line/column numbers for better readability
		startLine := startPoint.Row + 1
		startCol := startPoint.Column + 1
		endLine := endPoint.Row + 1
		endCol := endPoint.Column + 1

		// Format: "startLine,startCol-endLine,endCol"
		position := fmt.Sprintf("%d,%d-%d,%d", startLine, startCol, endLine, endCol)

		node := &NodeYAML{
			Type:     nodeType,
			Position: position,
		}

		// Add byte range if requested
		if options.ShowBytes {
			node.Bytes = fmt.Sprintf("%d-%d", n.StartByte(), n.EndByte())
		}

		// Only include these fields if true and ShowAttributes is enabled
		if options.ShowAttributes {
			if n.IsNamed() {
				node.Named = true
			}
			if n.IsMissing() {
				node.Missing = true
			}
			if n.IsExtra() {
				node.Extra = true
			}
			if n.HasError() {
				node.HasError = true
			}
		}

		// Add content if source is provided and ShowContent is enabled
		if source != nil && options.ShowContent {
			content := n.Content(source)
			// Clean up content for YAML
			content = strings.ReplaceAll(content, "\n", "\\n")
			content = strings.ReplaceAll(content, "\t", "\\t")
			content = strings.ReplaceAll(content, "\r", "\\r")

			// Truncate very long content
			if len(content) > 60 {
				content = content[:57] + "..."
			}

			node.Content = content
		}

		// Process children
		fieldMap := make(map[string][]*NodeYAML)
		var plainChildren []*NodeYAML

		for i := 0; i < int(n.ChildCount()); i++ {
			fieldName := n.FieldNameForChild(i)
			child := buildYAML(n.Child(i))

			if child == nil {
				continue
			}

			if fieldName != "" {
				if fieldMap[fieldName] == nil {
					fieldMap[fieldName] = []*NodeYAML{}
				}
				fieldMap[fieldName] = append(fieldMap[fieldName], child)
			} else {
				plainChildren = append(plainChildren, child)
			}
		}

		if len(fieldMap) > 0 {
			node.Fields = fieldMap
		}

		if len(plainChildren) > 0 {
			node.Children = plainChildren
		}

		return node
	}

	rootYAML := buildYAML(tree.RootNode())
	if rootYAML == nil {
		return errors.New("failed to build YAML representation of tree")
	}

	enc := yaml.NewEncoder(w)
	defer func() {
		if err := enc.Close(); err != nil {
			log.Warn().Err(err).Msg("error closing yaml encoder")
		}
	}()
	return enc.Encode(rootYAML)
}
