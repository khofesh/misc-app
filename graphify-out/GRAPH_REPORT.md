# Graph Report - misc-app  (2026-10-12)

## Corpus Check
- 33 files · ~11,818 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 6 file(s) not represented in the graph (top: (none) 6)

## Summary
- 256 nodes · 633 edges · 13 communities (11 shown, 2 thin omitted)
- Extraction: 88% EXTRACTED · 12% INFERRED · 0% AMBIGUOUS · INFERRED: 76 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `b3f40d3f`
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
- rmbg/main_test.go
- image.Image

## God Nodes (most connected - your core abstractions)
1. `Remover` - 20 edges
2. `New()` - 12 edges
3. `ModelPath()` - 12 edges
4. `ApplyAlpha()` - 9 edges
5. `Blend()` - 9 edges
6. `solid()` - 9 edges
7. `New()` - 9 edges
8. `encodeOutput()` - 8 edges
9. `Spinner` - 8 edges
10. `HtmlToText()` - 8 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `New()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/remover.go
- `parseFlags()` --calls--> `ParseHexColor()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/postprocess.go
- `processFile()` --calls--> `Threshold()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/postprocess.go
- `processFile()` --calls--> `Decode()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/preprocess.go
- `encodeOutput()` --calls--> `ApplyAlpha()`  [EXTRACTED]
  cmd/rmbg/main.go → internal/rmbg/postprocess.go

## Import Cycles
- None detected.

## Communities (13 total, 2 thin omitted)

### Community 0 - "utc2local/main_test.go"
Cohesion: 0.11
Nodes (12): main(), TestMainConvertsFile(), convertToTimezone(), main(), showHelp(), runMain(), TestConvertToTimezone(), TestConvertToTimezoneInvalid() (+4 more)

### Community 1 - "http-middleware/main_test.go"
Cohesion: 0.17
Nodes (10): admin(), home(), logRequest(), main(), requireBasicAuthentication(), serverHeader(), TestHandlers(), TestLogRequest() (+2 more)

### Community 2 - "postprocess_test.go"
Cohesion: 0.14
Nodes (19): ApplyAlpha(), Blend(), blend(), MaskToGray(), NormalizeMask(), ParseHexColor(), ResizeMask(), solid() (+11 more)

### Community 3 - "spinner_test.go"
Cohesion: 0.18
Nodes (10): main(), captureStdout(), TestNewDefaults(), TestRestart(), TestSetters(), TestStartStop(), TestStartTwiceIsNoop(), TestStopWithoutStartIsNoop() (+2 more)

### Community 4 - "html2text.go"
Cohesion: 0.35
Nodes (7): atLineStartOrSpace(), collapseSpace(), ensureNewline(), isHTMLSpace(), writeChildren(), writeInline(), writeNode()

### Community 5 - "testing.T"
Cohesion: 0.17
Nodes (23): main(), readOutput(), runMain(), setupDir(), TestDecode(), TestDecodeInvalidEscape(), TestEncode(), TestFlagValidation() (+15 more)

### Community 6 - "rmbg/main.go"
Cohesion: 0.14
Nodes (7): meanAbsGrayDiff(), TestCPUvsCUDA(), fixture(), ortLibPath(), BenchmarkRemove(), meanAlpha(), TestRemoveIntegration()

### Community 7 - "models.go"
Cohesion: 0.14
Nodes (19): CacheDir(), customSpec(), download(), ensureModel(), isModelPath(), Lookup(), ModelPath(), RegistryNames() (+11 more)

### Community 10 - "Remover"
Cohesion: 0.15
Nodes (8): destroyTensors(), New(), closeORT(), initORT(), newSessionOptions(), Device, Options, Remover

### Community 11 - "rmbg/main_test.go"
Cohesion: 0.16
Nodes (18): encodeOutput(), isJPEG(), isSupportedImage(), main(), outputBase(), parseFlags(), processDir(), processFile() (+10 more)

### Community 12 - "image.Image"
Cohesion: 0.15
Nodes (11): applyOrientation(), exifOrientation(), jpegOrientation(), Decode(), Preprocess(), TestApplyOrientation(), TestDecodeInvalid(), TestDecodePNG() (+3 more)

## Knowledge Gaps
- **1 isolated node(s):** `github.com/khofesh/misc-app`
  These have ≤1 connection - possible missing edges. (Counts symbols only; 31 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Remover` connect `Remover` to `postprocess_test.go`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Are the 7 inferred relationships involving `New()` (e.g. with `TestCPUvsCUDA()` and `BenchmarkRemove()`) actually correct?**
  _`New()` has 7 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/khofesh/misc-app` to the rest of the system?**
  _1 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `utc2local/main_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.11 - nodes in this community are weakly interconnected._
- **Why does `New()` connect `Remover` to `rmbg/main_test.go`, `testing.T`, `rmbg/main.go`, `models.go`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Should `postprocess_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.14245014245014245 - nodes in this community are weakly interconnected._
- **Should `rmbg/main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.1402116402116402 - nodes in this community are weakly interconnected._