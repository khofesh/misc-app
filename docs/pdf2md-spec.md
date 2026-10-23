# pdf2md — PDF to Markdown CLI (Spec)

Status: Draft
Location: `cmd/pdf2md`, `internal/pdf2md`

## 1. Goal

A command-line tool written in Go that converts PDF documents into readable
Markdown. It keeps the document structure that matters for reading and
searching: headings, paragraphs, lists, emphasis, links, code blocks, and
simple tables. Text and layout come from MuPDF's `mutool`. Scanned pages
without a text layer go through OCR, reusing the `tesseract` setup that
`pdf-ocr` already needs.

The output is meant to be good enough to read, diff, grep, or feed into an LLM.
It is not a pixel-faithful reproduction of the PDF.

### Non-goals (v1)

- Math and formulas (LaTeX output)
- Complex tables (merged cells, nested tables, tables that span pages)
- Forms, annotations, and comments
- Encrypted PDFs that need a password other than empty (see `-password`, §2)
- Right-to-left and vertical scripts
- A GUI or HTTP server
- Any cloud or LLM-based conversion

## 2. Usage

```shell
# single file, Markdown to stdout
pdf2md -i input/report.pdf

# single file to a file
pdf2md -i input/report.pdf -o output/report.md

# whole directory (non-recursive), output keeps the same base names as .md
pdf2md -i input/pdfs/ -o output/pdfs/

# only some pages, and extract embedded images next to the .md
pdf2md -i input/book.pdf -o output/book.md -pages 1-5,9 -images

# scanned document, force OCR in Indonesian and English
pdf2md -i input/scan.pdf -o output/scan.md -ocr always -lang ind+eng
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-i` | (required) | Input `.pdf` file or directory |
| `-o` | (empty) | Output `.md` file or directory. Empty means stdout (single-file mode only). Directories are created if missing |
| `-pages` | (all) | Page selection, e.g. `1-3,7,10-`. 1-based |
| `-password` | (empty) | User password for encrypted PDFs |
| `-backend` | `auto` | `auto`, `mutool`, or `native` (see §4) |
| `-mutool` | (`PATH`) | Path to the `mutool` binary |
| `-ocr` | `auto` | `auto`, `never`, or `always` (see §6) |
| `-lang` | `eng` | Tesseract language(s), passed as `-l` |
| `-images` | `false` | Extract embedded images into `<name>_images/` and link them |
| `-page-breaks` | `false` | Insert `<!-- page N -->` comments between pages |
| `-keep-headers` | `false` | Keep repeated page headers and footers (removed by default, §5.6) |
| `-tables` | `true` | Detect simple tables and emit GFM tables |
| `-front-matter` | `false` | Write a YAML front matter block with title, author, page count, and source file |
| `-workers` | `runtime.NumCPU()` | Pages processed in parallel |
| `-overwrite` | `false` | Overwrite existing output files |
| `-v` | `false` | Verbose logging (per-page timing, OCR decisions, detected body font size) |

### Exit codes

| Code | Meaning |
|---|---|
| 0 | All files converted |
| 1 | Invalid flags or input |
| 2 | A required external tool is missing (e.g. `-ocr always` without `tesseract`), or `-backend mutool` without `mutool` |
| 3 | Some files in directory mode failed (others succeeded) |

## 3. Architecture

```
cmd/pdf2md/main.go        flag parsing, logging (zerolog), file walking
internal/pdf2md/
  extract.go              Extractor interface, backend selection
  mutool.go               mutool stext XML → []Page (primary)
  native.go               ledongthuc/pdf → []Page (fallback)
  layout.go               spans → lines → blocks, reading order, columns
  classify.go             block → heading / paragraph / list / code / table / caption
  clean.go                header/footer removal, de-hyphenation, whitespace
  ocr.go                  page render + tesseract fallback
  images.go               embedded image extraction (pdfcpu)
  render.go               blocks → Markdown
  convert.go              Converter type tying everything together
```

### Core API

```go
type OCRMode string

const (
    OCRAuto   OCRMode = "auto"
    OCRNever  OCRMode = "never"
    OCRAlways OCRMode = "always"
)

type Options struct {
    Pages       []int // empty means all
    Password    string
    Backend     Backend // "auto", "mutool", "native"
    MutoolPath  string
    OCR         OCRMode
    Lang        string
    ImageDir    string // empty disables image extraction
    PageBreaks  bool
    KeepHeaders bool
    Tables      bool
    FrontMatter bool
    Workers     int
}

type Extractor interface {
    Extract(ctx context.Context, path string, pages []int) ([]Page, error)
}

type Converter struct { /* options, extractor, external tool paths */ }

func New(opts Options) (*Converter, error)
func (c *Converter) Convert(ctx context.Context, path string, w io.Writer) (Stats, error)

type Stats struct {
    Pages, OCRPages, Images, Tables int
}
```

