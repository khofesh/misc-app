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
