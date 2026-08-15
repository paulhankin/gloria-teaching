// Package negativenumbers builds an introductory worksheet about negative numbers.
package negativenumbers

import (
	_ "embed"
	"fmt"
	"strings"

	"learningmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

const css = `
  body { font-size: 14pt; }
  .number-line-panel { flex: 0 0 42mm; min-height: 0; margin-bottom: 2.5mm;
                       padding: 1.5mm 3mm 0; border: 2px dashed #d8dfe8;
                       border-radius: 10px; background: #fffdf8; }
  .number-line-title { margin: 0; text-align: center; color: #7a869a; font-size: 11.5pt; }
  .number-line { position: relative; width: 100%; height: 33mm; }
  .number-line svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .tip { flex: 0 0 auto; margin: 0 0 2.5mm; padding: 1.5mm 3mm;
         border-radius: 8px; background: #f5fff9; border: 2px dashed #bfe3cd; }
  .tip b { color: #2e8b57; }
  .groups { display: flex; flex-direction: column; gap: 2.5mm; flex: 1 1 auto; min-height: 0; }
  .task-group { flex: 1 1 0; min-height: 0; padding: 1.5mm 3mm 2mm;
                border: 2px dashed #d8dfe8; border-radius: 9px; }
  .task-group:nth-child(2) { border-color: #cfe2f3; }
  .task-group:nth-child(3) { border-color: #efd6df; }
  .group-heading { display: flex; align-items: baseline; gap: 4mm; margin-bottom: 1mm; }
  .group-heading h2 { margin: 0; color: #e8548c; font: 700 18pt 'Caveat', cursive; }
  .group-heading p { margin: 0; color: #7a869a; font-size: 10.5pt; }
  .task-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 2mm 8mm; }
  .equation { white-space: nowrap; font-size: 17pt; line-height: 1.35; }
  .equation .blank { min-width: 18mm; margin-left: 1.5mm; }
  .solution-box .answer-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1mm 4mm; }
  .solution-box .answer-grid p { margin: 0; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "negative_numbers",
		Title:   "Negative Zahlen",
		Date:    "15 Aug 2026",
		Meta:    "Primarschule / Unterstufe · einfache Minusaufgaben mit Zahlenstrahl · 1 Aufgabenblatt + Lösungen",
		Build:   build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Arbeitsblatt: Negative Zahlen",
		CSS:   css,
		Rough: true,
	}

	var groups strings.Builder
	var solutionBoxes []string
	for _, group := range taskGroups {
		var equations, answers strings.Builder
		for i, task := range group.Tasks {
			fmt.Fprintf(&equations, "      <div class=\"equation\">%d &minus; %d = <span class=\"blank\"></span></div>\n",
				task.Minuend, task.Subtrahend)
			fmt.Fprintf(&answers, "        <p>%c) %d &minus; %d = <b>%d</b></p>\n",
				'a'+i, task.Minuend, task.Subtrahend, task.result())
		}
		fmt.Fprintf(&groups, `    <section class="task-group">
      <div class="group-heading"><h2>%s</h2><p>%s</p></div>
      <div class="task-grid">
%s      </div>
    </section>
`, group.Title, group.Hint, equations.String())
		solutionBoxes = append(solutionBoxes, sheet.SolutionBox(group.Title,
			"      <div class=\"answer-grid\">\n"+answers.String()+"      </div>\n"))
	}

	content := fmt.Sprintf(`  <div class="number-line-panel">
    <p class="number-line-title">Der Zahlenstrahl: Links von der 0 liegen die negativen Zahlen.</p>
    <div class="number-line" data-number-line></div>
  </div>
  <p class="tip"><b>So geht es:</b> Starte bei der ersten Zahl. Gehe für die zweite Zahl nach links. Beispiel: 4 &minus; 5 = &minus;1.</p>
  <div class="groups">
%s  </div>
`, groups.String())

	d.Body = sheet.Page("Negative Zahlen &ndash; Minusaufgaben", content)
	d.Solutions = sheet.SolutionPage(solutionBoxes...)
	d.Set("NUMBER_LINE", map[string]int{"min": -15, "max": 15})
	d.AddScript(renderJS)
	return d
}
