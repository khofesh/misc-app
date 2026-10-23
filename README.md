# Miscellaneous apps for daily problems

## html2md

convert html `input/input.html` to markdown `output/output.md`

## decode-encode

decode or encode a string

input for encode: `input/encode.txt`

output for encode: `output/encoded.txt`

input for decode: `input/decode.txt`

output for decode: `output/decoded.txt`

## pdf-ocr

requirements

```shell
sudo dnf install tesseract
```

## html2text

convert html `input/input.html` to markdown `output/html2text-result.txt`

## pdf2md

Convert PDF documents to Markdown. Keeps headings, paragraphs, lists, bold and
italic text, links, code blocks and simple tables, reads multi-column pages in
column order, and drops repeated page headers and footers. Scanned pages go
through OCR.

```shell
sudo dnf install mupdf tesseract tesseract-langpack-ind   # mupdf recommended, tesseract for OCR

make install                      # build and install into ~/.local/bin

pdf2md -i report.pdf                          # Markdown to stdout
pdf2md -i report.pdf -o report.md
pdf2md -i pdfs/ -o mds/                       # whole directory
pdf2md -i book.pdf -o book.md -pages 1-5,9 -images
pdf2md -i scan.pdf -o scan.md -ocr always -lang ind+eng
```

`~/.local/bin` must be on `PATH` (it is by default on Fedora). Use
`make install BINDIR=/some/dir` to install elsewhere, and
`make uninstall/pdf2md` to remove it.

Text is extracted with MuPDF's `mutool`. Without it, a pure Go backend is used
(`-backend native`), which handles fewer PDFs. Run `pdf2md -h` for all flags.

## rmbg

Remove the background from images with a local ONNX segmentation model. Runs
on CPU, and on NVIDIA GPUs when the CUDA ONNX Runtime build is used.

```shell
make ort/cpu                      # download ONNX Runtime into ./lib (CPU)
make build/rmbg                   # build ./bin/rmbg

./bin/rmbg -i input/photo.jpg -o output/photo.png
./bin/rmbg -i input/photos/ -o output/photos/     # whole directory
./bin/rmbg -i photo.jpg -o mask.png -mask-only    # alpha mask only
./bin/rmbg -i photo.jpg -o out.jpg -bg "#ffffff"  # solid background
```

The first run downloads the model into `$XDG_CACHE_HOME/rmbg` (default
`~/.cache/rmbg`) and verifies its SHA-256.

Models: `u2net` (default), `u2netp`, `u2net_human_seg`, `isnet-general-use`.
Pass `-model /path/to/model.onnx` to use a custom model.

### CPU install (any x86_64 Linux)

```shell
make ort/cpu
```

This unpacks `libonnxruntime.so` into `./lib`, next to the binary. If you
install it elsewhere, point the tool at it with `-ort-lib` or the
`ONNXRUNTIME_LIB` environment variable.

### NVIDIA install (Fedora 44)

ONNX Runtime 1.31 GPU builds target CUDA 13.x (CUDA 13.0 or newer, so the
CUDA 13.4 toolkit works) and require cuDNN 9.

1. Install the NVIDIA driver, then the CUDA 13.4 toolkit from the Fedora 44
   CUDA repo:

   ```shell
   sudo dnf install -y dnf-plugins-core
   sudo dnf config-manager addrepo \
     --from-repofile=https://developer.download.nvidia.com/compute/cuda/repos/fedora44/x86_64/cuda-fedora44.repo
   sudo dnf install -y cuda-toolkit-13-4
   ```

   Add CUDA to your environment:

   ```shell
   export PATH=/usr/local/cuda/bin:$PATH
   export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
   ```

2. Install cuDNN 9 for CUDA 13 (not shipped in the CUDA repo). Download the
   archive and put its libraries on `LD_LIBRARY_PATH`:

   ```shell
   curl -fLO https://developer.download.nvidia.com/compute/cudnn/redist/cudnn/linux-x86_64/cudnn-linux-x86_64-9.27.0.42_cuda13-archive.tar.xz
   sudo tar -xJf cudnn-linux-x86_64-9.27.0.42_cuda13-archive.tar.xz -C /usr/local
   export LD_LIBRARY_PATH=/usr/local/cudnn-linux-x86_64-9.27.0.42_cuda13/lib:$LD_LIBRARY_PATH
   ```

3. Download the CUDA 13 build of ONNX Runtime:

   ```shell
   make ort/gpu              # ORT_CUDA=cuda13 by default; ORT_CUDA=cuda12 for CUDA 12.8
   ```

4. Run with `-device cuda` (or `-device auto`). The CUDA and cuDNN libraries
   must be on `LD_LIBRARY_PATH`, and ONNX Runtime's `libonnxruntime_providers_*`
   files sit next to `libonnxruntime.so` in `./lib`.

The CUDA, cuDNN, and ONNX Runtime versions must match the release pinned in the
Makefile (`ORT_VERSION`, `ORT_CUDA`). `-device auto` falls back to CPU with a
warning when CUDA is unavailable; `-device cuda` fails with exit code 2 instead.

If CUDA is selected but inference fails with
`CUDNN_STATUS_SUBLIBRARY_VERSION_MISMATCH`, more than one cuDNN 9 release is
visible and they are being mixed. Keep the loader and all `libcudnn_*.so.9`
sublibraries from a single cuDNN release, with that directory **before** any
other directory that ships its own cuDNN (for example an ArrayFire install):
`LD_LIBRARY_PATH=/opt/cudnn/current/lib:/usr/local/cuda/lib64`.
`make run/rmbg` prepends `CUDNN_LIB_DIRS` and `CUDA_LIB_DIRS` (defaults
`/opt/cudnn/current/lib` and `/usr/local/cuda/lib64`) for this reason; override
them to point at your install.

