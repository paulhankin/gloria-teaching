// Package venndiagrams builds the "Venn-Diagramme" worksheet.
package venndiagrams

import (
	_ "embed"
	"fmt"
	"strings"

	"learningmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

type diagram struct {
	Kind   string            `json:"type"`
	Box    string            `json:"box,omitempty"`
	Labels []string          `json:"labels"`
	Colors []string          `json:"colors"`
	Values map[string]string `json:"values"`
}

type task struct {
	Key       string
	Title     string
	Story     string
	Questions []string
	Diagram   diagram
	Solution  string // ready-made HTML for the solution box
}

const css = `
  body { font-size: 13.5pt; }
  .columns { display: flex; gap: 7mm; flex: 1 1 auto; min-height: 0; }
  .column { flex: 1 1 50%; min-width: 0; display: flex; }
  .column + .column { border-left: 2px dashed #d8dfe8; padding-left: 7mm; }
  .task { width: 100%; display: flex; flex-direction: column; }
  .task h2 { margin: 0 0 1.5mm; color: #e8548c; font: 700 20pt 'Caveat', cursive; }
  .story { margin: 0 0 2mm; }
  .figure { flex: 1 1 0; min-height: 0; position: relative; margin: 1mm 0 2mm; }
  .figure svg { position: absolute; top: 0; left: 0; width: 100%; height: 100%; }
  .text { flex: 0 0 auto; }
  ol.questions { margin: 0; padding-left: 6mm; }
  ol.questions li { margin-bottom: 2.8mm; }
  .calculation { padding: 1mm 2mm; border: 1px dashed #bfe3cd; border-radius: 4px;
                 background: #fff; font: 10pt 'Courier New', monospace; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "venn_diagrams",
		Title:   "Venn-Diagramme",
		Date:    "12 Aug 2026",
		Meta:    "Ca. 9 Jahre · 4 Aufgabenblätter + Lösungen",
		Build:   build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Arbeitsblatt: Venn-Diagramme",
		CSS:   css,
		Rough: true,
	}

	var body strings.Builder
	for i := 0; i < len(tasks); i += 2 {
		var cols strings.Builder
		for _, t := range tasks[i:min(i+2, len(tasks))] {
			var qs strings.Builder
			for _, q := range t.Questions {
				fmt.Fprintf(&qs, "            <li>%s</li>\n", q)
			}
			fmt.Fprintf(&cols, `    <div class="column">
      <div class="task">
        <h2>%s</h2>
        <p class="story">%s</p>
        <div class="figure" data-diagram="%s"></div>
        <div class="text">
          <ol class="questions">
%s          </ol>
        </div>
      </div>
    </div>
`, t.Title, t.Story, t.Key, qs.String())
		}
		if len(tasks)-i == 1 {
			cols.WriteString("    <div class=\"column\"></div>\n")
		}
		body.WriteString(sheet.Page(
			fmt.Sprintf("Venn-Diagramme &ndash; Blatt %d", i/2+1),
			fmt.Sprintf("  <div class=\"columns\">\n%s  </div>\n", cols.String())))
	}

	boxes := make([]string, 0, len(tasks))
	for _, t := range tasks {
		boxes = append(boxes, fmt.Sprintf("    <div class=\"solution-box\">%s</div>\n", t.Solution))
	}
	body.WriteString(sheet.SolutionPage(boxes...))
	d.Body = body.String()

	diagrams := map[string]diagram{}
	for _, t := range tasks {
		diagrams[t.Key] = t.Diagram
	}
	d.Set("DIAGRAMS", diagrams)
	d.AddScript(renderJS)
	return d
}
