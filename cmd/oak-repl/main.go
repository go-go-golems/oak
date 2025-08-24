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
	// Regular input (non-slash) not used for now
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

	// /lang <language>
	model.AddCustomCommand("lang", func(args []string) tea.Cmd {
		return func() tea.Msg {
			if len(args) != 1 {
				return repl.EvaluationCompleteMsg{Input: "/lang", Output: "usage: /lang <language>", Error: fmt.Errorf("invalid usage")}
			}
			evaluator.currentLanguage = args[0]
			return repl.EvaluationCompleteMsg{Input: "/lang " + args[0], Output: "language set", Error: nil}
		}
	})

	// /load <file>
	model.AddCustomCommand("load", func(args []string) tea.Cmd {
		return func() tea.Msg {
			if len(args) != 1 {
				return repl.EvaluationCompleteMsg{Input: "/load", Output: "usage: /load <file>", Error: fmt.Errorf("invalid usage")}
			}
			b, err := os.ReadFile(args[0])
			if err != nil {
				return repl.EvaluationCompleteMsg{Input: "/load " + args[0], Output: err.Error(), Error: err}
			}
			evaluator.currentFile = args[0]
			evaluator.content = b
			return repl.EvaluationCompleteMsg{Input: "/load " + args[0], Output: fmt.Sprintf("loaded %s (%d bytes)", args[0], len(b)), Error: nil}
		}
	})

	// /ast
	model.AddCustomCommand("ast", func(args []string) tea.Cmd {
		return func() tea.Msg {
			if evaluator.currentLanguage == "" || evaluator.currentFile == "" {
				return repl.EvaluationCompleteMsg{Input: "/ast", Output: "usage: /lang <lang> then /load <file>", Error: fmt.Errorf("missing context")}
			}
			qb := api.NewQueryBuilder(api.WithLanguage(evaluator.currentLanguage))
			expr, err := qb.ToLispExpression(context.Background(), evaluator.currentFile, false)
			if err != nil {
				return repl.EvaluationCompleteMsg{Input: "/ast", Output: err.Error(), Error: err}
			}
			evaluator.lispAST = expr
			return repl.EvaluationCompleteMsg{Input: "/ast", Output: expr.String(), Error: nil}
		}
	})

	// /pattern <pattern>
	model.AddCustomCommandRaw("pattern", func(raw string, args []string) tea.Cmd {
		return func() tea.Msg {
			if evaluator.lispAST == nil {
				return repl.EvaluationCompleteMsg{Input: "/pattern", Output: "no AST; run /ast first", Error: fmt.Errorf("no ast")}
			}
			patternStr := raw
			pat, err := pm.Parse(patternStr)
			if err != nil {
				return repl.EvaluationCompleteMsg{Input: "/pattern", Output: err.Error(), Error: err}
			}
			b := pm.PatMatch(pat, evaluator.lispAST, pm.NoBindings)
			if pm.IsFail(b) {
				return repl.EvaluationCompleteMsg{Input: "/pattern", Output: "NO MATCH", Error: nil}
			}
			return repl.EvaluationCompleteMsg{Input: "/pattern", Output: "MATCH " + b.String(), Error: nil}
		}
	})

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}


