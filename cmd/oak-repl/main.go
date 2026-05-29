package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/bobatea/pkg/eventbus"
	"github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/oak/pkg/api"
	pm "github.com/go-go-golems/oak/pkg/patternmatcher"
)

type PatternEvaluator struct {
	currentFile     string
	currentLanguage string
	content         []byte
	lispAST         pm.Expression
}

func (e *PatternEvaluator) EvaluateStream(ctx context.Context, code string, emit func(repl.Event)) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}

	output, err := e.evaluateCommand(ctx, code)
	if output != "" || err != nil {
		if err != nil {
			emit(repl.Event{Kind: repl.EventStderr, Props: map[string]any{"text": output, "error": err.Error(), "is_error": true}})
			return nil
		}
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"text": output, "markdown": output}})
	}
	return nil
}

func (e *PatternEvaluator) evaluateCommand(ctx context.Context, code string) (string, error) {
	cmd, rawArgs, _ := strings.Cut(code, " ")
	cmd = strings.TrimPrefix(cmd, "/")
	args := strings.Fields(rawArgs)

	switch cmd {
	case "lang":
		if len(args) != 1 {
			return "usage: /lang <language>", fmt.Errorf("invalid usage")
		}
		e.currentLanguage = args[0]
		return "language set", nil
	case "load":
		if len(args) != 1 {
			return "usage: /load <file>", fmt.Errorf("invalid usage")
		}
		b, err := os.ReadFile(args[0])
		if err != nil {
			return err.Error(), err
		}
		e.currentFile = args[0]
		e.content = b
		return fmt.Sprintf("loaded %s (%d bytes)", args[0], len(b)), nil
	case "ast":
		if e.currentLanguage == "" || e.currentFile == "" {
			return "usage: /lang <lang> then /load <file>", fmt.Errorf("missing context")
		}
		qb := api.NewQueryBuilder(api.WithLanguage(e.currentLanguage))
		expr, err := qb.ToLispExpression(ctx, e.currentFile, false)
		if err != nil {
			return err.Error(), err
		}
		e.lispAST = expr
		return expr.String(), nil
	case "pattern":
		if e.lispAST == nil {
			return "no AST; run /ast first", fmt.Errorf("no ast")
		}
		patternStr := strings.TrimSpace(rawArgs)
		pat, err := pm.Parse(patternStr)
		if err != nil {
			return err.Error(), err
		}
		matches := collectMatches(pat, e.lispAST)
		if len(matches) == 0 {
			return "NO MATCH", nil
		}
		out := fmt.Sprintf("matches: %d\n", len(matches))
		for i, b := range matches {
			if pm.IsFail(b) {
				continue
			}
			out += fmt.Sprintf("%d) %s\n", i+1, b.String())
		}
		return out, nil
	default:
		return "", nil
	}
}

func (e *PatternEvaluator) GetPrompt() string        { return "oak-pattern> " }
func (e *PatternEvaluator) GetName() string          { return "Oak Pattern Matcher" }
func (e *PatternEvaluator) SupportsMultiline() bool  { return true }
func (e *PatternEvaluator) GetFileExtension() string { return ".pattern" }

func main() {
	evaluator := &PatternEvaluator{}
	config := repl.DefaultConfig()
	config.Title = "Oak Pattern Matcher REPL"
	config.Prompt = "oak> "

	bus, err := eventbus.NewInMemoryBus()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	model := repl.NewModel(evaluator, config, bus.Publisher)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
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
