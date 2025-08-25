package dump

import (
    "fmt"
    "io"
    "strings"

    pm "github.com/go-go-golems/oak/pkg/patternmatcher"
)

type LispOptions struct {
    // Indent is the indentation string per nesting level. Defaults to two spaces if empty.
    Indent  string
    // Compact prints a single-line representation (delegates to expr.String()).
    Compact bool
}

// DumpLispExpression pretty-prints a pm.Expression as a Lisp S-expression.
func DumpLispExpression(expr pm.Expression, w io.Writer, opts LispOptions) error {
    if opts.Indent == "" {
        opts.Indent = "  "
    }
    if opts.Compact {
        _, err := fmt.Fprintln(w, exprString(expr))
        return err
    }
    return printExpr(expr, w, 0, opts)
}

func printExpr(expr pm.Expression, w io.Writer, depth int, opts LispOptions) error {
    if expr == nil {
        _, err := fmt.Fprint(w, "()")
        return err
    }

    switch e := expr.(type) {
    case pm.Symbol, pm.Atom:
        _, err := fmt.Fprint(w, exprString(e))
        return err
    case pm.Cons:
        // Gather list elements by traversing cons cells
        elements := consToSlice(e)
        if len(elements) == 0 {
            _, err := fmt.Fprint(w, "()")
            return err
        }

        // If list is a simple pair like (symbol atom), still pretty print on multiple lines for consistency
        indent := strings.Repeat(opts.Indent, depth)
        _, err := fmt.Fprint(w, "(")
        if err != nil { return err }

        // First element printed inline
        if err := printExpr(elements[0], w, depth, opts); err != nil { return err }

        // Subsequent elements on new lines, indented
        for i := 1; i < len(elements); i++ {
            if _, err := fmt.Fprint(w, "\n"); err != nil { return err }
            if _, err := fmt.Fprint(w, indent+opts.Indent); err != nil { return err }
            if err := printExpr(elements[i], w, depth+1, opts); err != nil { return err }
        }

        _, err = fmt.Fprint(w, ")")
        return err
    default:
        // Fallback to default string
        _, err := fmt.Fprint(w, exprString(e))
        return err
    }
}

func consToSlice(expr pm.Expression) []pm.Expression {
    var out []pm.Expression
    current := expr
    for current != nil {
        cons, ok := current.(pm.Cons)
        if !ok {
            // improper list tail
            out = append(out, current)
            break
        }
        out = append(out, cons.Car)
        current = cons.Cdr
    }
    return out
}

func exprString(e pm.Expression) string {
    if e == nil { return "()" }
    return e.String()
}




