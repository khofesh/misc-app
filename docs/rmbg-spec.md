# rmbg — Background Removal CLI (Spec)

Status: Draft
Location: `cmd/rmbg`, `internal/rmbg`

## 1. Goal

A command-line tool written in Go that removes the background from images
using a local segmentation model (ONNX). It must run on:

- **CPU**: any x86_64 Linux machine, with no GPU required
- **NVIDIA GPU**: CUDA, used when it is available

Both modes use **the same binary**. The device is chosen at runtime.

### Non-goals (v1)

- Video or webcam input
- AMD (ROCm), Intel (OpenVINO), or Apple (CoreML) acceleration
- TensorRT execution provider (possible later, see §11)
- A GUI or HTTP server
- Model training or fine-tuning

## 2. Usage

```shell
# single file
rmbg -i input/photo.jpg -o output/photo.png

# whole directory (non-recursive), output keeps the same base names as .png
rmbg -i input/photos/ -o output/photos/

# force GPU, pick a smaller model
rmbg -i photo.jpg -o out.png -device cuda -model u2netp

# output only the mask (grayscale PNG)
rmbg -i photo.jpg -o mask.png -mask-only

# replace the background with a solid color instead of transparency
rmbg -i photo.jpg -o out.jpg -bg "#ffffff"
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-i` | (required) | Input file or directory |
| `-o` | (required) | Output file or directory. Created if it doesn't exist |
| `-model` | `u2net` | Model name from the registry (§5) or a path to a `.onnx` file |
| `-device` | `auto` | `auto`, `cpu`, or `cuda` (see §4) |
| `-gpu-id` | `0` | CUDA device index |
| `-ort-lib` | (see §4.3) | Path to `libonnxruntime.so` |
| `-workers` | `1` | Number of images pre- and post-processed in parallel (directory mode) |
| `-threads` | `0` | ONNX Runtime intra-op threads on CPU. `0` lets ONNX Runtime decide |
| `-mask-only` | `false` | Write the alpha mask instead of the cut-out image |
| `-bg` | (empty) | Fill the background with a hex color instead of making it transparent |
| `-alpha-threshold` | `0` | `0` keeps a soft alpha. `1`–`254` turns the mask into a binary mask at that level |
| `-overwrite` | `false` | Overwrite existing output files |
| `-v` | `false` | Verbose logging (shows the device chosen and timing per image) |

### Exit codes

| Code | Meaning |
|---|---|
| 0 | All images processed |
| 1 | Invalid flags or input |
| 2 | Runtime or model failed to load, or `-device cuda` was asked for but is unavailable |
| 3 | Some images in directory mode failed (others succeeded) |

## 3. Architecture

```
cmd/rmbg/main.go          flag parsing, logging (zerolog), file walking
internal/rmbg/
  runtime.go              ORT init, device selection, session creation
  models.go               model registry, download, and cache
  preprocess.go           decode → resize → normalize → NCHW float32 tensor
  postprocess.go          mask → resize to original size → apply alpha / bg
  remover.go              Remover type tying everything together
```

### Core API

```go
type Device string

const (
    DeviceAuto Device = "auto"
    DeviceCPU  Device = "cpu"
    DeviceCUDA Device = "cuda"
)

type Options struct {
    Model      string // registry name or path to .onnx
    Device     Device
    GPUID      int
    ORTLibPath string
    Threads    int
}

type Remover struct { /* session, model spec, active device */ }

func New(opts Options) (*Remover, error)
func (r *Remover) Device() Device                  // the device actually used
func (r *Remover) Mask(img image.Image) (*image.Gray, error)
func (r *Remover) Remove(img image.Image) (*image.NRGBA, error)
func (r *Remover) Close() error
```

### Pipeline

1. **Decode.** Supports `image/jpeg`, `image/png`, and `golang.org/x/image/webp`.
   Apply the EXIF orientation for JPEG.
2. **Preprocess.** Resize to the model's input size (bilinear, using
   `golang.org/x/image/draw`). Convert to float32 RGB in `[0,1]`, normalize
   with the model's mean and std, and lay out as `[1,3,H,W]`.
3. **Inference.** Run the ONNX Runtime session and read the first output
   `[1,1,H,W]`.
