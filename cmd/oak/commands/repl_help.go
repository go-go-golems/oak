package commands

import (
    "context"
    "fmt"
    "strings"

    replhelp "github.com/go-go-golems/bobatea/pkg/repl/help"
    helpadapters "github.com/go-go-golems/bobatea/pkg/repl/help/adapters"
    "github.com/go-go-golems/glazed/pkg/help"
    "github.com/spf13/cobra"
)

// NewReplHelpCmd exposes the reusable REPL help backend as a CLI command for testing.
// We avoid clashing with existing glazed help by naming it "repl-help".
func NewReplHelpCmd(hs *help.HelpSystem) *cobra.Command {
    var (
        flagAll     bool
        flagQuery   string
        flagTypes   []string
        flagTopics  []string
        flagFlags   []string
        flagCmds    []string
        flagSearch  string
        flagList    bool
    )

    cmd := &cobra.Command{
        Use:   "repl-help [slug]",
        Short: "Help powered by the REPL backend (slug, --all, optional --query)",
        Args:  cobra.ArbitraryArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            backend := &helpadapters.GlazedBackend{HS: hs}
            // Build a synthetic /help input to reuse the same parser
            var parts []string
            parts = append(parts, "/help")
            if len(args) > 0 {
                parts = append(parts, args[0])
            }
            if flagAll { parts = append(parts, "--all") }
            if strings.TrimSpace(flagQuery) != "" { parts = append(parts, "--query=\""+flagQuery+"\"") }
            for _, t := range flagTypes { parts = append(parts, "--type="+t) }
            for _, t := range flagTopics { parts = append(parts, "--topic="+t) }
            for _, f := range flagFlags { parts = append(parts, "--flag="+f) }
            for _, c := range flagCmds { parts = append(parts, "--command="+c) }
            if strings.TrimSpace(flagSearch) != "" { parts = append(parts, "--search=\""+flagSearch+"\"") }
            if flagList { parts = append(parts, "--list") }

            input := strings.Join(parts, " ")
            md := replhelp.HandleHelpCommand(context.Background(), replhelp.Config{
                Backend:     backend,
                ShowRelated: true,
                Renderer:    replhelp.DefaultRenderer(),
            }, input)
            fmt.Print(md)
            return nil
        },
    }

    cmd.Flags().BoolVar(&flagAll, "all", false, "show top-level help")
    cmd.Flags().StringVar(&flagQuery, "query", "", "DSL query")
    cmd.Flags().StringSliceVar(&flagTypes, "type", nil, "filter by type (repeatable)")
    cmd.Flags().StringSliceVar(&flagTopics, "topic", nil, "filter by topic (repeatable)")
    cmd.Flags().StringSliceVar(&flagFlags, "flag", nil, "filter by flag (repeatable)")
    cmd.Flags().StringSliceVar(&flagCmds, "command", nil, "filter by command (repeatable)")
    cmd.Flags().StringVar(&flagSearch, "search", "", "full-text search (quoted)")
    cmd.Flags().BoolVar(&flagList, "list", false, "list mode for single result")

    return cmd
}


