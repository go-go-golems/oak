package tree_sitter

import (
    pm "github.com/go-go-golems/oak/pkg/patternmatcher"
    sitter "github.com/smacker/go-tree-sitter"
)

// NodeToLispExpression converts a tree-sitter node into a Lisp-style S-expression
// represented using the patternmatcher Expression types.
//
// The resulting expression has the structure:
//   (node_type [ (field_name child) | child ] ...)
//
// - node_type is a pm.Symbol with the node's Type()
// - Fields are represented as 2-element lists: (field_name child)
// - Anonymous children without a field are included directly
// - Leaf nodes are represented as a single-element list: (node_type)
func NodeToLispExpression(node *sitter.Node, content []byte, includeAnonymous bool) pm.Expression {
    if node == nil || node.IsNull() {
        return nil
    }

    // Start with the node type symbol
    elements := []pm.Expression{pm.Symbol{Name: node.Type()}}

    childCount := int(node.ChildCount())
    if childCount == 0 {
        // Include leaf content as Atom for matching text
        if content != nil {
            text := node.Content(content)
            if text != "" {
                elements = append(elements, pm.Atom{Value: text})
            }
        }
    }
    for i := 0; i < childCount; i++ {
        child := node.Child(i)
        if child == nil || child.IsNull() {
            continue
        }

        if !includeAnonymous && !child.IsNamed() {
            // Skip anonymous nodes unless explicitly requested
            continue
        }

        childExpr := NodeToLispExpression(child, content, includeAnonymous)
        fieldName := node.FieldNameForChild(i)
        if fieldName != "" {
            // Represent as (field childExpr)
            pair := pm.SliceToCons([]pm.Expression{
                pm.Symbol{Name: fieldName},
                childExpr,
            })
            elements = append(elements, pair)
        } else {
            elements = append(elements, childExpr)
        }
    }

    return pm.SliceToCons(elements)
}


