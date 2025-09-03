Title: Design for a reusable REPL help command
Date: 2025-09-03
Author: manuel (wesen)

## 1) Purpose and scope

We want a first-class, reusable REPL Help feature that integrates with pluggable help backends in timeline-first REPLs. It must not require a specific `HelpSystem` implementation. The goal is to:

- Expose help sections and metadata within any Evaluator-driven REPL.
- Provide consistent slash-commands and UX across different REPLs (oak-repl and general bobatea-based REPLs).
- Support both simple topic lookup (slug) and `--all` top-level listings as mandatory capabilities; optionally support a query DSL if the backend provides it.
- Be embeddable and configurable, not tied specifically to oak-repl or Glazed.

This document proposes the features, command syntax, a backend abstraction, adapters, and integration points. It also outlines filenames, function names, and architecture for a small library inside bobatea to power REPL help.

## 2) Current state (findings)

Relevant existing pieces:

- `glazed/pkg/help` already provides (useful for an adapter, but not a required dependency):
  - Storage via `store.Store` with in-memory backend.
  - Loading sections from FS and from Markdown with frontmatter (`LoadSectionFromMarkdown`).
  - Query interfaces: `SectionQuery` (structured builder) and `HelpSystem.QuerySections(query string)` (DSL + legacy fallback). See `glazed/pkg/help/dsl_bridge.go`, `glazed/pkg/help/query.go`.
  - Convenience page assembly: `HelpSystem.GetTopLevelHelpPage()` returning `HelpPage` grouped by types and visibility.
- `oak/cmd/oak-repl/main.go` currently implements `/help` inline, with partial integration:
  - It initializes `helpSys = help.NewHelpSystem()` and `oakdoc.AddDocToHelpSystem(helpSys)`.
  - It implements `/help [topic] [--all] [--query DSL]` but currently only supports:
    - `topic` lookup via `GetSectionWithSlug`.
    - `--all` to show top-level listings (`GetTopLevelHelpPage`).
    - `--query` prints a hint instead of executing queries inside the REPL.
  - It falls back to reading raw markdown files from `glazed/pkg/doc/topics` if a slug is not found in the store.
- bobatea REPL model streams markdown and logs via `EvaluateStream(ctx, code, emit)`, so help output should be rendered as markdown and optionally structured log events.

Conclusion: the Glazed help engine is mature and can be consumed via an adapter, but the `/help` REPL handler should target a generic backend interface. We want a reusable module that:
- Parses `/help` inputs
- Talks to an abstract `Backend` (slug, top-level required; DSL query optional)
- Renders results consistently (top level, specific section, related sections)
- Can be dropped into any Evaluator

### 2.1 Backend abstraction requirements (updated)

We will not depend on Glazed types in the REPL layer. Instead define minimal data contracts and a backend interface:

Types:

```go
// bobatea/pkg/repl/help/api.go
type Section struct {
    Slug           string
    Title          string
    Short          string
    Content        string
    Type           string   // "topic" | "example" | "application" | "tutorial" (extensible)
    Topics         []string
    Flags          []string
    Commands       []string
    ShowPerDefault bool
    Order          int
}

type TopLevelPage struct {
    AllGeneralTopics []*Section
    AllExamples      []*Section
    AllApplications  []*Section
    AllTutorials     []*Section
}

type Backend interface {
    // Mandatory
    TopLevel(ctx context.Context) (*TopLevelPage, error)
    GetBySlug(ctx context.Context, slug string) (*Section, error)

    // Optional DSL support. If not supported, ok=false and/or ErrNotSupported.
    Query(ctx context.Context, dsl string) (results []*Section, ok bool, err error)
}
```

Notes:
- No backward compatibility required: the REPL code will only target this interface.
- A Glazed adapter can populate these structures and map calls to `HelpSystem`.

## 3) Desired REPL help features

### 3.1 Slash commands

All commands are intended to be reusable across REPLs. Oak-specific REPLs can add extra commands (e.g., AST docs), but the following should be generic:

- `/help` – Show top-level help page (default curated listing) using `GetTopLevelHelpPage()`.
- `/help <slug>` – Show a specific section by slug. Optionally append related default sections of different types.
- `/help --all` – Show a comprehensive top-level listing (all types, all curated entries).
- `/help --query "<DSL>"` – Run the full help DSL query and render matching sections if the backend supports queries; otherwise show a helpful message.
- `/help --type=<type>` – Filter top-level listing by type (`topic|example|application|tutorial`). Convenience for beginners; internally could translate to DSL or SectionQuery.
- `/help --topic=<tag>` / `--flag=<flag>` / `--command=<name>` – Simple filters for discovery (translates to DSL: `topic:...`, `flag:...`, `command:...`).
- `/help --search "text"` – Full-text search (quoted) convenience; maps to `"text"` DSL.

