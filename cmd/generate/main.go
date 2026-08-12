// Kommando generate baut alle Arbeitsblaetter nach output/<fach>/<name>/.
//
//	go run ./cmd/generate            # HTML + PDF fuer alle Blaetter
//	go run ./cmd/generate -pdf=false # nur HTML (schnell)
//	go run ./cmd/generate venn       # nur passende Blaetter (Teilstring)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lernmaterial/internal/pdf"
	"lernmaterial/internal/sheet"

	_ "lernmaterial/generate/mathe/preisraetsel"
	_ "lernmaterial/generate/mathe/venn_diagramme"
	_ "lernmaterial/generate/mathe/zahlenfolgen"
)

func main() {
	out := flag.String("out", "output", "Ausgabeverzeichnis")
	makePDF := flag.Bool("pdf", true, "PDF mit erzeugen")
	wait := flag.Duration("wait", 4*time.Second, "Wartezeit fuers JS-Rendern vor dem PDF-Export")
	flag.Parse()

	filters := flag.Args()
	var gebaut []sheet.Worksheet

	for _, w := range sheet.All() {
		if !matches(w, filters) {
			continue
		}
		dir := filepath.Join(*out, w.Fach, w.Name)
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
		gebaut = append(gebaut, w)
	}

	if len(gebaut) == 0 {
		log.Fatal("kein Arbeitsblatt passt zum Filter")
	}

	// Startseite immer ueber alle bekannten Blaetter.
	idx := filepath.Join(*out, "index.html")
	if err := os.WriteFile(idx, []byte(indexHTML(sheet.All())), 0o644); err != nil {
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
