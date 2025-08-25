# Pattern Matcher + Tree-sitter: Getting Started

This guide shows how to use Oak's new PAIP-style pattern matcher on top of Tree-sitter ASTs. You'll learn to:

- Dump ASTs in multiple formats
- Convert ASTs to Lisp S-expressions
- Run expressive PAIP patterns via CLI
- Explore interactively with the REPL

## Prerequisites

- Go 1.24+
- Oak repository checked out (this project)

## Quick Tour

### 1) AST inspection (CLI)

Use `ast` to print ASTs in different formats:

- Lisp S-expression (pattern-friendly)
```bash
go run ./cmd/oak ast --language go --format lisp ./test-inputs/test.go | head -n 10
```

- Verbose text with positions and flags
```bash
go run ./cmd/oak ast --language go --format verbose ./test-inputs/test.go | head -n 30
```

- Other formats: `text`, `json`, `yaml`, `xml`
```bash
go run ./cmd/oak ast --language go --format json ./test-inputs/test.go | jq '.' | head -n 20
```

Flags:
- `--language <go|typescript|...>`: required
- `--format <lisp|verbose|text|json|yaml|xml>`: default `lisp`
- `--include-anonymous`: include anonymous nodes in Lisp output

### 2) Run PAIP patterns (CLI)

The `pattern` command runs PAIP patterns anywhere in the converted AST:

```bash
# Find all (name ...) pairs (works broadly across node types with a name field)
go run ./cmd/oak pattern --language go --pattern "(name ?n)" ./test-inputs/test.go | head -n 20
```

Example output:
```
=== /path/to/test.go (matches: 19) ===
1) {?n: (type_identifier MyStruct)}
2) {?n: (field_identifier Name)}
...
```

More sophisticated patterns:

- Find function declarations with names:
```bash
go run ./cmd/oak pattern --language go --pattern "(function_declaration (name ?n))" ./test-inputs/test.go
```

- Find identifiers in ALL-CAPS (using simple structure + post-filtering is possible via scripting; predicates can be extended later):
```bash
# Structure-level match first
go run ./cmd/oak pattern --language go --pattern "(identifier ?id)" ./test-inputs/test.go
```

Flags:
- `--language <lang>`: required
- `--pattern '<paip-pattern>'` or `--pattern-file file.pattern`
- `--include-anonymous`: include anonymous nodes in Lisp AST

Notes:
- Patterns use Lisp-like syntax with variables `?x`, logical ops `?and`, `?or`, `?not`, and predicates via `?is`.
- Current built-in predicates: `numberp`, `symbolp`, `atomp`, `oddp`, `evenp`.

### 3) Interactive REPL

Start the REPL to iterate on patterns quickly:

```bash
go run ./cmd/oak-repl
```

Once inside, use these slash commands:

- `/lang <language>`: set language (e.g., `/lang go`)
- `/load <file>`: load a source file (absolute or relative path)
- `/ast`: show current AST in Lisp form
- `/pattern <pattern>`: run a PAIP pattern against the current Lisp AST

Example session:
```
oak> /lang go
oak> /load ./test-inputs/test.go
oak> /ast
(source_file (package_clause ...) ...)
oak> /pattern (name ?n)
MATCH {?n: (identifier main)}
```

Tips:
- Use `Ctrl+J` for multiline mode. Enter on an empty line executes.
- `Ctrl+E` opens the current input in your `$EDITOR`.
- `/help` shows built-in REPL commands.

## Writing Patterns

PAIP patterns operate over the Lisp-ified AST. A few building blocks:

- Variables: `?x`, `?name`
- Exact symbols: `identifier`, `function_declaration`, `name`
- Lists: `(function_declaration (name ?n))`
- Logical: `(?and p1 p2)`, `(?or p1 p2)`, `(?not p)`
- Predicates: `(?is ?x numberp)`
- Segments (framework ready): `?*`, `?+`, `??` (sequence matching)

Examples:

- All function names:
```lisp
(function_declaration (name ?n))
```

- Any node with a `name` field:
```lisp
(name ?n)
```

- Name plus body presence:
```lisp
(function_declaration (name ?n) (body ?b))
```

- Logical combinations:
```lisp
(?and (function_declaration (name ?n)) (?not (result)))
```

## Programmatic API

Convert a file to a Lisp expression and evaluate a pattern in Go:

```go
qb := api.NewQueryBuilder(api.WithLanguage("go"))
expr, err := qb.ToLispExpression(ctx, "/path/to/file.go", false)
if err != nil { /* handle */ }

pat, _ := patternmatcher.Parse("(name ?n)")
b := patternmatcher.PatMatch(pat, expr, patternmatcher.NoBindings)
if !patternmatcher.IsFail(b) {
	fmt.Println("MATCH:", b)
}
```

## Troubleshooting

- No matches with structure-heavy patterns? Start by inspecting the Lisp AST with:
  ```bash
  go run ./cmd/oak ast --language go --format lisp ./yourfile.go | less
  ```
- The REPL shows "usage: /lang <lang> then /load <file>" when context missing; set both.
- For large files, prefer CLI for performance; REPL is best for iteration.

## Roadmap / Extensibility

- Add domain predicates (e.g., `identifier-screaming-snake-p`, `jsx-element-p`).
- Segment patterns over tree contexts for repetitive structures.
- Multi-file and cross-capture constraints.

Happy matching!


