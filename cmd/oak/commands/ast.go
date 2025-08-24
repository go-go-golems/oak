package commands

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/go-go-golems/oak/pkg/api"
    "github.com/go-go-golems/oak/pkg"
    tsdump "github.com/go-go-golems/oak/pkg/tree-sitter/dump"
    pm "github.com/go-go-golems/oak/pkg/patternmatcher"
    sitter "github.com/smacker/go-tree-sitter"
    "github.com/spf13/cobra"
)

var ASTCmd = &cobra.Command{
    Use:   "ast",
    Short: "Print AST of source files in various formats (lisp, verbose, text, json, yaml, xml)",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        language, _ := cmd.Flags().GetString("language")
        format, _ := cmd.Flags().GetString("format")
        includeAnonymous, _ := cmd.Flags().GetBool("include-anonymous")

        if language == "" {
            cobra.CheckErr(fmt.Errorf("--language is required"))
        }

        // Normalize format
        switch format {
        case "lisp", "verbose", "text", "json", "yaml", "xml":
        default:
            cobra.CheckErr(fmt.Errorf("invalid --format: %s", format))
        }

        // Prepare parser if needed
        var lang *sitter.Language
        var err error
        if format != "lisp" {
            lang, err = pkg.LanguageNameToSitterLanguage(language)
            cobra.CheckErr(err)
        }

        qb := api.NewQueryBuilder(api.WithLanguage(language))
        ctx := context.Background()

        for _, f := range args {
            filePath, err := filepath.Abs(f)
            cobra.CheckErr(err)
            content, err := os.ReadFile(filePath)
            cobra.CheckErr(err)

            fmt.Printf("=== %s (%s) ===\n", filePath, format)

            switch format {
            case "lisp":
                expr, err := qb.ToLispExpression(ctx, filePath, includeAnonymous)
                cobra.CheckErr(err)
                // Pretty print by default
                _ = tsdump.DumpLispExpression(expr, os.Stdout, tsdump.LispOptions{Indent: "  ", Compact: false})
                if _, ok := expr.(pm.Cons); ok {
                    fmt.Println()
                }
            case "verbose":
                parser := sitter.NewParser()
                parser.SetLanguage(lang)
                tree, err := parser.ParseCtx(ctx, nil, content)
                cobra.CheckErr(err)
                defer tree.Close()
                tsdump.DumpVerboseAST(tree.RootNode(), content, os.Stdout)
            default:
                parser := sitter.NewParser()
                parser.SetLanguage(lang)
                tree, err := parser.ParseCtx(ctx, nil, content)
                cobra.CheckErr(err)
                defer tree.Close()

                dumper := tsdump.NewDumper(tsdump.Format(format))
                // Reasonable defaults
                opts := tsdump.Options{
                    ShowBytes:      true,
                    ShowContent:    false,
                    ShowAttributes: true,
                    SkipWhitespace: false,
                }
                cobra.CheckErr(dumper.Dump(tree, content, os.Stdout, opts))
            }
        }
    },
}

func init() {
    ASTCmd.Flags().String("language", "", "Language of the source files (required)")
    ASTCmd.Flags().String("format", "lisp", "Output format: lisp|verbose|text|json|yaml|xml")
    ASTCmd.Flags().Bool("include-anonymous", false, "Include anonymous nodes in lisp output")
}


