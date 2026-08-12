// Package venn baut das Arbeitsblatt "Venn-Diagramme".
package venn

import (
	_ "embed"
	"fmt"
	"strings"

	"lernmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

type diagramm struct {
	Typ    string            `json:"type"`
	Box    string            `json:"box,omitempty"`
	Labels []string          `json:"labels"`
	Farben []string          `json:"colors"`
	Werte  map[string]string `json:"values"`
}

type aufgabe struct {
	Key      string
	Titel    string
	Story    string
	Fragen   []string
	Diagramm diagramm
	Loesung  string // fertiges HTML fuer den Loesungskasten
}

const css = `
  body { font-size: 13.5pt; }
  .spalten { display: flex; gap: 7mm; flex: 1 1 auto; min-height: 0; }
  .spalte { flex: 1 1 50%; min-width: 0; display: flex; }
  .spalte + .spalte { border-left: 2px dashed #d8dfe8; padding-left: 7mm; }
  .aufgabe { width: 100%; display: flex; flex-direction: column; }
  .aufgabe h2 { margin: 0 0 1.5mm; color: #e8548c; font: 700 20pt 'Caveat', cursive; }
  .story { margin: 0 0 2mm; }
  .bild { flex: 1 1 0; min-height: 0; position: relative; margin: 1mm 0 2mm; }
  .bild svg { position: absolute; top: 0; left: 0; width: 100%; height: 100%; }
  .text { flex: 0 0 auto; }
  ol.fragen { margin: 0; padding-left: 6mm; }
  ol.fragen li { margin-bottom: 2.8mm; }
  .rechnung { padding: 1mm 2mm; border: 1px dashed #bfe3cd; border-radius: 4px;
              background: #fff; font: 10pt 'Courier New', monospace; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Fach:  "mathe",
		Name:  "venn_diagramme",
		Titel: "Venn-Diagramme",
		Meta:  "Mathe, ca. 9 Jahre · 4 Aufgabenblätter + Lösungen",
		Build: build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Titel: "Mathe-Arbeitsblatt: Venn-Diagramme",
		CSS:   css,
		Rough: true,
	}

	var body strings.Builder
	for i := 0; i < len(aufgaben); i += 2 {
		var cols strings.Builder
		for _, a := range aufgaben[i:min(i+2, len(aufgaben))] {
			var fr strings.Builder
			for _, q := range a.Fragen {
				fmt.Fprintf(&fr, "            <li>%s</li>\n", q)
			}
			fmt.Fprintf(&cols, `    <div class="spalte">
      <div class="aufgabe">
        <h2>%s</h2>
        <p class="story">%s</p>
        <div class="bild" data-diagram="%s"></div>
        <div class="text">
          <ol class="fragen">
%s          </ol>
        </div>
      </div>
    </div>
`, a.Titel, a.Story, a.Key, fr.String())
		}
		if len(aufgaben)-i == 1 {
			cols.WriteString("    <div class=\"spalte\"></div>\n")
		}
		body.WriteString(sheet.Page(
			fmt.Sprintf("Venn-Diagramme &ndash; Blatt %d", i/2+1),
			fmt.Sprintf("  <div class=\"spalten\">\n%s  </div>\n", cols.String())))
	}

	boxen := make([]string, 0, len(aufgaben))
	for _, a := range aufgaben {
		boxen = append(boxen, fmt.Sprintf("    <div class=\"lbox\">%s</div>\n", a.Loesung))
	}
	body.WriteString(sheet.SolutionPage(boxen...))
	d.Body = body.String()

	diag := map[string]diagramm{}
	for _, a := range aufgaben {
		diag[a.Key] = a.Diagramm
	}
	d.Set("DIAGRAMS", diag)
	d.AddScript(renderJS)
	return d
}
