package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/bobatea/pkg/logutil"
	"github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/bobatea/pkg/repl/slash"
	"github.com/go-go-golems/bobatea/pkg/autocomplete"
	replhelp "github.com/go-go-golems/bobatea/pkg/repl/help"
	helpadapters "github.com/go-go-golems/bobatea/pkg/repl/help/adapters"
	"github.com/go-go-golems/glazed/pkg/help"
	oakdoc "github.com/go-go-golems/oak/pkg/doc"
	"github.com/go-go-golems/oak/pkg"
	"github.com/go-go-golems/oak/pkg/api"
	pm "github.com/go-go-golems/oak/pkg/patternmatcher"
	tsdump "github.com/go-go-golems/oak/pkg/tree-sitter/dump"
	"github.com/rs/zerolog"
	sitter "github.com/smacker/go-tree-sitter"
)

var astFenceLang string
var astMaxWidth int
var astBreakHeads map[string]map[string]struct{}
var astInlineFields map[string]map[string]struct{}
var astCompactLangs map[string]struct{}
var helpSys *help.HelpSystem

func parseLangMapList(spec string) map[string]map[string]struct{} {
	res := map[string]map[string]struct{}{}
	if strings.TrimSpace(spec) == "" {
		return res
	}
	langs := strings.Split(spec, ";")
	for _, entry := range langs {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}
		lang := strings.TrimSpace(parts[0])
		vals := strings.Split(parts[1], ",")
		m := map[string]struct{}{}
		for _, v := range vals {
			v = strings.TrimSpace(v)
			if v != "" {
				m[v] = struct{}{}
			}
		}
		res[lang] = m
	}
	return res
}

func parseLangSet(spec string) map[string]struct{} {
	res := map[string]struct{}{}
	if strings.TrimSpace(spec) == "" {
		return res
	}
	vals := strings.Split(spec, ",")
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" {
			res[v] = struct{}{}
		}
	}
	return res
}

type PatternEvaluator struct {
	currentFile     string
	currentLanguage string
	content         []byte
	lispAST         pm.Expression
}

func (e *PatternEvaluator) EvaluateStream(ctx context.Context, code string, emit func(repl.Event)) error {
	in := strings.TrimSpace(code)
	if in == "" {
		return nil
	}

	// All /commands are handled by the REPL slash-command dispatcher before EvaluateStream.
	// If any slipped through, just show a brief notice.
	if strings.HasPrefix(in, "/") {
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Unknown command. Type /help for available commands."}})
		return nil
	}

	if e.lispAST != nil {
		pat, err := pm.Parse(in)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		matches := collectMatches(pat, e.lispAST)
		if len(matches) == 0 {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "NO MATCH"}})
			return nil
		}
		var b strings.Builder
		for i, bind := range matches {
			if pm.IsFail(bind) {
				continue
			}
			if i > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString("```" + astFenceLang + "\n")
			b.WriteString(bind.String())
			b.WriteString("\n```")
		}
		md := fmt.Sprintf("matches: %d\n\n%s", len(matches), b.String())
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": md}})
		return nil
	}

	emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Unknown input. Type /help for commands or load an AST and enter a pattern."}})
	return nil
}

func (e *PatternEvaluator) GetPrompt() string        { return "oak-pattern> " }
func (e *PatternEvaluator) GetName() string          { return "Oak Pattern Matcher" }
func (e *PatternEvaluator) SupportsMultiline() bool  { return true }
func (e *PatternEvaluator) GetFileExtension() string { return ".pattern" }

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(s) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error", "err":
		return zerolog.ErrorLevel
	default:
		return zerolog.ErrorLevel
	}
}

