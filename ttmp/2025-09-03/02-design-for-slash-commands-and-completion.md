Title: Design for a reusable slash-command registry and completion API for the REPL
Date: 2025-09-03
Author: manuel (wesen)

## 1) Purpose and scope

We want a backend-agnostic, reusable slash-command system for the timeline-first REPL: programs using `bobatea/pkg/repl` can register `/commands` declaratively, get a built-in dispatcher to execute them (instead of hand-parsing in `EvaluateStream`), and benefit from argument/flag tab-completion in the input line.

Goals:
- Simple API for third parties to register commands: name, summary, args/flags schema, handler, and completer hooks.
- Bobatea handles parsing, dispatch, and UI integration, not the app-level evaluator.
- Completions for command names and arguments while typing (non-intrusive to existing REPL UX).
- No dependency on any particular backend (e.g., Glazed); adapters can register their own commands.

Non-goals (now):
- Full programmable shell with pipes and redirections.
- Deep integration with Cobra flag parser (we’ll provide a small, REPL-focused parser tailored to completion).

## 2) Exploration and current state

- Input flow lives in `bobatea/pkg/repl/model.go`:
  - `updateInput` collects key events and calls `submit(input)` on Enter.
  - On submit, it publishes an input entity, then calls `evaluator.EvaluateStream(ctx, code, emit)`.
  - Tab currently switches focus between "input" and "timeline".
- We already have an `autocomplete` widget (`bobatea/pkg/autocomplete`) that can produce suggestions with a `Completioner func(ctx, query) ([]Suggestion, error)`. It’s not wired into the REPL input line yet.
- Apps (like `oak-repl`) manually parse `/help` inside `EvaluateStream`. This should be replaced by a common dispatcher.

Implications:
- We’ll intercept inputs starting with `/` before delegating to `Evaluator`.
- We’ll add UI affordances to show suggestions when typing `/...` without breaking existing focus switching. We can bind completion to a different key (e.g., `Ctrl+Space`) by default, and allow Tab to be re-bound in “command mode”.

## 3) Feature set

### 3.1 Command registration

- Register by name (e.g., `"help"`) with:
  - Summary, long help/usage (optional)
  - Argument/flag schema with types and defaults
  - Handler function that receives parsed args/flags and an `emit` for timeline events
  - Completer hooks for dynamic suggestions per argument position and flag values
  - Optional category/tags for future palette integrations

### 3.2 Parsing and syntax

- Input form: `/name [args...] [--flag[=value]] ...` with standard quoting rules (single/double quotes, backslash escapes).
- Flags support `--flag value` and `--flag=value`. Boolean flags can be `--flag`/`--no-flag`.
- Arg schema supports positional args with arity (`required`, `optional`, `variadic`).
- Minimal parser tailored for completion: emits a token stream + AST with cursor location; avoids heavy libraries.

### 3.3 Completion behavior

- When the input starts with `/`:
  - If only `"/na"` typed → suggest command names (prefix/fuzzy) with descriptions.
  - After command name, use parser state + schema to decide what to complete (next positional, a `--flag`, or a flag value).
  - Dynamic completion via per-arg or per-flag `Completioner` function.
- Keybinding:
  - Default: `Ctrl+Space` toggles completion popup
  - Optional: when in “command mode” (input starts with `/`) allow Tab to trigger completion instead of focus switching (configurable).

### 3.4 Dispatching and timeline integration

- On Enter, if input starts with `/`, the dispatcher:
  - Parses input, validates schema
  - If errors: render a markdown usage block as a REPL result entity
  - If ok: call handler with a context and an `emit(Event)` function so handlers can stream logs and markdown like evaluators do
- If input doesn’t start with `/`, fall back to `Evaluator.EvaluateStream`.

### 3.5 Discoverability and help

- Built-in `/help` for the slash system itself:
  - `/help` → lists registered commands
  - `/help <name>` → shows usage, args, flags, examples for that command
- Backend-specific help (Glazed) stays a separate command (already implemented) but can be registered through this system.

## 4) Architecture and files

New package under bobatea:

