package main

import (
	"context"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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

func (e *PatternEvaluator) Evaluate(ctx context.Context, code string) (string, error) {
	if len(code) == 0 {
		return "", nil
	}
	if code[0] == '/' {
		var cmd, arg string
		for i := 1; i < len(code); i++ {
			if code[i] == ' ' {
				cmd = code[1:i]
				arg = code[i+1:]
				break
			}
		}
		if cmd == "" {
			cmd = code[1:]
		}
		switch cmd {
		case "load":
			b, err := os.ReadFile(arg)
			if err != nil {
				return "", err
			}
			e.content = b
			e.currentFile = arg
			return fmt.Sprintf("loaded %s (%d bytes)", arg, len(b)), nil
		case "lang":
			e.currentLanguage = arg
			return "language set", nil
		case "ast":
			if e.currentLanguage == "" || e.currentFile == "" {
				return "usage: /lang <lang> then /load <file>", nil
			}
			qb := api.NewQueryBuilder(api.WithLanguage(e.currentLanguage))
			expr, err := qb.ToLispExpression(ctx, e.currentFile, false)
			if err != nil {
				return "", err
			}
			e.lispAST = expr
			return expr.String(), nil
		case "pattern":
			if e.lispAST == nil {
				return "no AST; run /ast first", nil
			}
			pat, err := pm.Parse(arg)
			if err != nil {
				return "", err
			}
			b := pm.PatMatch(pat, e.lispAST, pm.NoBindings)
			if pm.IsFail(b) {
				return "NO MATCH", nil
			}
			return fmt.Sprintf("MATCH %s", b.String()), nil
		default:
			return "unknown command", nil
		}
	}
	return "", nil
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

	model := repl.NewModel(evaluator, config)
	model.SetTheme(repl.BuiltinThemes["dark"])

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}