func main() {
	// CLI flags for logging and formatting
	ll := flag.String("log-level", "error", "log level: trace, debug, info, warn, error")
	lf := flag.String("log-file", "", "log file path (optional)")
	astLang := flag.String("ast-lang", "lisp", "code fence language for AST highlighting (e.g., lisp, scheme, clojure)")
	astWidth := flag.Int("ast-width", 80, "maximum line width for AST pretty printing")
	astBreakHeadsFlag := flag.String("ast-break-heads", "", "per-language heads to force line breaks: lang:head1,head2;lang2:head3")
	astInlineFieldsFlag := flag.String("ast-inline-fields", "go:type,name,result,body", "per-language field names to keep inline: lang:field1,field2;...")
	astCompactLangsFlag := flag.String("ast-compact-langs", "", "languages that only break lines if width exceeded (comma-separated)")
	flag.Parse()

	astFenceLang = *astLang
	astMaxWidth = *astWidth
	astBreakHeads = parseLangMapList(*astBreakHeadsFlag)
	astInlineFields = parseLangMapList(*astInlineFieldsFlag)
	astCompactLangs = parseLangSet(*astCompactLangsFlag)

	level := parseLevel(*ll)
	if *lf != "" {
		logutil.InitTUILoggingToFile(level, *lf)
	} else {
		logutil.InitTUILoggingToDiscard(level)
	}

	// Initialize help system from embedded oak docs and local glazed topics
	helpSys = help.NewHelpSystem()
	_ = oakdoc.AddDocToHelpSystem(helpSys)

	evaluator := &PatternEvaluator{}
	config := repl.DefaultConfig()
	config.Title = "Oak Pattern Matcher REPL"
	config.Prompt = "oak> "
	config.HelperMarkdown = patternCheatSheet()
	// Use ESC to toggle between input and timeline focus (so Tab can be used for completion)
	config.FocusToggleKey = "esc"

	bus, model, p, err := repl.NewTimelineRepl(evaluator, config)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Register slash commands
	registerSlashCommands(model.SlashRegistry(), evaluator)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := make(chan error, 2)
	go func() { errs <- bus.Run(ctx) }()
	go func() { _, e := p.Run(); cancel(); errs <- e }()
	if e := <-errs; e != nil {
		log.Println(e)
		os.Exit(1)
	}
}