Notes:
- Flags can be combined, and if DSL is present (`--query`), we prefer the raw DSL.
- All outputs are markdown; we can render section content and brief metadata.

### 3.2 Result rendering

We standardize output renderers:

- Top-level page:
  - Title `# Available Help` then grouped sections by type with `Slug — Title` on one line and optional `Short` on the next line.
  - Same style as current oak-repl implementation, moved to a reusable function.
- Single section page:
  - Render the `section.Content` directly (markdown).
  - Optional related sections: default topics/examples/applications/tutorials associated with the section’s slug (using `DefaultGeneralTopic`, `DefaultExamples`, etc.). When shown, prepend `## {Type Plural}` then list titles + navigate hints.
- Query results:
  - `# Help Results` then list matching sections with a compact summary and a `To view: /help <slug>` hint.
  - If only one result and it’s a `GeneralTopic`, render content directly unless `--list` is specified.

### 3.3 Query handling

We support two layers:

1) Pass-through DSL via `Backend.Query(ctx, query)` as the primary interface whenever supported by the backend. If not supported, return an informative message and suggest slug or `--all`.
2) Convenience flag-to-DSL conversion:
   - `--type=tutorial` → `type:tutorial`
   - `--topic=x` → `topic:x` (multiple become `OR`-joined)
   - `--flag=--output` → `flag:--output`
   - `--command=json` → `command:json`
   - `--search "foo"` → `"foo"`
   - Combine with `AND` when multiple different fields are present; `OR` within the same field repeated.

Fallback: if no DSL and a single non-flag arg is present, treat it as `<slug>` lookup.

### 3.4 Extensibility hooks

- Pluggable renderer callbacks for:
  - Top-level page
  - Single section
  - Query results list
  These receive the raw `help.Section` slices to allow bespoke formatting.

- Optional context source for dynamic filtering: callers can provide `contextProvider func() (command string, flags []string)` so that `/help` with no args shows context-aware suggestions by composing a query from the current command and active flags (similar to `BuildContextualHelp` in the docs).

### 3.5 Error handling and ergonomics

- For invalid DSL, return a markdown error block and a quick primer link: `glaze help simple-query-dsl`.
- For unknown slug, explain it wasn’t found and suggest `/help --query "text"` search.
- Don’t crash; always produce a helpful page.

## 4) Architecture and files

Proposed new reusable module under bobatea (generic) with a small oak wiring:

1) Generic REPL help utilities (new):
   - File: `bobatea/pkg/repl/help/api.go`
     - `type Section`, `type TopLevelPage`, `type Backend` (as above)

   - File: `bobatea/pkg/repl/help/handler.go`
     - `type Config struct { Backend Backend; ShowRelated bool; Renderer Renderer; ContextProvider func() (string, []string) }`
     - `type Renderer interface { RenderTopLevel(page *TopLevelPage) string; RenderSection(section *Section, related map[string][]*Section) string; RenderQueryResults(results []*Section) string }`
     - `func DefaultRenderer() Renderer`
     - `func HandleHelpCommand(ctx context.Context, cfg Config, input string) (markdown string)`
       - Parses `/help` args/flags
       - Builds DSL or slug request
       - Calls into `cfg.Backend` methods
       - Uses `cfg.Renderer` to produce markdown

   - File: `bobatea/pkg/repl/help/parse.go`
     - `type HelpArgs struct { Slug string; ShowAll bool; Query string; Types []string; Topics []string; Flags []string; Commands []string; Search string; ListOnly bool }`
     - `func ParseHelpInput(raw string) (HelpArgs, error)`
     - `func BuildDSL(args HelpArgs) (string, bool)` — returns DSL and whether it’s valid/used.

   - File: `bobatea/pkg/repl/help/render.go`
     - Implements `DefaultRenderer` rendering rules described in 3.2 with generic types.

   - File: `bobatea/pkg/repl/help/adapters/glazed.go`
     - Adapter from Glazed to `Backend`:
       - `type GlazedBackend struct { HS *help.HelpSystem }`
       - Implements `TopLevel`, `GetBySlug`, `Query` (mapping Glazed types to generic ones)

2) Oak REPL integration (wiring only):
   - File: `oak/cmd/oak-repl/help_integration.go`
     - `func NewREPLHelp(cfg replhelp.Config) func(ctx context.Context, code string, emit func(repl.Event)) bool`
       - Returns a handler closure that can be called early in `EvaluateStream` to intercept `/help` input; if it handled, returns true after emitting markdown via `emit`.
     - Replace the inlined `/help` branch in `main.go` with a call to this handler.
   - Update `main.go`: construct a backend (e.g., `GlazedBackend{HS: helpSys}`); wire config into `NewREPLHelp` with `ShowRelated=true` and optional `ContextProvider`.

