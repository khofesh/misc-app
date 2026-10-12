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

### NVIDIA install (Fedora)

1. Install the NVIDIA driver and CUDA 12 from the CUDA repo:

   ```shell
   sudo dnf install -y dnf-plugins-core
   sudo dnf config-manager addrepo --from-repofile=https://developer.download.nvidia.com/compute/cuda/repos/fedora39/x86_64/cuda-fedora39.repo
   sudo dnf install -y cuda-toolkit-12-8
   ```

2. Install cuDNN 9 for CUDA 12 and make sure its libraries are on
   `LD_LIBRARY_PATH`.

3. Download the GPU build of ONNX Runtime:

   ```shell
   make ort/gpu
   ```

4. Run with `-device cuda` (or `-device auto`). The CUDA and cuDNN libraries
   must be on `LD_LIBRARY_PATH`, and ONNX Runtime's `libonnxruntime_providers_*`
   files sit next to `libonnxruntime.so` in `./lib`.

The CUDA/cuDNN versions must match the ONNX Runtime release pinned in the
Makefile (`ORT_VERSION`). `-device auto` falls back to CPU with a warning when
CUDA is unavailable; `-device cuda` fails with exit code 2 instead.