func registerSlashCommands(reg slash.Registry, e *PatternEvaluator) {
    // /help
    reg.Register(&slash.Command{
        Name:    "help",
        Summary: "Show help topics, examples, and tutorials",
        Usage:   "/help [slug] [--all] [--query=DSL]",
        Schema:  slash.Schema{},
        Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
            backend := &helpadapters.GlazedBackend{HS: helpSys}
            // rebuild input line minimally
            s := "/help"
            if len(in.Positionals) > 0 { s += " " + in.Positionals[0] }
            if _, ok := in.Flags["all"]; ok { s += " --all" }
            if qv, ok := in.Flags["query"]; ok && len(qv) > 0 { s += " --query=\"" + qv[0] + "\"" }
            md := replhelp.HandleHelpCommand(ctx, replhelp.Config{
                Backend:     backend,
                ShowRelated: true,
                Renderer:    replhelp.DefaultRenderer(),
            }, s)
            emit(string(repl.EventResultMarkdown), map[string]any{"markdown": md})
            return nil
        },
        Complete: func(ctx context.Context, st slash.CompletionState) ([]autocomplete.Suggestion, error) {
            // suggest slugs from top-level page when completing first positional
            if st.Phase != slash.PhasePositional || st.PositionalIndex != 0 { return nil, nil }
            page := helpSys.GetTopLevelHelpPage()
            var ss []autocomplete.Suggestion
            add := func(sections []*help.Section) {
                for _, s := range sections {
                    if st.Partial == "" || strings.HasPrefix(s.Slug, st.Partial) {
                        disp := s.Slug
                        if strings.TrimSpace(s.Title) != "" { disp += " — " + s.Title }
                        ss = append(ss, autocomplete.Suggestion{Id: s.Slug, Value: s.Slug, DisplayText: disp})
                    }
                }
            }
            add(page.AllGeneralTopics)
            add(page.AllExamples)
            add(page.AllApplications)
            add(page.AllTutorials)
            return ss, nil
        },
    })

    // /lang
    reg.Register(&slash.Command{
        Name:    "lang",
        Summary: "Set current language",
        Usage:   "/lang <language>",
        Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
            if len(in.Positionals) < 1 {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "Usage: /lang <language>"})
                return nil
            }
            e.currentLanguage = in.Positionals[0]
            emit(string(repl.EventLog), map[string]any{"level": "info", "message": "language set", "fields": map[string]any{"language": e.currentLanguage}})
            return nil
        },
    })

    // /load
    reg.Register(&slash.Command{
        Name:    "load",
        Summary: "Load a source file and infer language",
        Usage:   "/load <file>",
        Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
            if len(in.Positionals) < 1 {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "Usage: /load <file>"})
                return nil
            }
            path := in.Positionals[0]
            b, err := os.ReadFile(path)
            if err != nil {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)})
                return nil
            }
            e.currentFile = path
            e.content = b
            if lang := inferLanguageFromFilename(path); lang != "" { e.currentLanguage = lang }
            emit(string(repl.EventLog), map[string]any{"level": "info", "message": fmt.Sprintf("loaded %s (%d bytes)", path, len(b)), "fields": map[string]any{"language": e.currentLanguage}})
            return nil
        },
    })

    // /ast
    reg.Register(&slash.Command{
        Name:    "ast",
        Summary: "Pretty-print Lisp AST",
        Usage:   "/ast [outFile]",
        Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
            var outFile string
            if len(in.Positionals) >= 1 { outFile = in.Positionals[0] }
            if e.currentFile == "" || len(e.content) == 0 {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "No file loaded. Use /load <file> first."})
                return nil
            }
            if e.currentLanguage == "" { if lang := inferLanguageFromFilename(e.currentFile); lang != "" { e.currentLanguage = lang } }
            qb := api.NewQueryBuilder(api.WithLanguage(e.currentLanguage))
            expr, err := qb.ToLispExpression(ctx, e.currentFile, false)
            if err != nil {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)})
                return nil
            }
            e.lispAST = expr
            pp := prettyPrintLisp(expr, e.currentLanguage)
            if outFile != "" {
                if err := safeWriteFile(outFile, []byte(pp)); err != nil {
                    emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error writing %s: %v", outFile, err)})
                    return nil
                }
                emit(string(repl.EventLog), map[string]any{"level": "info", "message": fmt.Sprintf("AST written to %s", outFile)})
                return nil
            }
            md := "```" + astFenceLang + "\n" + pp + "\n```"
            emit(string(repl.EventResultMarkdown), map[string]any{"markdown": md})
            return nil
        },
    })

    // /raw-ast
    reg.Register(&slash.Command{
        Name:    "raw-ast",
        Summary: "Verbose tree with positions/bytes/flags",
        Usage:   "/raw-ast [outFile]",
        Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
            var outFile string
            if len(in.Positionals) >= 1 { outFile = in.Positionals[0] }
            if e.currentFile == "" || len(e.content) == 0 {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "No file loaded. Use /load <file> first."})
                return nil
            }
            if e.currentLanguage == "" { if lang := inferLanguageFromFilename(e.currentFile); lang != "" { e.currentLanguage = lang } }
            lang, err := pkg.LanguageNameToSitterLanguage(e.currentLanguage)
            if err != nil {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)})
                return nil
            }
            parser := sitter.NewParser()
            parser.SetLanguage(lang)
            tree, err := parser.ParseCtx(ctx, nil, e.content)
            if err != nil { emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}); return nil }
            defer tree.Close()
            var buf bytes.Buffer
            tsdump.DumpVerboseAST(tree.RootNode(), e.content, &buf)
            if outFile != "" {
                if err := safeWriteFile(outFile, buf.Bytes()); err != nil {
                    emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error writing %s: %v", outFile, err)})
                    return nil
                }
                emit(string(repl.EventLog), map[string]any{"level": "info", "message": fmt.Sprintf("Raw AST written to %s", outFile)})
                return nil
            }
            md := "```text\n" + buf.String() + "\n```"
            emit(string(repl.EventResultMarkdown), map[string]any{"markdown": md})
            return nil
        },
    })

    // /yaml-ast
    reg.Register(&slash.Command{
        Name:    "yaml-ast",
        Summary: "Full YAML dump of the AST",
        Usage:   "/yaml-ast [outFile]",
        Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
            var outFile string
            if len(in.Positionals) >= 1 { outFile = in.Positionals[0] }
            if e.currentFile == "" || len(e.content) == 0 {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "No file loaded. Use /load <file> first."})
                return nil
            }
            if e.currentLanguage == "" { if lang := inferLanguageFromFilename(e.currentFile); lang != "" { e.currentLanguage = lang } }
            lang, err := pkg.LanguageNameToSitterLanguage(e.currentLanguage)
            if err != nil { emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}); return nil }
            parser := sitter.NewParser()
            parser.SetLanguage(lang)
            tree, err := parser.ParseCtx(ctx, nil, e.content)
            if err != nil { emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}); return nil }
            defer tree.Close()
            var buf bytes.Buffer
            dumper := tsdump.NewDumper(tsdump.Format("yaml"))
            opts := tsdump.Options{ShowBytes: true, ShowContent: true, ShowAttributes: true, SkipWhitespace: false}
            if err := dumper.Dump(tree, e.content, &buf, opts); err != nil { emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}); return nil }
            if outFile != "" {
                if err := safeWriteFile(outFile, buf.Bytes()); err != nil { emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error writing %s: %v", outFile, err)}); return nil }
                emit(string(repl.EventLog), map[string]any{"level": "info", "message": fmt.Sprintf("YAML AST written to %s", outFile)})
                return nil
            }
            md := "```yaml\n" + buf.String() + "\n```"
            emit(string(repl.EventResultMarkdown), map[string]any{"markdown": md})
            return nil
        },
    })

    // /pattern
    reg.Register(&slash.Command{
        Name:    "pattern",
        Summary: "Run a pattern match against current AST",
        Usage:   "/pattern <pattern>",
        Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
            if len(in.Positionals) < 1 {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "Usage: /pattern <pattern>"})
                return nil
            }
            if e.lispAST == nil {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "No AST; run /ast first"})
                return nil
            }
            raw := strings.TrimSpace(in.Positionals[0])
            pat, err := pm.Parse(raw)
            if err != nil {
                emit(string(repl.EventResultMarkdown), map[string]any{"markdown": fmt.Sprintf("Error: %v", err)})
                return nil
            }
            matches := collectMatches(pat, e.lispAST)
            if len(matches) == 0 { emit(string(repl.EventResultMarkdown), map[string]any{"markdown": "NO MATCH"}); return nil }
            var b strings.Builder
            for i, bind := range matches {
                if pm.IsFail(bind) { continue }
                if i > 0 { b.WriteString("\n\n") }
                b.WriteString("```" + astFenceLang + "\n")
                b.WriteString(bind.String())
                b.WriteString("\n```")
            }
            md := fmt.Sprintf("matches: %d\n\n%s", len(matches), b.String())
            emit(string(repl.EventResultMarkdown), map[string]any{"markdown": md})
            return nil
        },
    })
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