3) Optional helper for general apps:
   - File: `bobatea/pkg/repl/help/middleware.go`
     - `func InterceptHelp(e repl.Evaluator, cfg Config) repl.Evaluator` — decorator that wraps an Evaluator and intercepts `/help` before delegating.

## 5) Function-level design

### 5.1 ParseHelpInput

Responsibility: Convert raw `/help ...` into a `HelpArgs` struct.

Rules:
- Recognize `/help` prefix; trim.
- Tokens starting with `--` are flags. Support forms: `--all`, `--query=...`, `--type=tutorial`, `--topic=db` (repeatable), `--flag=--x`, `--command=json`, `--search="text"`, `--list`.
- First non-flag token becomes `Slug` if present.

### 5.2 BuildDSL

Responsibility: If `args.Query` is present, use it. Else translate other filters to DSL.

Algorithm:
- Collect per-field disjunctions (OR within field). Types map to `type:example`, etc.
- Conjoin different fields with `AND`.
- If only `Search` is present: `"Search"`.
- Return `("", false)` if we should do slug/top-level logic instead.

### 5.3 HandleHelpCommand

Responsibility: One entrypoint to produce markdown for any help input.

Flow:
1) Parse args → `HelpArgs`.
2) If `ShowAll || (no slug && no filters && no query && no search)` → top-level page via `Backend.TopLevel`.
3) If `Slug != ""` → fetch by slug via `Backend.GetBySlug`; if not found, suggest search.
4) Else try `BuildDSL`. If DSL present → `Backend.Query(DSL)`; if `ok==false`, render a message indicating the backend does not support queries.
5) Render via `Renderer`.
6) If `ShowRelated` and a single section was chosen, compute related sets when the backend can supply them. For the generic contract, related content is optional; backends may embed discoverability inside `TopLevel` or expose a `Related` helper in an extended interface. The Glazed adapter can implement this by running internal SectionQuery calls and mapping the result to `related map[string][]*Section`.

### 5.4 Renderer contract

Default renderer produces conservative markdown that works well in the REPL timeline. Apps can swap it for fancier formatting. The contract uses generic `Section` and `TopLevelPage` only, ensuring no dependency on oak or Glazed.

## 6) Backward compatibility and migration

No backwards compatibility is required in the REPL layer:
- New code targets the `Backend` interface only.
- `oak-repl` adopts the new handler and uses a Glazed adapter to populate the backend. The old inline logic is removed.
- If desired, a simple file-based backend can be provided later (slug + top-level only) without touching the REPL code.

## 7) Testing strategy

- Unit tests for parsing: `ParseHelpInput` and `BuildDSL` combinations.
- Integration tests with an in-memory `HelpSystem` loaded with fixtures; verify rendering for:
  - Top-level; `--all`
  - Slug show and not found case
  - DSL queries with AND/OR/NOT and parentheses
  - Convenience flags mapped to DSL
  - Related sections rendering

## 8) Open questions / future work

- Should we surface pagination for large query results? For now, we limit to a max count and show a hint when truncated.
- Add `--json` output that emits `repl.Event{Kind: StructuredData}` for tooling? Not required now.
- Add fuzzy matching for slugs/titles as a fallback before full-text search.

## 9) Summary of planned changes (files and functions)

- bobatea/pkg/repl/help/api.go
  - `type Section`, `type TopLevelPage`, `type Backend`

- bobatea/pkg/repl/help/handler.go
  - `HandleHelpCommand(ctx, cfg, input) (string)`
  - `type Config { Backend Backend; ShowRelated bool; Renderer Renderer; ContextProvider func() (string, []string) }`

- bobatea/pkg/repl/help/parse.go
  - `ParseHelpInput(raw string) (HelpArgs, error)`
  - `BuildDSL(args HelpArgs) (string, bool)`

- bobatea/pkg/repl/help/render.go
  - `type Renderer interface { RenderTopLevel(*TopLevelPage) string; RenderSection(*Section, map[string][]*Section) string; RenderQueryResults([]*Section) string }`
  - `DefaultRenderer() Renderer`

- bobatea/pkg/repl/help/adapters/glazed.go
  - `type GlazedBackend struct { HS *help.HelpSystem }` implementing `Backend`

- oak/cmd/oak-repl/help_integration.go
  - `NewREPLHelp(cfg replhelp.Config) func(ctx context.Context, code string, emit func(repl.Event)) bool`
  - Update `main.go` to call the handler from `EvaluateStream` using a `GlazedBackend`

This will deliver a consistent, powerful, and reusable help experience across REPLs, powered by pluggable backends (with a Glazed adapter as one option). Queries are supported when the chosen backend implements them; slug and `--all` are always available.


