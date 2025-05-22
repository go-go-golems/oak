package dump

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// TextDumper implements the enhanced text format dumper
type TextDumper struct{}

// Dump outputs the tree in enhanced text format
func (d *TextDumper) Dump(tree *sitter.Tree, source []byte, w io.Writer, options Options) error {
	var visitEnhanced func(n *sitter.Node, name string, depth int) error
	visitEnhanced = func(n *sitter.Node, name string, depth int) error {
		if n.IsNull() {
			return nil
		}

		nodeType := n.Type()
		// Skip whitespace nodes if configured
		if options.SkipWhitespace {
			if matched, _ := regexp.MatchString(`^\s+$`, nodeType); matched {
				return nil
			}
		}

		indent := strings.Repeat("  ", depth)
		prefix := ""
		if name != "" {
			prefix = name + ": "
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

		// Basic node info with position
		fmt.Fprintf(w, "%s%s%s [%s]", indent, prefix, nodeType, position)

		// Add byte offsets if requested
		if options.ShowBytes {
			fmt.Fprintf(w, " bytes:%d-%d", n.StartByte(), n.EndByte())
		}

		// Additional attributes if requested
		if options.ShowAttributes {
			flags := []string{}
			if n.IsNamed() {
				flags = append(flags, "named")
			}
			if n.IsMissing() {
				flags = append(flags, "missing")
			}
			if n.IsExtra() {
				flags = append(flags, "extra")
			}
			if n.HasError() {
				flags = append(flags, "error")
			}

			if len(flags) > 0 {
				fmt.Fprintf(w, " [%s]", strings.Join(flags, ","))
			}
		}

		// Content if source is provided and requested
		if source != nil && options.ShowContent {
			content := n.Content(source)
			// Escape newlines and other control characters for display
			content = strings.ReplaceAll(content, "\n", "\\n")
			content = strings.ReplaceAll(content, "\t", "\\t")
			content = strings.ReplaceAll(content, "\r", "\\r")

			// Truncate very long content
			if len(content) > 60 {
				content = content[:57] + "..."
			}

			fmt.Fprintf(w, " \"%s\"", content)
		}

		fmt.Fprintln(w)

		// Visit children
		for i := 0; i < int(n.ChildCount()); i++ {
			fieldName := n.FieldNameForChild(i)
			err := visitEnhanced(n.Child(i), fieldName, depth+1)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return visitEnhanced(tree.RootNode(), "", 0)
}
