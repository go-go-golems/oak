package dump

import (
    "fmt"
    "io"
    "strings"

    sitter "github.com/smacker/go-tree-sitter"
)

// DumpVerboseAST writes a detailed AST with line/col, byte offsets, and flags.
func DumpVerboseAST(node *sitter.Node, content []byte, w io.Writer) {
    if node == nil || node.IsNull() {
        return
    }

    var visit func(n *sitter.Node, depth int)
    visit = func(n *sitter.Node, depth int) {
        if n == nil || n.IsNull() {
            return
        }

        indent := strings.Repeat("  ", depth)

        start := n.StartPoint()
        end := n.EndPoint()
        // 1-based
        pos := fmt.Sprintf("%d:%d-%d:%d", start.Row+1, start.Column+1, end.Row+1, end.Column+1)

        flags := []string{}
        if n.IsNamed() {
            flags = append(flags, "named")
        } else {
            flags = append(flags, "anonymous")
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

        fmt.Fprintf(
            w,
            "%s[%s] %s (bytes: %d-%d) {%s}\n",
            indent,
            pos,
            n.Type(),
            n.StartByte(),
            n.EndByte(),
            strings.Join(flags, ","),
        )

        // Leaf content
        if n.ChildCount() == 0 && content != nil {
            text := n.Content(content)
            if len(text) > 0 {
                esc := strings.ReplaceAll(text, "\n", "\\n")
                esc = strings.ReplaceAll(esc, "\t", "\\t")
                if len(esc) > 80 {
                    esc = esc[:77] + "..."
                }
                fmt.Fprintf(w, "%s  Content: \"%s\"\n", indent, esc)
            }
        }

        for i := 0; i < int(n.ChildCount()); i++ {
            child := n.Child(i)
            field := n.FieldNameForChild(i)
            if field != "" {
                fmt.Fprintf(w, "%s  Field: %q\n", indent, field)
            }
            visit(child, depth+1)
        }
    }

    visit(node, 0)
}


