.PHONY: all html serve build clean

# Alles bauen (HTML + PDF) nach output/
all:
	go run ./cmd/generate

# Nur HTML (schnell, ohne headless Chrome)
html:
	go run ./cmd/generate -pdf=false

# Binaries nach bin/
build:
	go build -o bin/generate ./cmd/generate
	go build -o bin/serve ./cmd/serve

# Lokal ausliefern (Passwort aus SITE_PASSWORD)
serve:
	go run ./cmd/serve -addr :8000 -dir output

clean:
	rm -rf output bin
