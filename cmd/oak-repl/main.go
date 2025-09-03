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
	"github.com/go-go-golems/glazed/pkg/help"
	oakdoc "github.com/go-go-golems/oak/pkg/doc"
	"github.com/go-go-golems/oak/pkg"
	"github.com/go-go-golems/oak/pkg/api"
	pm "github.com/go-go-golems/oak/pkg/patternmatcher"
	tsdump "github.com/go-go-golems/oak/pkg/tree-sitter/dump"
	"github.com/rs/zerolog"
	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
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

	if strings.HasPrefix(in, "/lang ") {
		parts := strings.Fields(in)
		if len(parts) != 2 {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Usage: /lang <language>"}})
			return nil
		}
		e.currentLanguage = parts[1]
		emit(repl.Event{Kind: repl.EventLog, Props: map[string]any{"level": "info", "message": "language set", "fields": map[string]any{"language": e.currentLanguage}}})
		return nil
	}
	if strings.HasPrefix(in, "/load ") {
		parts := strings.Fields(in)
		if len(parts) != 2 {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Usage: /load <file>"}})
			return nil
		}
		path := parts[1]
		b, err := os.ReadFile(path)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		e.currentFile = path
		e.content = b
		// infer language from file extension if not set
		if lang := inferLanguageFromFilename(path); lang != "" {
			e.currentLanguage = lang
		}
		emit(repl.Event{Kind: repl.EventLog, Props: map[string]any{"level": "info", "message": fmt.Sprintf("loaded %s (%d bytes)", path, len(b)), "fields": map[string]any{"language": e.currentLanguage}}})
		return nil
	}
	if strings.HasPrefix(in, "/help") {
		// /help [topic] [--all] [--query DSL]
		parts := strings.Fields(in)
		var topic string
		var showAll bool
		var queryDSL string
		for i := 1; i < len(parts); i++ {
			p := parts[i]
			if p == "--all" {
				showAll = true
				continue
			}
			if strings.HasPrefix(p, "--query=") {
				queryDSL = strings.TrimPrefix(p, "--query=")
				continue
			}
			if p == "--query" && i+1 < len(parts) {
				i++
				queryDSL = parts[i]
				continue
			}
			if !strings.HasPrefix(p, "--") && topic == "" {
				topic = p
			}
		}

		if helpSys != nil {
			if queryDSL != "" {
				// Free-form DSL query
				// For now, show a hint and the built-in help (full DSL execution can be added later)
				emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Query support is available via Glazed CLI. For now, use specific topics or --all."}})
				return nil
			}

			if topic != "" {
				if sec, err := helpSys.GetSectionWithSlug(topic); err == nil && sec != nil {
					md := sec.Content
					if !showAll {
						emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": md}})
						return nil
					}
					// showAll: also include related default sections
					page := sec.DefaultGeneralTopic()
					var b strings.Builder
					b.WriteString(md)
					for _, s := range page {
						if strings.TrimSpace(s.Content) == "" {
							continue
						}
						b.WriteString("\n\n## ")
						b.WriteString(s.Title)
						b.WriteString("\n\n")
						b.WriteString(s.Content)
					}
					emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": b.String()}})
					return nil
				}
			}

			if showAll {
				page := helpSys.GetTopLevelHelpPage()
				var b strings.Builder
				b.WriteString("# Available Help\n\n")
				// General Topics
				if len(page.AllGeneralTopics) > 0 {
					b.WriteString("## General Topics\n\n")
					for _, s := range page.AllGeneralTopics {
						b.WriteString("- ")
						b.WriteString(s.Slug)
						if strings.TrimSpace(s.Title) != "" {
							b.WriteString(" — ")
							b.WriteString(s.Title)
						}
						if strings.TrimSpace(s.Short) != "" {
							b.WriteString("\n  ")
							b.WriteString(s.Short)
						}
						b.WriteString("\n")
					}
					b.WriteString("\n")
				}
				// Examples
				if len(page.AllExamples) > 0 {
					b.WriteString("## Examples\n\n")
					for _, s := range page.AllExamples {
						b.WriteString("- ")
						b.WriteString(s.Slug)
						if strings.TrimSpace(s.Title) != "" {
							b.WriteString(" — ")
							b.WriteString(s.Title)
						}
						if strings.TrimSpace(s.Short) != "" {
							b.WriteString("\n  ")
							b.WriteString(s.Short)
						}
						b.WriteString("\n")
					}
					b.WriteString("\n")
				}
				// Applications
				if len(page.AllApplications) > 0 {
					b.WriteString("## Applications\n\n")
					for _, s := range page.AllApplications {
						b.WriteString("- ")
						b.WriteString(s.Slug)
						if strings.TrimSpace(s.Title) != "" {
							b.WriteString(" — ")
							b.WriteString(s.Title)
						}
						if strings.TrimSpace(s.Short) != "" {
							b.WriteString("\n  ")
							b.WriteString(s.Short)
						}
						b.WriteString("\n")
					}
					b.WriteString("\n")
				}
				// Tutorials
				if len(page.AllTutorials) > 0 {
					b.WriteString("## Tutorials\n\n")
					for _, s := range page.AllTutorials {
						b.WriteString("- ")
						b.WriteString(s.Slug)
						if strings.TrimSpace(s.Title) != "" {
							b.WriteString(" — ")
							b.WriteString(s.Title)
						}
						if strings.TrimSpace(s.Short) != "" {
							b.WriteString("\n  ")
							b.WriteString(s.Short)
						}
						b.WriteString("\n")
					}
					b.WriteString("\n")
				}
				emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": b.String()}})
				return nil
			}
		}

		// Fallback to file system scan then built-in
		if topic != "" {
			if md, err := loadHelpMarkdownBySlug(topic); err == nil && strings.TrimSpace(md) != "" {
				emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": md}})
				return nil
			}
		}
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": builtinHelpMarkdown()}})
		return nil
	}
	if strings.HasPrefix(in, "/ast") {
		// /ast [outFile]
		parts := strings.Fields(in)
		var outFile string
		if len(parts) >= 2 {
			outFile = parts[1]
		}
		if e.currentFile == "" || len(e.content) == 0 {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "No file loaded. Use /load <file> first."}})
			return nil
		}
		if e.currentLanguage == "" {
			if lang := inferLanguageFromFilename(e.currentFile); lang != "" {
				e.currentLanguage = lang
			}
		}
		qb := api.NewQueryBuilder(api.WithLanguage(e.currentLanguage))
		expr, err := qb.ToLispExpression(ctx, e.currentFile, false)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		e.lispAST = expr
		pp := prettyPrintLisp(expr, e.currentLanguage)
		if outFile != "" {
			if err := safeWriteFile(outFile, []byte(pp)); err != nil {
				emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error writing %s: %v", outFile, err)}})
				return nil
			}
			emit(repl.Event{Kind: repl.EventLog, Props: map[string]any{"level": "info", "message": fmt.Sprintf("AST written to %s", outFile)}})
			return nil
		}
		md := "```" + astFenceLang + "\n" + pp + "\n```"
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": md}})
		return nil
	}
	if strings.HasPrefix(in, "/raw-ast") {
		// /raw-ast [outFile]
		parts := strings.Fields(in)
		var outFile string
		if len(parts) >= 2 {
			outFile = parts[1]
		}
		if e.currentFile == "" || len(e.content) == 0 {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "No file loaded. Use /load <file> first."}})
			return nil
		}
		if e.currentLanguage == "" {
			if lang := inferLanguageFromFilename(e.currentFile); lang != "" {
				e.currentLanguage = lang
			}
		}
		lang, err := pkg.LanguageNameToSitterLanguage(e.currentLanguage)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		parser := sitter.NewParser()
		parser.SetLanguage(lang)
		tree, err := parser.ParseCtx(ctx, nil, e.content)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		defer tree.Close()
		var buf bytes.Buffer
		tsdump.DumpVerboseAST(tree.RootNode(), e.content, &buf)
		if outFile != "" {
			if err := safeWriteFile(outFile, buf.Bytes()); err != nil {
				emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error writing %s: %v", outFile, err)}})
				return nil
			}
			emit(repl.Event{Kind: repl.EventLog, Props: map[string]any{"level": "info", "message": fmt.Sprintf("Raw AST written to %s", outFile)}})
			return nil
		}
		md := "```text\n" + buf.String() + "\n```"
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": md}})
		return nil
	}
	if strings.HasPrefix(in, "/yaml-ast") {
		// /yaml-ast [outFile]
		parts := strings.Fields(in)
		var outFile string
		if len(parts) >= 2 {
			outFile = parts[1]
		}
		if e.currentFile == "" || len(e.content) == 0 {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "No file loaded. Use /load <file> first."}})
			return nil
		}
		if e.currentLanguage == "" {
			if lang := inferLanguageFromFilename(e.currentFile); lang != "" {
				e.currentLanguage = lang
			}
		}
		lang, err := pkg.LanguageNameToSitterLanguage(e.currentLanguage)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		parser := sitter.NewParser()
		parser.SetLanguage(lang)
		tree, err := parser.ParseCtx(ctx, nil, e.content)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		defer tree.Close()
		var buf bytes.Buffer
		dumper := tsdump.NewDumper(tsdump.Format("yaml"))
		opts := tsdump.Options{ShowBytes: true, ShowContent: true, ShowAttributes: true, SkipWhitespace: false}
		if err := dumper.Dump(tree, e.content, &buf, opts); err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		if outFile != "" {
			if err := safeWriteFile(outFile, buf.Bytes()); err != nil {
				emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error writing %s: %v", outFile, err)}})
				return nil
			}
			emit(repl.Event{Kind: repl.EventLog, Props: map[string]any{"level": "info", "message": fmt.Sprintf("YAML AST written to %s", outFile)}})
			return nil
		}
		md := "```yaml\n" + buf.String() + "\n```"
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": md}})
		return nil
	}

	if strings.HasPrefix(in, "/pattern") {
		raw := strings.TrimSpace(strings.TrimPrefix(in, "/pattern"))
		if raw == "" {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Usage: /pattern <pattern>"}})
			return nil
		}
		if e.lispAST == nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "No AST; run /ast first"}})
			return nil
		}
		pat, err := pm.Parse(raw)
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

	emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Unknown input. Use /lang, /load, /ast, /raw-ast, /yaml-ast, /pattern"}})
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

	if err := repl.RunTimelineRepl(evaluator, config); err != nil {
		log.Println(err)
		os.Exit(1)
	}
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

func loadHelpMarkdownBySlug(slug string) (string, error) {
	dir := "glazed/pkg/doc/topics"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fm, body := splitFrontMatter(string(b))
		var meta struct{ Slug string `yaml:"Slug"`; Title string `yaml:"Title"` }
		if fm != "" {
			_ = yaml.Unmarshal([]byte(fm), &meta)
		}
		nameSlug := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if strings.EqualFold(meta.Slug, slug) || strings.EqualFold(meta.Title, slug) || strings.EqualFold(nameSlug, slug) {
			return body, nil
		}
	}
	return "", fmt.Errorf("help topic not found: %s", slug)
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