### Intermediate model

```go
type Span struct {
    Text     string
    X, Y     float64 // baseline origin, PDF units, top-left origin after normalization
    W, H     float64
    Font     string
    Size     float64
    Bold     bool
    Italic   bool
    Mono     bool
    Link     string // URI if inside a link annotation
}

type Line  struct { Spans []Span; BBox Rect }
type Block struct { Kind BlockKind; Lines []Line; Level int; BBox Rect }
type Page  struct { Num int; W, H float64; Blocks []Block; Images []ImageRef }
```

Every stage after `extract.go` works only on this model, so it can be tested
without real PDFs.

## 4. Text extraction

### 4.1 Backend selection

| `-backend` | Behavior |
|---|---|
| `mutool` | Use `mutool`. Exit 2 if it isn't found |
| `native` | Use `ledongthuc/pdf` (pure Go). For machines without MuPDF |
| `auto` | Use `mutool` if found, else log a warning once and use `native` |

`mutool` is looked up in this order: `-mutool` flag, `MUTOOL` environment
variable, `PATH`. The chosen backend and the `mutool -v` version are logged at
startup with `-v`.

### 4.2 mutool (primary)

Run MuPDF's structured text output, one process per page chunk (§9):

```shell
mutool draw -q -F stext -o - [-p <password>] file.pdf <pages>
```

`<pages>` uses mutool's own syntax (`1-3,7,10-N`), built from `-pages`. The
default options expand ligatures and normalize whitespace, which covers most
of the Unicode cleanup in §5.6.

Output shape:

```xml
<document name="file.pdf">
<page id="page1" width="612" height="792">
  <block bbox="72 70 540 96">
    <line bbox="72 70 310 96" wmode="0" dir="1 0">
      <font name="Helvetica-Bold" size="24">
        <char quad="72 70 86 70 72 96 86 96" x="72" y="92" color="#000000" c="I"/>
        ...
      </font>
    </line>
  </block>
  <block type="image" bbox="..."/>
</page>
</document>
```

Mapping to the model (§3):

- Parse with `encoding/xml` as a stream (`xml.Decoder.Token`), never loading
  the whole document into memory.
- Consecutive `<char>` elements under the same `<font>` become one `Span`.
  Span bbox is the union of the char quads. `X`, `Y` come from the first
  char's `x`, `y`. Coordinates are already top-left origin.
- MuPDF's `<line>` maps to `Line` directly. MuPDF's `<block>` is a starting
  hint only: §5.3 may split or merge blocks, because MuPDF block grouping is
  not always paragraph-accurate.
- Bold, italic, and monospace come from char `flags` when the installed MuPDF
  version emits them, otherwise from the font name (`Bold`, `Black`,
  `Semibold`, `Italic`, `Oblique`, `Mono`, `Courier`, `Consolas`, ...).
  Subset prefixes like `ABCDEF+` are stripped first.
- Lines with `dir` other than `1 0` (rotated text, e.g. margin labels) are
  dropped.
- `<block type="image">` bboxes are kept as `ImageRef` positions for §7 and
  for the OCR decision in §6.
- A page whose stext has no chars is marked empty for §6.
- A non-zero exit from `mutool` fails only the chunk's pages; stderr is
  included in the error.

### 4.3 native (fallback)

