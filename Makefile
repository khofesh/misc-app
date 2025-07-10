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

# ==================================================================================== #
# BUILD 
# ==================================================================================== #

## build/utc2local: build the cmd/utc2local application
.PHONY: build/utc2local
build/utc2local:
	@echo 'Building cmd/utc2local...'
	go build -ldflags="-s -w" -gcflags=all=-l -o=./bin/utc2local ./cmd/utc2local
	GOOS=linux go build -ldflags="-s -w" -gcflags=all=-l -o=./bin/linux/utc2local ./cmd/utc2local

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
