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
	"sort"
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
	target, err := filepath.Abs(*out)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		log.Fatal(err)
	}
	buildDir, err := os.MkdirTemp(filepath.Dir(target), ".output-build-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(buildDir)

	var built []sheet.Worksheet

	for _, w := range sheet.All() {
		if !matches(w, filters) {
			continue
		}
		dir := filepath.Join(buildDir, w.Subject, w.Name)
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

	// The index page and manifest always cover every known worksheet. The
	// server reads the manifest at request time, so worksheet additions and
	// metadata changes do not require rebuilding the server binary.
	worksheets := siteWorksheets(sheet.All())
	idx := filepath.Join(buildDir, "index.html")
	page := site.Index(site.Data{Worksheets: worksheets, Static: true})
	if err := os.WriteFile(idx, []byte(page), 0o644); err != nil {
		log.Fatal(err)
	}
	if err := site.WriteManifest(filepath.Join(buildDir, site.ManifestName), worksheets); err != nil {
		log.Fatal(err)
	}
	if err := publishOutput(buildDir, target); err != nil {
		log.Fatal(err)
	}
	fmt.Println(filepath.Join(target, "index.html"))
}

func publishOutput(buildDir, target string) error {
	var files []string
	if err := filepath.WalkDir(buildDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			rel, err := filepath.Rel(buildDir, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	}); err != nil {
		return err
	}
	// Publish the two catalogs last. New worksheet links cannot become visible
	// before all of their generated files are in place; the live-server manifest
	// is the final commit point.
	rank := func(path string) int {
		switch path {
		case "index.html":
			return 1
		case site.ManifestName:
			return 2
		default:
			return 0
		}
	}
	sort.Slice(files, func(i, j int) bool {
		ri, rj := rank(files[i]), rank(files[j])
		if ri != rj {
			return ri < rj
		}
		return files[i] < files[j]
	})
	for _, rel := range files {
		src := filepath.Join(buildDir, rel)
		dst := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("publish %s: %w", rel, err)
		}
	}
	return nil
}

func siteWorksheets(in []sheet.Worksheet) []site.Worksheet {
	out := make([]site.Worksheet, 0, len(in))
	for _, w := range in {
		out = append(out, site.Worksheet{
			Subject: w.Subject,
			Name:    w.Name,
			Title:   w.Title,
			Meta:    w.Meta,
		})
	}
	return out
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
