package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-go-golems/bobatea/pkg/logutil"
	"github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/oak/pkg/api"
	pm "github.com/go-go-golems/oak/pkg/patternmatcher"
	"github.com/rs/zerolog"
)

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
		emit(repl.Event{Kind: repl.EventLog, Props: map[string]any{"level": "info", "message": fmt.Sprintf("loaded %s (%d bytes)", path, len(b))}})
		return nil
	}
	if in == "/ast" {
		if e.currentLanguage == "" || e.currentFile == "" {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Usage: /lang <lang> then /load <file>"}})
			return nil
		}
		qb := api.NewQueryBuilder(api.WithLanguage(e.currentLanguage))
		expr, err := qb.ToLispExpression(ctx, e.currentFile, false)
		if err != nil {
			emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
			return nil
		}
		e.lispAST = expr
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": expr.String()}})
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
		out := fmt.Sprintf("matches: %d\n", len(matches))
		for i, b := range matches {
			if pm.IsFail(b) {
				continue
			}
			out += fmt.Sprintf("%d) %s\n", i+1, b.String())
		}
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": out}})
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
		out := fmt.Sprintf("matches: %d\n", len(matches))
		for i, b := range matches {
			if pm.IsFail(b) {
				continue
			}
			out += fmt.Sprintf("%d) %s\n", i+1, b.String())
		}
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": out}})
		return nil
	}

	emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": "Unknown input. Use /lang, /load, /ast, /pattern"}})
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
	// CLI flags for logging
	ll := flag.String("log-level", "error", "log level: trace, debug, info, warn, error")
	lf := flag.String("log-file", "", "log file path (optional)")
	flag.Parse()

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
