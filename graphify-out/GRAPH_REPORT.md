# Graph Report - misc-app  (2026-10-12)

## Corpus Check
- 17 files · ~4,132 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 8 file(s) not represented in the graph (top: (none) 8)

## Summary
- 111 nodes · 239 edges · 10 communities (7 shown, 3 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 20 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `0d0f4394`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- utc2local/main.go
- http-middleware/main_test.go
- HtmlToText
- spinner_test.go
- html2text.go
- testing.T
- go_pkg_os
- Spinner
- github.com/khofesh/misc-app
- utc2local/main_test.go

## God Nodes (most connected - your core abstractions)
1. `New()` - 9 edges
2. `HtmlToText()` - 8 edges
3. `Spinner` - 8 edges
4. `runMain()` - 7 edges
5. `setupDir()` - 7 edges
6. `writeNode()` - 7 edges
7. `captureStdout()` - 6 edges
8. `TestEncode()` - 5 edges
9. `TestDecode()` - 5 edges
10. `main()` - 5 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `HtmlToText()`  [EXTRACTED]
  cmd/html2text/main.go → internal/htmltag/html2text.go
- `main()` --calls--> `New()`  [EXTRACTED]
  cmd/spinner/main.go → internal/spinner/spinnner.go
- `TestMainProcess()` --calls--> `main()`  [INFERRED]
  cmd/decode-encode/main_test.go → cmd/pdf-ocr/main.go
- `TestServerHeader()` --calls--> `serverHeader()`  [INFERRED]
  cmd/examples/http-middleware/main_test.go → cmd/examples/http-middleware/main.go
- `TestLogRequest()` --calls--> `logRequest()`  [INFERRED]
  cmd/examples/http-middleware/main_test.go → cmd/examples/http-middleware/main.go

## Import Cycles
- None detected.

## Communities (10 total, 3 thin omitted)

### Community 0 - "utc2local/main.go"
Cohesion: 0.19
Nodes (5): convertToTimezone(), main(), showHelp(), TestConvertToTimezone(), TestConvertToTimezoneInvalid()

### Community 1 - "http-middleware/main_test.go"
Cohesion: 0.16
Nodes (10): admin(), home(), logRequest(), main(), requireBasicAuthentication(), serverHeader(), TestHandlers(), TestLogRequest() (+2 more)

### Community 2 - "HtmlToText"
Cohesion: 0.48
Nodes (6): HtmlToText(), TestHtmlToText(), TestHtmlToTextEmpty(), TestHtmlToTextLayout(), TestHtmlToTextNoBlankLineRuns(), TestHtmlToTextNoDuplicateText()

### Community 3 - "spinner_test.go"
Cohesion: 0.32
Nodes (9): main(), captureStdout(), TestNewDefaults(), TestRestart(), TestSetters(), TestStartStop(), TestStartTwiceIsNoop(), TestStopWithoutStartIsNoop() (+1 more)

### Community 4 - "html2text.go"
Cohesion: 0.35
Nodes (7): atLineStartOrSpace(), collapseSpace(), ensureNewline(), isHTMLSpace(), writeChildren(), writeInline(), writeNode()

### Community 5 - "testing.T"
Cohesion: 0.35
Nodes (13): readOutput(), runMain(), setupDir(), TestDecode(), TestDecodeInvalidEscape(), TestEncode(), TestFlagValidation(), TestMainProcess() (+5 more)

### Community 9 - "utc2local/main_test.go"
Cohesion: 0.53
Nodes (4): runMain(), TestMainInvalidTime(), TestMainNoArgsShowsHelp(), TestMainOutput()

## Knowledge Gaps
- **1 isolated node(s):** `github.com/khofesh/misc-app`
  These have ≤1 connection - possible missing edges. (Counts symbols only; 18 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Spinner` connect `Spinner` to `spinner_test.go`?**
  _High betweenness centrality (0.105) - this node is a cross-community bridge._
- **Are the 6 inferred relationships involving `New()` (e.g. with `TestNewDefaults()` and `TestRestart()`) actually correct?**
  _`New()` has 6 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/khofesh/misc-app` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Why does `New()` connect `spinner_test.go` to `Spinner`?**
  _High betweenness centrality (0.075) - this node is a cross-community bridge._
- **Are the 5 inferred relationships involving `HtmlToText()` (e.g. with `TestHtmlToText()` and `TestHtmlToTextEmpty()`) actually correct?**
  _`HtmlToText()` has 5 INFERRED edges - model-reasoned connections that need verification._
- **Why does `HtmlToText()` connect `HtmlToText` to `html2text.go`, `go_pkg_os`?**
  _High betweenness centrality (0.044) - this node is a cross-community bridge._