4. **Postprocess.** Min-max normalize the output to `[0,1]`, resize it to the
   original image size (bilinear), and convert to an 8-bit alpha. Apply the
   threshold if one is set.
5. **Compose.** Write the alpha into an `NRGBA` copy of the original image, or
   blend the image over the `-bg` color.
6. **Encode.** Write PNG (or JPEG only when `-bg` is set and the output ends in
   `.jpg`). Writing a transparent image to `.jpg` is an error.

## 4. CPU and NVIDIA support

### 4.1 Library

Use [`github.com/yalue/onnxruntime_go`](https://github.com/yalue/onnxruntime_go).
It needs cgo and loads the ONNX Runtime shared library at runtime.

- **CPU**: the standard ONNX Runtime build (`onnxruntime-linux-x64-<ver>.tgz`)
- **NVIDIA**: the GPU build (`onnxruntime-linux-x64-gpu-<ver>.tgz`). It also
  runs on CPU, so a machine with the GPU build can use either device.

The Go binary is the same in both cases. Only the `.so` it loads is different.

### 4.2 Device selection

| `-device` | Behavior |
|---|---|
| `cpu` | Create the session with the default CPU execution provider |
| `cuda` | Add the CUDA execution provider. If that fails, exit with code 2 and show the ONNX Runtime error |
| `auto` | Try CUDA first. If it fails, log a warning with the reason and fall back to CPU |

Implementation outline:

```go
so, _ := ort.NewSessionOptions()
defer so.Destroy()

if dev == DeviceCUDA || dev == DeviceAuto {
    cuda, err := ort.NewCUDAProviderOptions()
    if err == nil {
        defer cuda.Destroy()
        err = cuda.Update(map[string]string{"device_id": strconv.Itoa(gpuID)})
    }
    if err == nil {
        err = so.AppendExecutionProviderCUDA(cuda)
    }
    if err != nil {
        if dev == DeviceCUDA {
            return fmt.Errorf("cuda unavailable: %w", err)
        }
        log.Warn().Err(err).Msg("cuda unavailable, falling back to cpu")
        // rebuild options without CUDA
    }
}
if threads > 0 {
    so.SetIntraOpNumThreads(threads)
}
```

The device actually used is logged at startup:
`device=cuda gpu=0 model=u2net` or `device=cpu threads=auto model=u2net`.

### 4.3 Finding the shared library

Search in this order, using the first match:

1. `-ort-lib` flag
2. `ONNXRUNTIME_LIB` environment variable
3. `./lib/libonnxruntime.so` next to the binary
4. `libonnxruntime.so` on the system loader path

If none is found, exit with code 2 and a message that links to the install
steps in the README.

### 4.4 NVIDIA requirements

- NVIDIA driver, CUDA, and cuDNN versions that match the ONNX Runtime GPU
  release in use. See the ONNX Runtime CUDA EP requirements table.
  For example, ORT 1.2x GPU builds target CUDA 12.x and cuDNN 9.
- The CUDA and cuDNN libraries must be on `LD_LIBRARY_PATH`. The ONNX Runtime
  CUDA provider (`libonnxruntime_providers_cuda.so`,
  `libonnxruntime_providers_shared.so`) must be in the same directory as
  `libonnxruntime.so`.
- The README documents installation on Fedora (`dnf`, via the NVIDIA CUDA repo).

### 4.5 Performance expectations (to verify in benchmarks)

- Create one session per process and reuse it for every image. Never create a
  session per image.
- ONNX Runtime sessions are safe for concurrent `Run` calls. Even so, v1
  serializes inference through one goroutine and runs decode, preprocess, and
  postprocess in parallel across `-workers`.
- The first CUDA run has warm-up cost. Benchmarks exclude it.

## 5. Model registry

| Name | Input | Size | Mean / Std | License | Notes |
|---|---|---|---|---|---|
| `u2net` (default) | 320×320 | ~176 MB | ImageNet¹ | Apache-2.0 | General purpose |
| `u2netp` | 320×320 | ~4.7 MB | ImageNet¹ | Apache-2.0 | Fast, lower quality |
| `u2net_human_seg` | 320×320 | ~176 MB | ImageNet¹ | Apache-2.0 | People |
| `isnet-general-use` | 1024×1024 | ~179 MB | 0.5 / 1.0 | Apache-2.0 | Sharper edges, slower on CPU |

¹ mean `(0.485, 0.456, 0.406)`, std `(0.229, 0.224, 0.225)`

Each registry entry stores the name, download URL, SHA-256, input size, mean
and std, and input and output tensor names.

- **Source**: the `rembg` model releases on GitHub. Pin the URLs and SHA-256
  sums when implementing.
- **Cache**: `$XDG_CACHE_HOME/rmbg/` (default `~/.cache/rmbg/`). Download on
  first use, show progress on stderr, and verify the SHA-256. Delete the file
  and fail if the hash doesn't match.
- **Custom path**: `-model /path/to/model.onnx` skips the registry. Input and
  output names and shapes are read from the model. The default is ImageNet
  normalization; v1 has no flag to change it.
- BRIA RMBG-1.4/2.0 is **not** bundled because it is non-commercial only.
  Users can still load it with a custom path.

## 6. Error handling

- Unsupported or corrupt image: in single-file mode, exit 1. In directory mode,
  log the error, skip the file, and exit 3 at the end.
- Output exists and `-overwrite` isn't set: skip with a warning.
- Every error message names the file it relates to.

## 7. Logging

Use `github.com/rs/zerolog`, which the repo already uses, with console output
on stderr. Normal mode logs one line per image. `-v` adds timings for decode,
preprocess, inference, and postprocess.

## 8. Build and Makefile

cgo is required (`CGO_ENABLED=1`).

```make
## run/rmbg: run rmbg on a sample image
run/rmbg:
	go run ./cmd/rmbg -i input/sample.jpg -o output/sample.png -v

## build/rmbg: build the cmd/rmbg application
build/rmbg:
	CGO_ENABLED=1 go build -ldflags="-s -w" -o=./bin/rmbg ./cmd/rmbg

## build/debug/rmbg
build/debug/rmbg:
	CGO_ENABLED=1 go build -gcflags=all="-N -l" -o=./bin/rmbg-debug ./cmd/rmbg

## ort/cpu, ort/gpu: download ONNX Runtime into ./lib
ort/cpu:  # fetch onnxruntime-linux-x64-$(ORT_VERSION).tgz → ./lib
ort/gpu:  # fetch onnxruntime-linux-x64-gpu-$(ORT_VERSION).tgz → ./lib
```

`ORT_VERSION` is pinned in the Makefile and must match the version that the
`onnxruntime_go` release supports.

`lib/` and the model cache are added to `.gitignore`.

## 9. Testing

- **Unit tests (no ONNX Runtime needed)**: preprocess tensor layout and
  normalization values, postprocess resize and threshold, alpha composition,
  `-bg` blending, hex color parsing, and output path mapping for directory mode.
- **Integration tests (`//go:build integration`)**: use `u2netp` on a small
  fixture image on CPU. Check that the output size matches the input and that
  the corners are mostly transparent while the center is mostly opaque.
- **GPU tests (`//go:build cuda`)**: run the same fixture on CUDA. The mask
  must be within a small mean absolute difference (≤ 2/255) of the CPU result.
  These are run manually on a machine with a GPU.
- **Benchmark**: `BenchmarkRemove` per model and device, reporting ms per image.

## 10. Acceptance criteria

1. `rmbg -i x.jpg -o x.png -device cpu` works on a machine with no NVIDIA
   driver, using the CPU ONNX Runtime build.
2. `-device cuda` on a machine with an NVIDIA GPU and the GPU ONNX Runtime
   build runs inference on the GPU (visible in `nvidia-smi`) and logs
   `device=cuda`.
3. `-device auto` on a machine without a GPU falls back to CPU with a warning
   and exits 0.
4. `-device cuda` without a working CUDA setup exits with code 2 and a clear
   message.
5. CPU and CUDA masks for the same input are within the tolerance in §9.
6. Directory mode processes every supported image and reports a summary
   (`ok=N failed=M skipped=K`).
7. The README has a section with install steps for CPU and NVIDIA.

## 11. Future work

- TensorRT execution provider (`-device tensorrt`) for faster NVIDIA inference
- Fallback that needs no model: flood-fill from the corners for plain backgrounds
- Batched inference (`N>1`) on GPU for directory mode
- Edge refinement with alpha matting
- More execution providers: ROCm, OpenVINO, CoreML
