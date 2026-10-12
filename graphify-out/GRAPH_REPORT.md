# Graph Report - misc-app  (2026-10-12)

## Corpus Check
- 32 files · ~11,505 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 6 file(s) not represented in the graph (top: (none) 6)

## Summary
- 248 nodes · 615 edges · 12 communities (10 shown, 2 thin omitted)
- Extraction: 88% EXTRACTED · 12% INFERRED · 0% AMBIGUOUS · INFERRED: 72 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `814d10a2`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- utc2local/main_test.go
- http-middleware/main_test.go
- postprocess_test.go
- spinner_test.go
- html2text.go
- testing.T
- rmbg/main.go
- models.go
- github.com/khofesh/misc-app
- opencode.json
- Remover
- processFile

## God Nodes (most connected - your core abstractions)
1. `Remover` - 20 edges
2. `ModelPath()` - 12 edges
3. `New()` - 12 edges
4. `ApplyAlpha()` - 9 edges
5. `Blend()` - 9 edges
6. `solid()` - 9 edges
7. `New()` - 9 edges
8. `processFile()` - 8 edges
9. `encodeOutput()` - 8 edges
10. `HtmlToText()` - 8 edges

## Surprising Connections (you probably didn't know these)
- `config` --references--> `Device`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/runtime.go
- `run()` --calls--> `New()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/remover.go
- `processDir()` --references--> `Remover`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/remover.go
- `processFile()` --calls--> `Threshold()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/postprocess.go
- `processFile()` --calls--> `Decode()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/preprocess.go

## Import Cycles
- None detected.

## Communities (12 total, 2 thin omitted)

### Community 0 - "utc2local/main_test.go"
Cohesion: 0.22
Nodes (10): convertToTimezone(), main(), showHelp(), runMain(), TestConvertToTimezone(), TestConvertToTimezoneInvalid(), TestMainInvalidTime(), TestMainNoArgsShowsHelp() (+2 more)

### Community 1 - "http-middleware/main_test.go"
Cohesion: 0.17
Nodes (10): admin(), home(), logRequest(), main(), requireBasicAuthentication(), serverHeader(), TestHandlers(), TestLogRequest() (+2 more)

### Community 2 - "postprocess_test.go"
Cohesion: 0.09
Nodes (29): encodeOutput(), applyOrientation(), exifOrientation(), jpegOrientation(), ApplyAlpha(), Blend(), blend(), MaskToGray() (+21 more)

### Community 3 - "spinner_test.go"
Cohesion: 0.18
Nodes (10): main(), captureStdout(), TestNewDefaults(), TestRestart(), TestSetters(), TestStartStop(), TestStartTwiceIsNoop(), TestStopWithoutStartIsNoop() (+2 more)

### Community 4 - "html2text.go"
Cohesion: 0.35
Nodes (7): atLineStartOrSpace(), collapseSpace(), ensureNewline(), isHTMLSpace(), writeChildren(), writeInline(), writeNode()

### Community 5 - "testing.T"
Cohesion: 0.13
Nodes (30): main(), readOutput(), runMain(), setupDir(), TestDecode(), TestDecodeInvalidEscape(), TestEncode(), TestFlagValidation() (+22 more)

### Community 6 - "rmbg/main.go"
Cohesion: 0.09
Nodes (12): main(), TestMainConvertsFile(), meanAbsGrayDiff(), TestCPUvsCUDA(), fixture(), ortLibPath(), BenchmarkRemove(), meanAlpha() (+4 more)

### Community 7 - "models.go"
Cohesion: 0.14
Nodes (19): CacheDir(), customSpec(), download(), ensureModel(), isModelPath(), Lookup(), ModelPath(), RegistryNames() (+11 more)

### Community 10 - "Remover"
Cohesion: 0.18
Nodes (6): destroyTensors(), Remover, closeORT(), Device, newSessionOptions(), Options

### Community 11 - "processFile"
Cohesion: 0.36
Nodes (7): parseFlags(), processDir(), processFile(), run(), ParseHexColor(), TestParseHexColor(), config

## Knowledge Gaps
- **1 isolated node(s):** `github.com/khofesh/misc-app`
  These have ≤1 connection - possible missing edges. (Counts symbols only; 30 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Remover` connect `Remover` to `postprocess_test.go`, `processFile`, `rmbg/main.go`, `models.go`?**
  _High betweenness centrality (0.077) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `ModelPath()` (e.g. with `TestModelPathCustom()` and `TestModelPathMissingCustom()`) actually correct?**
  _`ModelPath()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/khofesh/misc-app` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `postprocess_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.0859465737514518 - nodes in this community are weakly interconnected._
- **Why does `New()` connect `rmbg/main.go` to `Remover`, `processFile`, `models.go`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.12834224598930483 - nodes in this community are weakly interconnected._
- **Should `rmbg/main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08599290780141844 - nodes in this community are weakly interconnected._