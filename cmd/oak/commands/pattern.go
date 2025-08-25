package commands

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/go-go-golems/oak/pkg/api"
    pm "github.com/go-go-golems/oak/pkg/patternmatcher"
    "github.com/spf13/cobra"
)

// PatternCmd applies a PAIP pattern to the AST (converted to Lisp) and reports matches
var PatternCmd = &cobra.Command{
    Use:   "pattern",
    Short: "Run a PAIP pattern against source files (matches anywhere in the AST)",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        language, _ := cmd.Flags().GetString("language")
        patternStr, _ := cmd.Flags().GetString("pattern")
        patternFile, _ := cmd.Flags().GetString("pattern-file")
        includeAnonymous, _ := cmd.Flags().GetBool("include-anonymous")

        if language == "" {
            cobra.CheckErr(fmt.Errorf("--language is required"))
        }
        if patternStr == "" && patternFile == "" {
            cobra.CheckErr(fmt.Errorf("either --pattern or --pattern-file is required"))
        }
        if patternFile != "" {
            b, err := os.ReadFile(patternFile)
            cobra.CheckErr(err)
            patternStr = string(b)
        }
        patternStr = strings.TrimSpace(patternStr)

        // Parse pattern once
        pat, err := pm.Parse(patternStr)
        cobra.CheckErr(err)

        qb := api.NewQueryBuilder(api.WithLanguage(language))
        ctx := context.Background()

        totalMatches := 0
        for _, f := range args {
            filePath, err := filepath.Abs(f)
            cobra.CheckErr(err)
            expr, err := qb.ToLispExpression(ctx, filePath, includeAnonymous)
            cobra.CheckErr(err)

            matches := collectMatches(pat, expr)
            if len(matches) == 0 {
                continue
            }

            fmt.Printf("=== %s (matches: %d) ===\n", filePath, len(matches))
            for i, b := range matches {
                // Filter out the FAIL sentinel if present
                if pm.IsFail(b) {
                    continue
                }
                fmt.Printf("%d) %s\n", i+1, b.String())
            }
            totalMatches += len(matches)
        }

        if totalMatches == 0 {
            os.Exit(1)
        }
    },
}

func init() {
    PatternCmd.Flags().String("language", "", "Language of the source files (required)")
    PatternCmd.Flags().String("pattern", "", "PAIP pattern to run")
    PatternCmd.Flags().String("pattern-file", "", "Read pattern from file")
    PatternCmd.Flags().Bool("include-anonymous", false, "Include anonymous nodes in Lisp AST")
}

// collectMatches traverses the expression tree and returns all bindings for matches
func collectMatches(pattern pm.Expression, expr pm.Expression) []pm.Binding {
    var out []pm.Binding
    walkExpressions(expr, func(e pm.Expression) {
        b := pm.PatMatch(pattern, e, pm.NoBindings)
        if !pm.IsFail(b) {
            out = append(out, b)
        }
    })
    return out
}

// walkExpressions calls fn for the expression and all its sub-expressions
func walkExpressions(expr pm.Expression, fn func(pm.Expression)) {
    if expr == nil {
        return
    }
    fn(expr)
    if cons, ok := expr.(pm.Cons); ok {
        walkExpressions(cons.Car, fn)
        walkExpressions(cons.Cdr, fn)
    }
}




