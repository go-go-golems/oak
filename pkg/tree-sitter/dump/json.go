package dump

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
)

// JSONDumper implements the JSON format dumper
type JSONDumper struct{}

// NodeJSON represents a tree node in JSON format
type NodeJSON struct {
	Type      string                 `json:"type"`
	Position  string                 `json:"pos"`             // Format: "startLine,startCol-endLine,endCol"
	Bytes     string                 `json:"bytes,omitempty"` // Optional
	IsNamed   bool                   `json:"is_named,omitempty"`
	IsMissing bool                   `json:"is_missing,omitempty"`
	IsExtra   bool                   `json:"is_extra,omitempty"`
	HasError  bool                   `json:"has_error,omitempty"`
	Content   string                 `json:"content,omitempty"`
	Fields    map[string][]*NodeJSON `json:"fields,omitempty"`
	Children  []*NodeJSON            `json:"children,omitempty"`
}

// Dump outputs the tree in JSON format
func (d *JSONDumper) Dump(tree *sitter.Tree, source []byte, w io.Writer, options Options) error {
	var buildJSON func(n *sitter.Node) *NodeJSON
	buildJSON = func(n *sitter.Node) *NodeJSON {
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

		// Convert to 1-based line/column numbers for better readability
		startPoint := n.StartPoint()
		endPoint := n.EndPoint()
		startLine := startPoint.Row + 1
		startCol := startPoint.Column + 1
		endLine := endPoint.Row + 1
		endCol := endPoint.Column + 1

		// Format position as "startLine,startCol-endLine,endCol"
		position := fmt.Sprintf("%d,%d-%d,%d", startLine, startCol, endLine, endCol)

		// Create the JSON node
		node := &NodeJSON{
			Type:     nodeType,
			Position: position,
		}

		// Add byte range if requested
		if options.ShowBytes {
			node.Bytes = fmt.Sprintf("%d-%d", n.StartByte(), n.EndByte())
		}

		// Only include these fields if true and if ShowAttributes is enabled
		if options.ShowAttributes {
			if n.IsNamed() {
				node.IsNamed = true
			}
			if n.IsMissing() {
				node.IsMissing = true
			}
			if n.IsExtra() {
				node.IsExtra = true
			}
			if n.HasError() {
				node.HasError = true
			}
		}

		// Add content if source is provided and ShowContent is enabled
		if source != nil && options.ShowContent {
			content := n.Content(source)
			// For very long content, truncate
			if len(content) > 60 {
				content = content[:57] + "..."
			}
			node.Content = content
		}

		// Process children
		fieldMap := make(map[string][]*NodeJSON)
		var plainChildren []*NodeJSON

		for i := 0; i < int(n.ChildCount()); i++ {
			fieldName := n.FieldNameForChild(i)
			child := buildJSON(n.Child(i))

			if child == nil {
				continue
			}

			if fieldName != "" {
				if fieldMap[fieldName] == nil {
					fieldMap[fieldName] = []*NodeJSON{}
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

	rootJSON := buildJSON(tree.RootNode())
	if rootJSON == nil {
		return errors.New("failed to build JSON representation of tree")
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rootJSON)
}
