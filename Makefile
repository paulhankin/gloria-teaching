.PHONY: all html serve build clean

# Build everything (HTML + PDF) into output/
all:
	go run ./cmd/generate

# HTML only (fast, no headless Chrome)
html:
	go run ./cmd/generate -pdf=false

# Binaries into bin/
build:
	go build -o bin/generate ./cmd/generate
	go build -o bin/serve ./cmd/serve

# Serve locally (password from SITE_PASSWORD)
serve:
	go run ./cmd/serve -addr :8000 -dir output

clean:
	rm -rf output bin