1) `bobatea/pkg/repl/slash/api.go`
   - `type ArgType string` (string, number, bool, enum, file, dir, slug, …)
   - `type ArgSpec struct { Name string; Type ArgType; Required bool; Variadic bool; Enum []string; Description string }`
   - `type FlagSpec struct { Name string; Type ArgType; Default any; Enum []string; Description string; Negatable bool }`
   - `type Schema struct { Positionals []ArgSpec; Flags []FlagSpec }`
   - `type Handler func(ctx context.Context, in Input, out Emitter) error`
   - `type Completer func(ctx context.Context, state CompletionState) ([]autocomplete.Suggestion, error)`
   - `type Command struct { Name, Summary, Usage string; Schema Schema; Run Handler; Complete Completer }`
   - `type Registry interface { Register(*Command) error; Unregister(name string); Get(name string) *Command; List() []*Command }`
   - `type Dispatcher interface { TryHandle(ctx, input string, emit func(repl.Event)) (handled bool) }`

2) `bobatea/pkg/repl/slash/registry.go`
   - In-memory `Registry` with thread-safety
   - `NewRegistry()` returns a registry
   - `NewDispatcher(reg Registry, opts ...) Dispatcher` – returns a dispatcher that:
     - Checks prefix `/`, parses command, finds a registered handler
     - Invokes handler with parsed args/flags
     - Emits markdown errors on parse/validation problems

3) `bobatea/pkg/repl/slash/parser.go`
   - Tokenizer: supports quotes and escapes; reports token spans and cursor-relative state
   - Parser: builds a simple structure `{ Name string; Positionals []string; Flags map[string][]string }`
   - Validation against `Schema`
   - Cursor-aware `CompletionState { Name string; Phase enum{Name,Positional,Flag,FlagValue}; Index int; Partial string; ParsedSoFar … }`

4) `bobatea/pkg/repl/slash/completion.go`
   - Bridge parser `CompletionState` to registered command’s `Completer`
   - Built-ins for common types (filesystem, enum, boolean, commands list)
   - Aggregation of suggestions with highlighting spans

5) `bobatea/pkg/repl/slash/ui.go`
   - Integrate into REPL model:
     - Hook in `updateInput`: when input starts with `/`, optionally intercept Tab (config) or `Ctrl+Space` to show autocomplete popup using `autocomplete.Model`
     - Drive the popup using a `Completioner` that delegates to dispatcher’s completion engine
     - Dismiss on Enter/Escape; persist selection into the input buffer

6) REPL wiring (non-breaking):
   - `bobatea/pkg/repl/model.go`
     - Before calling `Evaluator.EvaluateStream` in `submit`, call `dispatcher.TryHandle` and return if handled
     - Provide configuration on `Config` to toggle slash system and keybindings

7) Optional: bridge helpers
   - `bobatea/pkg/repl/help/register_slash.go` – registers the previously implemented help backend command into the registry

## 5) Key APIs (signatures)

```go
// slash/api.go
type ArgType string
const (
    ArgString ArgType = "string"
    ArgNumber ArgType = "number"
    ArgBool   ArgType = "bool"
    ArgEnum   ArgType = "enum"
    ArgFile   ArgType = "file"
    ArgDir    ArgType = "dir"
)

type ArgSpec struct {
    Name        string
    Type        ArgType
    Required    bool
    Variadic    bool
    Enum        []string
    Description string
}

type FlagSpec struct {
    Name        string
    Type        ArgType
    Default     any
    Enum        []string
    Description string
    Negatable   bool // supports --no-flag
}

type Schema struct {
    Positionals []ArgSpec
    Flags       []FlagSpec
}

type Input struct {
    Raw        string
    Name       string
    Positionals []string
    Flags       map[string][]string
}

type Emitter func(repl.Event)

type Handler func(ctx context.Context, in Input, out Emitter) error

type CompletionPhase int
const (
    PhaseName CompletionPhase = iota
    PhasePositional
    PhaseFlag
    PhaseFlagValue
)

type CompletionState struct {
    Raw             string
    Caret           int
    Phase           CompletionPhase
    Name            string
    PositionalIndex int // for PhasePositional
    CurrentFlag     string // for PhaseFlagValue
    Partial         string // current token under cursor
    Parsed          Input  // best-effort parse so far
}

type Completer func(ctx context.Context, st CompletionState) ([]autocomplete.Suggestion, error)

type Command struct {
    Name     string
    Summary  string
    Usage    string
    Schema   Schema
    Run      Handler
    Complete Completer // optional
}

type Registry interface {
    Register(cmd *Command) error
    Unregister(name string)
    Get(name string) *Command
    List() []*Command
}

type Dispatcher interface {
    TryHandle(ctx context.Context, input string, emit func(repl.Event)) bool
    Complete(ctx context.Context, raw string, caret int) ([]autocomplete.Suggestion, error)
}
```

