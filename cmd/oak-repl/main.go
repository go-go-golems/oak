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

	emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Unknown input. Use /lang, /load, /ast, /raw-ast, /pattern"}})
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

	evaluator := &PatternEvaluator{}
	config := repl.DefaultConfig()
	config.Title = "Oak Pattern Matcher REPL"
	config.Prompt = "oak> "

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

// prettyPrintLisp renders pm.Expression using width-aware formatting and per-language rules.
func prettyPrintLisp(expr pm.Expression, lang string) string {
	var b strings.Builder
	writeExprFit(&b, expr, 0, astMaxWidth, lang)
	return b.String()
}

func writeExprFit(b *strings.Builder, expr pm.Expression, indent, width int, lang string) {
	inline := inlineString(expr)
	avail := width - indent
	if avail <= 0 {
		avail = 1
	}
	if len(inline) <= avail && compactOnly(lang) {
		b.WriteString(inline)
		return
	}
	if cons, ok := expr.(pm.Cons); ok {
		writeListFit(b, cons, indent, width, lang)
		return
	}
	// atom
	b.WriteString(inline)
}

func writeListFit(b *strings.Builder, cons pm.Cons, indent, width int, lang string) {
	elems, dot := collectList(cons)
	if len(elems) == 0 {
		b.WriteString("()")
		return
	}
	innerIndent := indent + 2
	b.WriteString("(")
	firstInline := inlineString(elems[0])
	b.WriteString(firstInline)
	lineLen := 1 + len(firstInline)
	forceBreak := headBreaks(lang, elems[0])
	for i := 1; i < len(elems); i++ {
		el := elems[i]
		elInline := inlineString(el)
		isField := isFieldList(el)
		fieldName := fieldHeadName(el)
		fieldInline := isField && allowInlineField(lang, fieldName)
		if !forceBreak && (fieldInline || (!isField)) && lineLen+1+len(elInline) <= width {
			b.WriteString(" ")
			b.WriteString(elInline)
			lineLen += 1 + len(elInline)
			continue
		}
		b.WriteString("\n")
		b.WriteString(strings.Repeat(" ", innerIndent))
		writeExprFit(b, el, innerIndent, width, lang)
		lineLen = width
	}
	if dot != nil {
		dotInline := inlineString(dot)
		if !forceBreak && lineLen+3+len(dotInline) <= width {
			b.WriteString(" . ")
			b.WriteString(dotInline)
		} else {
			b.WriteString("\n")
			b.WriteString(strings.Repeat(" ", innerIndent))
			b.WriteString(". ")
			writeExprFit(b, dot, innerIndent+2, width, lang)
		}
	}
	b.WriteString(")")
}

func inlineString(expr pm.Expression) string {
	if cons, ok := expr.(pm.Cons); ok {
		elems, dot := collectList(cons)
		parts := make([]string, 0, len(elems))
		for _, el := range elems {
			parts = append(parts, inlineString(el))
		}
		s := "(" + strings.Join(parts, " ")
		if dot != nil {
			s += " . " + inlineString(dot)
		}
		s += ")"
		return s
	}
	return fmt.Sprint(expr)
}

func collectList(expr pm.Cons) ([]pm.Expression, pm.Expression) {
	var elems []pm.Expression
	var tail pm.Expression
	cur := pm.Expression(expr)
	for {
		c, ok := cur.(pm.Cons)
		if !ok {
			tail = cur
			break
		}
		elems = append(elems, c.Car)
		cur = c.Cdr
	}
	return elems, tail
}

func headBreaks(lang string, head pm.Expression) bool {
	m := astBreakHeads[lang]
	if m == nil {
		return false
	}
	if s, ok := head.(pm.Symbol); ok {
		_, yes := m[s.Name]
		return yes
	}
	return false
}

func isFieldList(expr pm.Expression) bool {
	c, ok := expr.(pm.Cons)
	if !ok {
		return false
	}
	elems, _ := collectList(c)
	if len(elems) == 0 {
		return false
	}
	if s, ok := elems[0].(pm.Symbol); ok {
		return s.Name != ""
	}
	return false
}

func fieldHeadName(expr pm.Expression) string {
	c, ok := expr.(pm.Cons)
	if !ok {
		return ""
	}
	elems, _ := collectList(c)
	if len(elems) == 0 {
		return ""
	}
	if s, ok := elems[0].(pm.Symbol); ok {
		return s.Name
	}
	return ""
}

func allowInlineField(lang, field string) bool {
	m := astInlineFields[lang]
	if m == nil {
		return false
	}
	_, ok := m[field]
	return ok
}

func compactOnly(lang string) bool {
	_, ok := astCompactLangs[lang]
	return ok
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