[`github.com/ledongthuc/pdf`](https://github.com/ledongthuc/pdf), pure Go. It
returns per-glyph text with font name, size, and position, so this backend has
to group glyphs itself:

- Flip Y to a top-left origin.
- Merge glyphs into spans when they share font and size and the gap to the
  previous glyph is below ~0.3× the font size. A larger gap inserts a space.
- Group spans into lines per §5.1. Expand ligatures and normalize to NFC.

This backend is known to fail on some PDFs. A page that fails is passed to OCR
(if `-ocr` allows) or replaced by a placeholder comment. Golden tests (§12)
run against `mutool` only; `native` gets a smaller, looser set.

### 4.4 Shared metadata

Independent of the backend, pdfcpu (already a dependency) provides:

- Link annotations (`/Annots` with `/URI`), attached to spans whose bbox
  overlaps the annotation rect.
- The outline (bookmarks) for heading levels (§5.4).
- Document metadata (title, author) for `-front-matter`.

## 5. Layout analysis

### 5.1 Lines

Group spans whose vertical centers are within 0.5× the median font height.
Sort spans in a line by X.

### 5.2 Columns and reading order

1. Build a horizontal projection of all line bboxes on the page.
2. A vertical gutter is a gap of at least 2× the body font size that runs
   across at least 60% of the page body height.
3. Split the page into columns at gutters. Full-width blocks (titles, figures)
   above or below the columns are placed in order by Y.
4. Reading order: top to bottom within a column, columns left to right.

Max 3 columns in v1. More than that is treated as a table candidate (§5.5).

### 5.3 Blocks

Start a new block when the gap to the previous line is more than 1.3× the
line's typical leading, when the font size changes by more than 10%, or when
the left indent changes by more than 1× the font size (except for the first
line of a paragraph).

### 5.4 Classification

The **body font size** is the most common font size by character count across
the document (not per page).

| Kind | Rule |
|---|---|
| Heading | Font size ≥ 1.15× body size, or bold and on its own short line (≤ 12 words, no final period). Levels are assigned by clustering distinct heading sizes: largest → `#`, next → `##`, up to `######`. The PDF outline (bookmarks), if present, overrides this: block text that matches an outline entry gets that entry's depth |
| List item | First span matches a bullet (`•`, `◦`, `▪`, `–`, `-`, `*`, `·`) or `^\(?\d+[.)]`, `^[a-z][.)]`. Nesting level from left indent |
| Code | All spans monospace. Rendered as a fenced block, keeping indentation from X offsets |
| Caption | Starts with `Figure`, `Fig.`, `Table` + number, smaller than body size. Rendered as italic paragraph |
| Paragraph | Everything else |

### 5.5 Tables (`-tables`)

A block group is a table when at least 3 consecutive lines split into the same
number (≥ 2) of cells, where cells are separated by gaps of at least 1.5× the
font size and column X positions align within a tolerance of 0.5× the font
size. The first row becomes the header. Cell text has `|` escaped. If the shape
is irregular, fall back to plain paragraphs. Ruling lines in the PDF are
ignored in v1.

### 5.6 Cleanup

- **Headers and footers.** A line in the top or bottom 8% of the page whose
  text (with digits normalized to `#`) appears on at least 50% of pages is
  removed. Standalone page numbers are always removed. Disabled by
  `-keep-headers`.
- **De-hyphenation.** A line ending in `-` followed by a line starting with a
  lowercase letter is joined without the hyphen.
- **Line joining.** Lines inside a paragraph are joined with a single space.
- **Ligatures and Unicode.** Normalize to NFC and drop soft hyphens and
  zero-width characters. Ligatures (`ﬁ`, `ﬂ`, `ﬀ`, `ﬃ`, `ﬄ`) are already
  expanded by `mutool`; the `native` backend expands them itself.
- **Paragraphs across pages.** If the last block of a page is a paragraph that
  doesn't end in sentence punctuation and the first block of the next page is a
  paragraph that starts lowercase, merge them.

## 6. OCR

| `-ocr` | Behavior |
|---|---|
| `never` | Never OCR. Pages without text produce a warning and a placeholder comment |
| `always` | OCR every selected page, ignoring the text layer |
| `auto` | OCR a page when it has fewer than 20 extracted characters and contains at least one image covering ≥ 50% of the page |

- Render the page to PNG at 300 DPI with
  `mutool draw -q -r 300 -o <tmp>/p%d.png file.pdf N`. Without `mutool`
  (`native` backend), extract the embedded image with pdfcpu as `pdf-ocr`
  does.
- Run `tesseract <png> - -l <lang> hocr` and parse the hOCR into the same
  `Span` model (word bboxes, `x_size` as font size), so layout and
  classification (§5) work the same way on OCR pages. Bold and italic are not
  detected on OCR pages.
- External tools are looked up once at startup with `exec.LookPath`. Missing
  `tesseract` with `-ocr always` exits with code 2. With `-ocr auto` it logs a
  warning once and continues.

## 7. Images (`-images`)

- Extract embedded images per page with pdfcpu (`api.ExtractImages`, already a
  dependency) into `<output-base>_images/p<page>_<n>.<ext>`.
- Images smaller than 32×32 px or under 1 KB are skipped (decorations).
- Insert `![](<name>_images/p3_1.png)` at the image's position in the reading
  order. A caption block directly below becomes the alt text.
- With stdout output, `-images` is an error (exit 1).

## 8. Rendering

- Output is CommonMark plus GFM tables. One blank line between blocks. Lines
  are not hard-wrapped.
- Emphasis: runs of bold spans → `**text**`, italic → `*text*`, both →
  `***text***`. Adjacent runs with the same style are merged before wrapping,
  and whitespace is moved outside the markers.
- Links: `[text](uri)`. Bare URLs in text are left as-is.
- Escape Markdown-special characters (`*`, `_`, `` ` ``, `#` at line start,
  `[`, `]`, `<`) in plain text, but not inside code blocks.
- Ordered lists keep the source numbering of the first item.
- Front matter (`-front-matter`):

  ```yaml
  ---
  title: "..."
  author: "..."
  pages: 12
  source: report.pdf
  ---
  ```

  `title` falls back to the first `#` heading when the metadata has none.

## 9. Concurrency

Extraction, layout, and OCR run per page in a worker pool of `-workers`.
With `mutool`, the selected pages are split into `-workers` contiguous chunks
and each chunk is one `mutool draw` process, so process start-up cost is paid
per chunk, not per page. Results are collected by page index and rendered in
order. Header/footer
detection and body font size need all pages, so they run after extraction and
before classification. Directory mode processes files one at a time; the page
pool is shared.

## 10. Error handling and logging

- Corrupt or non-PDF input: single-file mode exits 1, directory mode logs,
  skips, and exits 3 at the end.
- Output exists and `-overwrite` isn't set: skip with a warning.
- A page that fails is replaced by a placeholder comment; the file still counts
  as converted, with a warning naming the page.
- Every error names the file (and page, when relevant).
- Logging uses `github.com/rs/zerolog`, console writer on stderr. Normal mode
  logs one line per file with `pages=N ocr=K images=I tables=T elapsed=...`.
  Directory mode ends with `ok=N failed=M skipped=K`.

## 11. Build and Makefile

Pure Go (`CGO_ENABLED=0`). MuPDF is called as a separate process, never
linked, so its AGPL license doesn't apply to the binary.

Runtime tools:

- `mutool` (recommended, primary backend and OCR renderer)
- `tesseract` (optional, OCR)

The README lists the Fedora packages:
`dnf install mupdf tesseract tesseract-langpack-ind`.

```make
## run/pdf2md: run pdf2md on a sample pdf
.PHONY: run/pdf2md
run/pdf2md:
	go run ./cmd/pdf2md -i input/sample.pdf -o output/sample.md -v

## build/debug/pdf2md: build the cmd/pdf2md application with debugging flags enabled
.PHONY: build/debug/pdf2md
build/debug/pdf2md:
	@echo 'Building cmd/pdf2md...'
	go build -gcflags=all="-N -l" -o=./bin/pdf2md-debug ./cmd/pdf2md

## build/pdf2md: build the cmd/pdf2md application
.PHONY: build/pdf2md
build/pdf2md:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o=./bin/pdf2md ./cmd/pdf2md
```

A `pdf2md` section is added to README.md.

## 12. Testing

- **Unit tests (no PDFs)**: build `[]Page` of synthetic spans and check line
  grouping, column split, block splitting, heading levels, list detection and
  nesting, table detection, header/footer removal, de-hyphenation, cross-page
  paragraph merging, emphasis rendering, and Markdown escaping. Table-driven
  with `t.Run`.
- **mutool parser tests**: checked-in stext XML fixtures in `testdata/`
  (no `mutool` needed) for span merging, font flag/name parsing, subset prefix
  stripping, rotated line dropping, and image blocks.
- **Golden tests (`mutool` required, skipped if not on `PATH`)**: small fixture PDFs in `internal/pdf2md/testdata/`
  (generated, not personal files): single column with headings and lists,
  two-column paper layout, a simple table, a code listing, and a page with a
  link. Compare output with `testdata/*.golden.md`. `-update` flag rewrites
  the golden files.
- **native backend tests**: the same fixtures with `-backend native`,
  checking only that key headings and paragraphs appear.
- **OCR tests (`//go:build ocr`)**: a scanned one-page fixture. Assert that
  key words appear in output; run manually where `tesseract` is installed.
- **CLI tests**: `cmd/pdf2md/main_test.go` uses the `runMain` /
  `TestMainProcess` subprocess pattern from `cmd/decode-encode` to check flags
  and exit codes, with `t.TempDir()` and `t.Chdir()`. Never touch the real
  `input/` or `output/`.
- All tests pass with `go test -race ./...` and `make audit`.

## 13. Acceptance criteria

1. A text-based single-column PDF converts with correct heading levels,
   paragraphs joined across line breaks, and lists rendered as Markdown lists.
2. A two-column paper reads column by column, not across columns.
3. Repeated running headers and page numbers don't appear in output by default.
4. A simple ruled or aligned table becomes a GFM table with the right number of
   columns.
5. A scanned PDF with `-ocr auto` and `tesseract` installed produces text; the
   same run without `tesseract` warns and exits 0 with placeholders.
6. `-images` writes image files and the `.md` links resolve relative to the
   `.md` file.
7. Directory mode converts every `.pdf` and prints the summary line.
8. The binary builds with `CGO_ENABLED=0` and passes `make audit`.
9. Without `mutool`, `-backend auto` warns and converts with `native`;
   `-backend mutool` exits 2 with a message naming the missing tool.

## 14. Future work

- Tables using ruling lines, merged cells, and tables that span pages
- Formula detection and LaTeX output
- Footnote linking (`[^1]`)
- Table of contents generation from the outline
- Layout-model backend (ONNX, reusing the `rmbg` runtime code) for hard layouts
- Recursive directory mode
