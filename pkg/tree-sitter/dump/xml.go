package dump

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// XMLDumper implements the XML format dumper
type XMLDumper struct{}

// Dump outputs the tree in XML format
func (d *XMLDumper) Dump(tree *sitter.Tree, source []byte, w io.Writer, options Options) error {
	// Write XML header
	fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(w, "<tree>\n")

	var visitXML func(n *sitter.Node, depth int) error
	visitXML = func(n *sitter.Node, depth int) error {
		if n.IsNull() {
			return nil
		}

		indent := strings.Repeat("  ", depth)
		nodeType := n.Type()

		// Skip pure whitespace nodes
		if options.SkipWhitespace {
			if matched, _ := regexp.MatchString(`^\s+$`, nodeType); matched {
				return nil
			}
		}

		// Get node text content if source is provided
		var contentAttr string
		if source != nil && options.ShowContent {
			content := n.Content(source)
			// XML-escape the content
			content = strings.ReplaceAll(content, "&", "&amp;")
			content = strings.ReplaceAll(content, "<", "&lt;")
			content = strings.ReplaceAll(content, ">", "&gt;")
			content = strings.ReplaceAll(content, "\"", "&quot;")
			contentAttr = fmt.Sprintf(" content=\"%s\"", content)
		}

		// Convert to 1-based line/column numbers for better readability
		startPoint := n.StartPoint()
		endPoint := n.EndPoint()
		startLine := startPoint.Row + 1
		startCol := startPoint.Column + 1
		endLine := endPoint.Row + 1
		endCol := endPoint.Column + 1

		// Format position as "startLine,startCol-endLine,endCol"
		posAttr := fmt.Sprintf(" pos=\"%d,%d-%d,%d\"", startLine, startCol, endLine, endCol)

		// Optionally include byte positions
		bytesAttr := ""
		if options.ShowBytes {
			bytesAttr = fmt.Sprintf(" bytes=\"%d-%d\"", n.StartByte(), n.EndByte())
		}

		attributesStr := ""
		if options.ShowAttributes {
			attributesStr = fmt.Sprintf(" named=\"%t\" missing=\"%t\" extra=\"%t\" has_error=\"%t\"",
				n.IsNamed(),
				n.IsMissing(),
				n.IsExtra(),
				n.HasError())
		}

		// Output opening tag with attributes
		fmt.Fprintf(w, "%s<node type=\"%s\"%s%s%s%s>\n",
			indent,
			nodeType,
			posAttr,
			bytesAttr,
			attributesStr,
			contentAttr,
		)

		// Visit children
		for i := 0; i < int(n.ChildCount()); i++ {
			fieldName := n.FieldNameForChild(i)
			child := n.Child(i)

			if fieldName != "" {
				fmt.Fprintf(w, "%s  <field name=\"%s\">\n", indent, fieldName)
				err := visitXML(child, depth+2)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "%s  </field>\n", indent)
			} else {
				err := visitXML(child, depth+1)
				if err != nil {
					return err
				}
			}
		}

		// Output closing tag
		fmt.Fprintf(w, "%s</node>\n", indent)
		return nil
	}

	err := visitXML(tree.RootNode(), 1)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "</tree>\n")
	return nil
}