// Helpers
func inferLanguageFromFilename(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cc", ".cpp", ".cxx", ".hpp", ".h":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".lua":
		return "lua"
	case ".sh":
		return "bash"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		return ""
	}
}

func safeWriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644) // #nosec G306
}

func splitFrontMatter(content string) (yamlPart string, body string) {
	s := strings.TrimSpace(content)
	if !strings.HasPrefix(s, "---") {
		return "", s
	}
	rest := strings.TrimPrefix(s, "---")
	parts := strings.SplitN(rest, "---", 2)
	if len(parts) != 2 {
		return "", s
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func builtinHelpMarkdown() string {
	return `# Oak Pattern Matcher REPL

## Commands
- /lang <language>: set language (e.g., go, javascript)
- /load <file>: load a source file and infer language
- /ast [outFile]: pretty-print Lisp AST (width-aware)
- /raw-ast [outFile]: verbose tree with positions/bytes/flags
- /yaml-ast [outFile]: full YAML dump of the AST
- /pattern <pattern>: run a pattern match against current AST
- /help [topic]: show this help or render a help topic by slug

## Keyboard
- Tab: switch focus
- Ctrl+H: toggle helper cheat sheet
- Up/Down: history or selection
- Enter: submit
- c/y: copy code/text on selected entity
- Ctrl+C: quit
`}

func patternCheatSheet() string {
	return `### Pattern matching cheat sheet (Go)

Try patterns like:
- (function_declaration (name (identifier _)))
- (function_declaration (name (identifier Simple)))
- (parameter_declaration (name (identifier _)) (type (type_identifier string)))
- (return_statement (expression_list (identifier _)))
- (slice_type (element (type_identifier string)))

Tips:
- Use _ as wildcard in symbols
- Use field-pairs: (name (identifier X))
- Nest patterns to match deeper structures
`
}