## 6) UI/UX details

- While typing `/` commands, a subtle hint appears: “Command mode: Ctrl+Space to complete”.
- Completion popup shows:
  - Command suggestions with summary
  - When inside args/flags, shows value suggestions with highlights
  - Selecting a suggestion inserts or replaces the token under the cursor, respecting quotes
- Keymap (configurable):
  - `Ctrl+Space`: toggle completion popup
  - `Tab`: in command mode, accept top suggestion or move to next slot (optional)
  - `Esc`: close popup

## 7) Examples

Registering a command:

```go
reg := slash.NewRegistry()
disp := slash.NewDispatcher(reg)

reg.Register(&slash.Command{
    Name:    "help",
    Summary: "Show help for commands",
    Usage:   "/help [name] [--all] [--query=DSL]",
    Schema: slash.Schema{
        Positionals: []slash.ArgSpec{{Name: "name", Type: slash.ArgString}},
        Flags: []slash.FlagSpec{
            {Name: "all", Type: slash.ArgBool},
            {Name: "query", Type: slash.ArgString},
        },
    },
    Run: func(ctx context.Context, in slash.Input, emit slash.Emitter) error {
        md := replhelp.HandleHelpCommand(ctx, cfg, strings.TrimSpace(in.Raw))
        emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": md}})
        return nil
    },
})
```

Integrating into REPL submit:

```go
// inside bobatea/pkg/repl/model.go submit()
if strings.HasPrefix(code, "/") {
    if m.slashDispatcher.TryHandle(context.Background(), code, func(e repl.Event) { _ = m.publishReplEvent(turnID, e) }) {
        return nil
    }
}
// fallback to evaluator
_ = m.evaluator.EvaluateStream(ctx, code, func(e Event) { _ = m.publishReplEvent(turnID, e) })
```

Wiring completion:

```go
// in updateInput, when Ctrl+Space pressed and input starts with "/":
suggestions, _ := m.slashDispatcher.Complete(context.Background(), m.textInput.Value(), m.textInput.Cursor())
// show suggestions via autocomplete.Model
```

## 8) Edge cases and error handling

- Unknown command → show a short suggestion list of closest commands.
- Parse errors → render a concise error + usage for that command.
- Ambiguous or missing required args → usage with highlighted missing parts.
- Very large completion sets → paginate/truncate and allow narrowing.

## 9) Migration plan

- Introduce the slash package and dispatcher (no breaking changes).
- Integrate into REPL model with a config toggle (defaults on).
- Register the help backend command through the registry.
- Update `oak-repl` to remove ad hoc `/help` parsing – already partially done; will be moved to registered slash command.

## 10) Deliverables (files and functions)

- bobatea/pkg/repl/slash/api.go – core types and interfaces
- bobatea/pkg/repl/slash/registry.go – registry + dispatcher
- bobatea/pkg/repl/slash/parser.go – tokenizer, parser, cursor-aware state
- bobatea/pkg/repl/slash/completion.go – completion engine
- bobatea/pkg/repl/slash/ui.go – REPL input integration (keybinding + autocomplete.Model wiring)
- bobatea/pkg/repl/help/register_slash.go – bridge to register the help command

This system centralizes `/` command handling and completions in bobatea so apps don’t implement custom `/command` parsing inside their evaluators, while remaining backend-agnostic and extensible.


