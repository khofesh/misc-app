# Graph Report - misc-app  (2026-10-23)

## Corpus Check
- 57 files · ~26,922 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 7 file(s) not represented in the graph (top: (none) 6, .xml 1)

## Summary
- 495 nodes · 1397 edges · 17 communities (13 shown, 4 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 195 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `40513458`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- pdf2md/main.go
- go_pkg_testing
- rmbg/main.go
- spinner_test.go
- html2text.go
- testing.T
- layout.go
- models.go
- github.com/khofesh/misc-app
- opencode.json
- Remover
- Page
- convert_test.go
- render.go
- clean.go
- gen.sh

## God Nodes (most connected - your core abstractions)
1. `Page` - 27 edges
2. `Remover` - 22 edges
3. `Line` - 21 edges
4. `Block` - 21 edges
5. `Span` - 15 edges
6. `sp()` - 15 edges
7. `Converter` - 14 edges
8. `Rect` - 14 edges
9. `newLine()` - 13 edges
10. `renderBlock()` - 12 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `New()`  [EXTRACTED]
  cmd/pdf2md/main.go → internal/pdf2md/convert.go
- `config` --references--> `Device`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/runtime.go
- `run()` --calls--> `New()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/remover.go
- `processDir()` --references--> `Remover`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/remover.go
- `processFile()` --references--> `Remover`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/remover.go

## Import Cycles
- None detected.

## Communities (17 total, 4 thin omitted)

### Community 0 - "pdf2md/main.go"
Cohesion: 0.07
Nodes (19): main(), main(), TestMainConvertsFile(), New(), parsePageRanges(), selectPages(), TestParsePageRanges(), FindMutool() (+11 more)

### Community 1 - "go_pkg_testing"
Cohesion: 0.17
Nodes (10): admin(), home(), logRequest(), main(), requireBasicAuthentication(), serverHeader(), TestHandlers(), TestLogRequest() (+2 more)

### Community 2 - "rmbg/main.go"
Cohesion: 0.06
Nodes (46): encodeOutput(), isSupportedImage(), main(), outputBase(), parseFlags(), processDir(), processFile(), run() (+38 more)

### Community 3 - "spinner_test.go"
Cohesion: 0.18
Nodes (10): main(), captureStdout(), TestNewDefaults(), TestRestart(), TestSetters(), TestStartStop(), TestStartTwiceIsNoop(), TestStopWithoutStartIsNoop() (+2 more)

### Community 4 - "html2text.go"
Cohesion: 0.40
Nodes (7): atLineStartOrSpace(), collapseSpace(), ensureNewline(), isHTMLSpace(), writeChildren(), writeInline(), writeNode()

### Community 5 - "testing.T"
Cohesion: 0.09
Nodes (45): readOutput(), runMain(), setupDir(), TestDecode(), TestDecodeInvalidEscape(), TestEncode(), TestFlagValidation(), TestMainProcess() (+37 more)

### Community 6 - "layout.go"
Cohesion: 0.07
Nodes (53): attachCaptions(), hasLetter(), isBoldHeading(), isSizeHeading(), listLevels(), newClassifier(), para(), TestAttachCaptions() (+45 more)

### Community 7 - "models.go"
Cohesion: 0.15
Nodes (19): CacheDir(), customSpec(), download(), ensureModel(), isModelPath(), Lookup(), ModelPath(), RegistryNames() (+11 more)

### Community 10 - "Remover"
Cohesion: 0.12
Nodes (13): destroyTensors(), Remover, New(), closeORT(), FindORTLib(), Device, initORT(), newSessionOptions() (+5 more)

### Community 11 - "Page"
Cohesion: 0.08
Nodes (29): isPDF(), isTerminal(), parseFlags(), processDir(), processFile(), run(), bodySize(), normTitle() (+21 more)

### Community 12 - "convert_test.go"
Cohesion: 0.12
Nodes (18): TestFontStyle(), TestFormatPages(), TestNeedsOCR(), TestParseHOCR(), cleanFontName(), fontStyle(), stringsReader(), attr() (+10 more)

### Community 13 - "render.go"
Cohesion: 0.19
Nodes (20): collapseSpace(), needsSpace(), emphasize(), escapeInline(), escapeStart(), escapeURL(), formatRuns(), inlineRuns() (+12 more)

### Community 14 - "clean.go"
Cohesion: 0.19
Nodes (9): dehyphenate(), joinLines(), mergeAcrossPages(), normalizeSpans(), normalizeText(), removeHeadersFooters(), TestJoinLines(), TestMergeAcrossPages() (+1 more)

## Knowledge Gaps
- **1 isolated node(s):** `github.com/khofesh/misc-app`
  These have ≤1 connection - possible missing edges. (Counts symbols only; 52 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Remover` connect `Remover` to `rmbg/main.go`, `models.go`?**
  _High betweenness centrality (0.043) - this node is a cross-community bridge._
- **What connects `github.com/khofesh/misc-app` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `pdf2md/main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07403846153846154 - nodes in this community are weakly interconnected._
- **Why does `Page` connect `Page` to `pdf2md/main.go`, `layout.go`, `convert_test.go`, `render.go`, `clean.go`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Should `rmbg/main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.06416275430359937 - nodes in this community are weakly interconnected._
- **Why does `Line` connect `layout.go` to `pdf2md/main.go`, `Page`, `render.go`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.08525506638714186 - nodes in this community are weakly interconnected._