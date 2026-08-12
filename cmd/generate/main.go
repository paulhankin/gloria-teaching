// Command generate builds every worksheet into output/<subject>/<name>/.
//
//	go run ./cmd/generate            # HTML + PDF for all worksheets
//	go run ./cmd/generate -pdf=false # HTML only (fast)
//	go run ./cmd/generate venn       # only matching worksheets (substring)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"learningmaterial/internal/pdf"
	"learningmaterial/internal/sheet"
	"learningmaterial/internal/site"

	_ "learningmaterial/generate/math/answer_checks"
	_ "learningmaterial/generate/math/ordinal_numbers"
	_ "learningmaterial/generate/math/price_puzzles"
	_ "learningmaterial/generate/math/venn_diagrams"
)

func main() {
	out := flag.String("out", "output", "output directory")
	makePDF := flag.Bool("pdf", true, "also produce PDFs")
	wait := flag.Duration("wait", 4*time.Second, "time to wait for JS rendering before the PDF export")
	flag.Parse()

	filters := flag.Args()
	var built []sheet.Worksheet

	for _, w := range sheet.All() {
		if !matches(w, filters) {
			continue
		}
		dir := filepath.Join(*out, w.Subject, w.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatal(err)
		}
		html := w.Build().HTML()
		htmlPath := filepath.Join(dir, "index.html")
		if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-34s %4d KB\n", htmlPath, len(html)/1024)

		if *makePDF {
			pdfPath := filepath.Join(dir, "index.pdf")
			err := pdf.Render(htmlPath, pdfPath, pdf.Options{
				Landscape: !w.Portrait, Wait: *wait,
			})
			if err != nil {
				log.Fatalf("%s: %v", pdfPath, err)
			}
			st, _ := os.Stat(pdfPath)
			fmt.Printf("%-34s %4d KB\n", pdfPath, st.Size()/1024)
		}
		built = append(built, w)
	}

	if len(built) == 0 {
		log.Fatal("no worksheet matches the filter")
	}

	// The index page always covers every known worksheet. This static copy
	// has no request UI; cmd/serve renders the interactive front page.
	idx := filepath.Join(*out, "index.html")
	page := site.Index(site.Data{Worksheets: sheet.All(), Static: true})
	if err := os.WriteFile(idx, []byte(page), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println(idx)
}

func matches(w sheet.Worksheet, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if strings.Contains(w.Path(), f) {
			return true
		}
	}
	return false
}
