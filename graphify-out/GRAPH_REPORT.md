# Graph Report - misc-app  (2026-10-12)

## Corpus Check
- Corpus is ~6,884 words - fits in a single context window. You may not need a graph.

## Summary
- 111 nodes · 239 edges · 9 communities (6 shown, 3 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 20 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- utc2local & decode-encode CLIs
- HTTP Middleware Example
- HTML-to-Text Tests
- Spinner Package & Tests
- HtmlToText Converter
- decode-encode Tests
- html2md & pdf-ocr CLIs
- Spinner Methods
- Go Module

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
10. `convertToTimezone()` - 5 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `New()`  [EXTRACTED]
  cmd/spinner/main.go → internal/spinner/spinnner.go
- `TestMainProcess()` --calls--> `main()`  [INFERRED]
  cmd/decode-encode/main_test.go → cmd/decode-encode/main.go
- `TestMainConvertsFile()` --calls--> `main()`  [INFERRED]
  cmd/html2md/main_test.go → cmd/html2md/main.go
- `main()` --calls--> `HtmlToText()`  [EXTRACTED]
  cmd/html2text/main.go → internal/htmltag/html2text.go
- `TestServerHeader()` --calls--> `serverHeader()`  [INFERRED]
  cmd/examples/http-middleware/main_test.go → cmd/examples/http-middleware/main.go

## Import Cycles
- None detected.

## Communities (9 total, 3 thin omitted)

### Community 0 - "utc2local & decode-encode CLIs"
Cohesion: 0.14
Nodes (11): main(), convertToTimezone(), main(), showHelp(), runMain(), TestConvertToTimezone(), TestConvertToTimezoneInvalid(), TestMainInvalidTime() (+3 more)

### Community 1 - "HTTP Middleware Example"
Cohesion: 0.16
Nodes (10): admin(), home(), logRequest(), main(), requireBasicAuthentication(), serverHeader(), TestHandlers(), TestLogRequest() (+2 more)

### Community 2 - "HTML-to-Text Tests"
Cohesion: 0.21
Nodes (9): TestMainConvertsFile(), main(), TestMainConvertsFile(), HtmlToText(), TestHtmlToText(), TestHtmlToTextEmpty(), TestHtmlToTextLayout(), TestHtmlToTextNoBlankLineRuns() (+1 more)

### Community 3 - "Spinner Package & Tests"
Cohesion: 0.32
Nodes (9): main(), captureStdout(), TestNewDefaults(), TestRestart(), TestSetters(), TestStartStop(), TestStartTwiceIsNoop(), TestStopWithoutStartIsNoop() (+1 more)

### Community 4 - "HtmlToText Converter"
Cohesion: 0.35
Nodes (7): atLineStartOrSpace(), collapseSpace(), ensureNewline(), isHTMLSpace(), writeChildren(), writeInline(), writeNode()

### Community 5 - "decode-encode Tests"
Cohesion: 0.55
Nodes (9): readOutput(), runMain(), setupDir(), TestDecode(), TestDecodeInvalidEscape(), TestEncode(), TestFlagValidation(), TestMainProcess() (+1 more)

## Knowledge Gaps
- **1 isolated node(s):** `github.com/khofesh/misc-app`
  These have ≤1 connection - possible missing edges. (Counts symbols only; 17 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Spinner` connect `Spinner Methods` to `Spinner Package & Tests`?**
  _High betweenness centrality (0.105) - this node is a cross-community bridge._
- **Are the 6 inferred relationships involving `New()` (e.g. with `TestNewDefaults()` and `TestRestart()`) actually correct?**
  _`New()` has 6 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/khofesh/misc-app` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `utc2local & decode-encode CLIs` be split into smaller, more focused modules?**
  _Cohesion score 0.14285714285714285 - nodes in this community are weakly interconnected._
- **Why does `New()` connect `Spinner Package & Tests` to `Spinner Methods`?**
  _High betweenness centrality (0.077) - this node is a cross-community bridge._
- **Are the 5 inferred relationships involving `HtmlToText()` (e.g. with `TestHtmlToText()` and `TestHtmlToTextEmpty()`) actually correct?**
  _`HtmlToText()` has 5 INFERRED edges - model-reasoned connections that need verification._
- **Why does `HtmlToText()` connect `HTML-to-Text Tests` to `HtmlToText Converter`?**
  _High betweenness centrality (0.041) - this node is a cross-community bridge._