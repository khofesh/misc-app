## Response Style

- Be extremely concise. No pleasantries, no filler.
- When asked to write code, return code only unless explanation is explicitly requested.
- No sycophantic preambles ("Sure!", "Great question!", "Absolutely!").
- No "Here's a function that..." preambles.
- Don't restate the question before answering.
- No "Note:", "Tip:", or "Remember:" appendices unless asked.
- No usage examples unless asked.
- No unsolicited suggestions or improvements beyond what was asked.
- Use short variable names where meaning is clear from context.

## Project

Collection of small Go CLI tools for daily problems. Module `github.com/khofesh/misc-app`, Go 1.26.

```
cmd/<tool>/main.go       one binary per tool (package main)
cmd/examples/<name>/     learning examples, not real tools
internal/<pkg>/          shared packages (htmltag, spinner)
docs/                    specs for planned tools (e.g. rmbg-spec.md)
input/, output/          local sample data, gitignored, may contain personal files
bin/                     build output, gitignored
```

Tools: `html2md`, `html2text`, `decode-encode`, `pdf-ocr` (needs `tesseract`), `utc2local`, `spinner`, `examples/http-middleware`.

Most tools read fixed paths under `input/` and write to `output/`, relative to the repo root. Run them from the repo root.

## Commands

```bash
make run/<tool>            # go run ./cmd/<tool>
make build/debug/<tool>    # debug build into ./bin
make build/utc2local       # release build
make audit                 # tidy, fmt, vet, staticcheck, test -race
go test ./...
```

When adding a tool, add `run/<tool>` and `build/debug/<tool>` targets to the Makefile and a section to README.md.

## Code Style

- Go, standard library first. Format with `gofmt`. Code must pass `go vet` and `staticcheck`.
- New tools go in `cmd/<tool>`. Move logic that's worth testing or reusing to `internal/<pkg>`.
- Errors: wrap with `fmt.Errorf("...: %w", err)` in packages. In `main`, log and `os.Exit(1)`.
- Logging: `github.com/rs/zerolog` for CLI tools, `log/slog` in examples.
- Minimal inline comments, only for genuinely complex logic.

## Testing

- Table-driven tests with `t.Run`, standard `testing` package only.
- Never read or write the real `input/` or `output/` in tests. Use `t.TempDir()` and `t.Chdir()`.
- For a `main` that calls `os.Exit`, re-run the test binary as a subprocess (see `runMain` / `TestMainProcess` in `cmd/decode-encode/main_test.go`).
- Tests must pass with `go test -race ./...`.

## Context Retrieval Policy

Always retrieve the smallest amount of information necessary.
Escalate only when necessary. Stop escalating as soon as sufficient information has been obtained.

Preferred order:

1. Need code structure? `graphify query` / `explain` / `path` (see Graphify).
2. Need directory structure? `rtk tree`.
3. Need symbols? `ast-grep`.
4. Need implementation? Source files.
5. Use repository-wide search as last resort.

Avoid reading entire directories or the whole repository unless explicitly requested.

<!-- rtk-instructions v2 -->

## RTK (Rust Token Killer) - Token-Optimized CLI

`rtk` is a CLI proxy that filters and compresses command outputs, saving 60-90% tokens.

**Always** prefix commands with `rtk`. If RTK has a dedicated filter, it uses it. If not, it passes through unchanged. This means RTK is always safe to use.

Instead of:

```
ls -la .
git status
git log -10
go test ./...
```

Use:

```
rtk ls -la .
rtk git status
rtk git log -10
rtk go test ./...
```

### RTK commands example

```bash
rtk ls <path>
rtk tree <path>
rtk read <file>
rtk find <pattern>

rtk err <cmd>           # Filter errors only from any command
rtk log <file>          # Deduplicated logs with counts
rtk json <file>         # JSON structure without values

rtk curl <url>          # Compact HTTP responses
```

<!-- /rtk-instructions -->

## Graphify - Codebase context & knowledge graph protocol

A pre-computed AST knowledge graph is available in `graphify-out/` (`graph.json`, `GRAPH_REPORT.md`, `graph.html`). It covers the Go code only. `input/`, `output/` and `bin/` are not in it.

Use the graph before searching or reading multiple source files:

```bash
graphify query "<question>"     # scoped subgraph for a question
graphify explain "<symbol>"     # one node and its neighbours
graphify path "<A>" "<B>"       # shortest path between two symbols
```

Workflow:

1. Query the graph. Don't read `graph.json` whole.
2. Identify the relevant symbols, files, and dependency paths.
3. Read only the source files required for the task.
4. Avoid scanning unrelated files.

Use the graph for:

- dependency tracing
- call-path discovery
- identifying high-centrality modules
- impact analysis before refactoring
- locating symbol definitions

Read `graphify-out/GRAPH_REPORT.md` only for broad architecture review.

The graph is generated from AST extraction and represents extracted structural relationships only. Treat graph edges as authoritative for structural relationships.
Do not infer dependencies that are absent from the graph. Stdlib and third-party packages appear as stub nodes with no internals.

Keeping it current: a post-commit hook rebuilds the graph after every commit. After modifying code mid-session, run `graphify update .` (AST-only, no API cost).

## ast-grep

Prefer `ast-grep` over `grep` whenever searching source code.

```bash
ast-grep run -l go -p 'func $NAME($$$) $$$'           # functions
ast-grep run -l go -p 'func ($R $T) $NAME($$$) $$$'   # methods
ast-grep run -l go -p 'type $NAME struct { $$$ }'     # structs
ast-grep run -l go -p '$PKG.$FN($$$)'                 # calls
```

Use `ast-grep` for:

- locating function definitions
- locating type and struct definitions
- locating method implementations
- finding imports
- matching AST patterns
- performing structural search

Use `grep` only for:

- Markdown
- JSON
- YAML
- go.mod / go.sum
- log files
- generated text
- plain-text configuration

Avoid using `grep` to search Go source when `ast-grep` can answer the query.
