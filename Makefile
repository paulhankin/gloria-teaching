.DEFAULT_GOAL := all

.PHONY: all html serve build clean prepare

# Discover worksheet packages in the local per-user repositories.
prepare:
	go run ./cmd/importworksheets

# Build everything (HTML + PDF) into output/
all: prepare
	go run ./cmd/generate

# HTML only (fast, no headless Chrome)
html: prepare
	go run ./cmd/generate -pdf=false

# Binaries into bin/
build: prepare
	go build -o bin/generate ./cmd/generate
	go build -o bin/serve ./cmd/serve

# Serve locally with account sign-in
serve: html
	go run ./cmd/serve -addr :8000 -dir output

clean:
	rm -rf output bin
