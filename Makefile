# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: run html2md
.PHONY: run/html2md
run/html2md:
	go run ./cmd/html2md 

## run: run encode
.PHONY: run/encode
run/encode:
	go run ./cmd/decode-encode -encode

## run: run decode
.PHONY: run/decode
run/decode:
	go run ./cmd/decode-encode -decode

## run: run pdf-ocr
.PHONY: run/pdf-ocr
run/pdf-ocr:
	go run ./cmd/pdf-ocr

## run: run html2text
.PHONY: run/html2text
run/html2text:
	go run ./cmd/html2text

## run: run utc2local
.PHONY: run/utc2local
run/utc2local:
	go run ./cmd/utc2local -utc="2025-07-09T00:00:00Z"

## run: run http-middleware
.PHONY: run/http-middleware
run/http-middleware:
	go run ./cmd/examples/http-middleware

## run: run spinner
.PHONY: run/spinner
run/spinner:
	go run ./cmd/spinner

## run/rmbg: run rmbg on a sample image
# Directories searched before the inherited LD_LIBRARY_PATH. Override these to
# point at the CUDA toolkit and a single cuDNN 9 install (keep the whole cuDNN
# set from one release together, or ONNX Runtime aborts with a sublibrary
# version mismatch).
CUDA_LIB_DIRS ?= /usr/local/cuda/lib64
CUDNN_LIB_DIRS ?= /opt/cudnn/current/lib

.PHONY: run/rmbg
run/rmbg:
	LD_LIBRARY_PATH=$(CUDNN_LIB_DIRS):$(CUDA_LIB_DIRS):$$LD_LIBRARY_PATH \
	ONNXRUNTIME_LIB=$(CURDIR)/lib/libonnxruntime.so \
	go run ./cmd/rmbg -i input/sample.png -o output/sample.png -v

## run/pdf2md: run pdf2md on a sample pdf
.PHONY: run/pdf2md
run/pdf2md:
	go run ./cmd/pdf2md -i input/sample.pdf -o output/sample.md -v

# ==================================================================================== #
# BUILD DEBUG
# ==================================================================================== #

## build/debug/html2md: build the api with debugging flags enabled
.PHONY: build/debug/html2md
build/debug/html2md:
	@echo 'Building cmd/html2md...'
	go build -gcflags=all="-N -l" -o=./bin/html2md-debug ./cmd/html2md

## build/debug/decode-encode: build the api with debugging flags enabled
.PHONY: build/debug/decode-encode
build/debug/decode-encode:
	@echo 'Building cmd/decode-encode...'
	go build -gcflags=all="-N -l" -o=./bin/decode-encode-debug ./cmd/decode-encode

## build/debug/pdf-ocr: build the api with debugging flags enabled
.PHONY: build/debug/pdf-ocr
build/debug/pdf-ocr:
	@echo 'Building cmd/pdf-ocr...'
	go build -gcflags=all="-N -l" -o=./bin/pdf-ocr-debug ./cmd/pdf-ocr

## build/debug/html2text: build the api with debugging flags enabled
.PHONY: build/debug/html2text
build/debug/html2text:
	@echo 'Building cmd/html2text...'
	go build -gcflags=all="-N -l" -o=./bin/html2text-debug ./cmd/html2text

## build/debug/utc2local: build the api with debugging flags enabled
.PHONY: build/debug/utc2local
build/debug/utc2local:
	@echo 'Building cmd/utc2local...'
	go build -gcflags=all="-N -l" -o=./bin/utc2local-debug ./cmd/utc2local

## build/debug/rmbg: build the cmd/rmbg application with debugging flags enabled
.PHONY: build/debug/rmbg
build/debug/rmbg:
	@echo 'Building cmd/rmbg...'
	CGO_ENABLED=1 go build -gcflags=all="-N -l" -o=./bin/rmbg-debug ./cmd/rmbg

## build/debug/pdf2md: build the cmd/pdf2md application with debugging flags enabled
.PHONY: build/debug/pdf2md
build/debug/pdf2md:
	@echo 'Building cmd/pdf2md...'
	go build -gcflags=all="-N -l" -o=./bin/pdf2md-debug ./cmd/pdf2md

# ==================================================================================== #
# BUILD 
# ==================================================================================== #

## build/utc2local: build the cmd/utc2local application
.PHONY: build/utc2local
build/utc2local:
	@echo 'Building cmd/utc2local...'
	go build -ldflags="-w" -gcflags=all=-l -o=./bin/utc2local ./cmd/utc2local
	GOOS=linux go build -ldflags="-w" -gcflags=all=-l -o=./bin/linux/utc2local ./cmd/utc2local

# ONNX Runtime version. Must match the C API version the pinned
# onnxruntime_go release supports (it is built against ORT 1.29 headers, and
# ORT keeps the older C API available in newer releases).
ORT_VERSION ?= 1.31.0
# GPU build flavour: cuda13 (CUDA 13.x + cuDNN 9) or cuda12 (CUDA 12.8 + cuDNN 9).
ORT_CUDA ?= cuda13

## build/rmbg: build the cmd/rmbg application (requires ./lib/libonnxruntime.so at runtime)
.PHONY: build/rmbg
build/rmbg:
	@echo 'Building cmd/rmbg...'
	CGO_ENABLED=1 go build -ldflags="-w" -o=./bin/rmbg ./cmd/rmbg

## build/pdf2md: build the cmd/pdf2md application
.PHONY: build/pdf2md
build/pdf2md:
	@echo 'Building cmd/pdf2md...'
	CGO_ENABLED=0 go build -ldflags="-w" -o=./bin/pdf2md ./cmd/pdf2md

# ==================================================================================== #
# INSTALL
# ==================================================================================== #

BINDIR ?= $(HOME)/.local/bin

## install/pdf2md: build pdf2md and install it into $(BINDIR)
.PHONY: install/pdf2md
install/pdf2md: build/pdf2md
	install -Dm755 ./bin/pdf2md $(BINDIR)/pdf2md

## uninstall/pdf2md: remove pdf2md from $(BINDIR)
.PHONY: uninstall/pdf2md
uninstall/pdf2md:
	rm -f $(BINDIR)/pdf2md

## install: install all installable tools into $(BINDIR)
.PHONY: install
install: install/pdf2md

## ort/cpu: download the ONNX Runtime CPU build into ./lib
.PHONY: ort/cpu
ort/cpu:
	@mkdir -p lib
	curl -fsSL https://github.com/microsoft/onnxruntime/releases/download/v$(ORT_VERSION)/onnxruntime-linux-x64-$(ORT_VERSION).tgz | tar -xz -C lib --strip-components=2

## ort/gpu: download the ONNX Runtime GPU build (ORT_CUDA=cuda13|cuda12) into ./lib
.PHONY: ort/gpu
ort/gpu:
	@mkdir -p lib
	curl -fsSL https://github.com/microsoft/onnxruntime/releases/download/v$(ORT_VERSION)/onnxruntime-linux-x64-gpu_$(ORT_CUDA)-$(ORT_VERSION).tgz | tar -xz -C lib --strip-components=2

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit: tidy
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor 
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy 
	go mod verify 
	@echo 'Vendoring dependencies...'
	go mod vendor

.PHONY: tidy 
tidy:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy 
	go mod verify 
