// Package pricepuzzles builds the "Preisrätsel" worksheet
// (total price, price difference, half, double, triple).
package pricepuzzles

import (
	_ "embed"
	"fmt"
	"strings"

	"learningmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

type task struct {
	Key      string
	Kind     string // motif for render.js
	Title    string
	Text     string
	Question string
	Names    [2]string
	Values   [2]int
}

const css = `
  .columns { display: flex; gap: 8mm; flex: 1 1 auto; min-height: 0; }
  .column { display: flex; flex-direction: column; flex: 1 1 50%; min-width: 0; }
  .column + .column { padding-left: 8mm; border-left: 2px dashed #d8dfe8; }
  h2 { margin: 0 0 3mm; color: #e8548c; font: 700 24pt 'Caveat', cursive; }
  h3 { margin: 0 0 2mm; color: #e8548c; font: 700 20pt 'Caveat', cursive; }
  .story { margin: 0 0 3mm; }
  .figure { flex: 0 0 auto; margin: 0 0 3mm; line-height: 0; }
  .figure svg { display: block; width: 100%; height: auto; }
  .question { margin: 0 0 3mm; font-weight: 700; }
  .answers { margin: 0 0 6mm; font-size: 15pt; }
  .answer { display: block; margin-bottom: 5mm; white-space: nowrap; }
  .blank { min-width: 34mm; border-bottom-color: #aab5c4; }
  .calculation { flex: 1 1 auto; min-height: 40mm; }
  .reasoning { flex: 1 1 auto; min-height: 55mm; }
  .writing-lines { margin-top: 1mm; }
  .solution-box p { font-size: 12.5pt; }
  .check { color: #65778a; font-size: 11.5pt !important; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "price_puzzles",
		Title:   "Preisrätsel",
		Meta:    "Mathe, Deutsch · 8 Aufgabenblätter + Lösungen · Summe, Differenz, Vielfache",
		Build:   build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Arbeitsblatt: Preisrätsel",
		CSS:   css,
		Rough: true,
	}

	var body strings.Builder
	for nr, t := range tasks {
		nr++
		content := fmt.Sprintf(`  <div class="columns">
    <div class="column">
      <h2>Aufgabe %d &nbsp; <small>%s</small></h2>
      <p class="story">%s</p>
      <div class="figure" data-price="%s"></div>
      <p class="question">%s</p>
      <div class="box calculation">Rechnung:</div>
    </div>
    <div class="column">
      <h3>Meine Antwort</h3>
      <div class="answers">
        <span class="answer">%s: <span class="blank"></span> Fr.</span>
        <span class="answer">%s: <span class="blank"></span> Fr.</span>
      </div>
      <h3>Warum meine Antwort stimmt</h3>
      <div class="box reasoning">
        %s
      </div>
    </div>
  </div>
`, nr, t.Title, t.Text, t.Key, t.Question, t.Names[0], t.Names[1], sheet.Lines(12))
		body.WriteString(sheet.Page(fmt.Sprintf("Preisrätsel &ndash; Blatt %d", nr), content))
	}

	var boxes []string
	for nr, t := range tasks {
		nr++
		x, y := t.Values[0], t.Values[1]
		boxes = append(boxes, sheet.SolutionBox(
			fmt.Sprintf("Aufgabe %d &ndash; %s", nr, t.Title),
			fmt.Sprintf("      <p>%s: <b>%d Franken</b> &nbsp; · &nbsp; %s: <b>%d Franken</b></p>\n",
				t.Names[0], x, t.Names[1], y),
			fmt.Sprintf("      <p class=\"check\">Probe: %d + %d = %d Franken.</p>\n", x, y, x+y),
		))
	}
	body.WriteString(sheet.SolutionPage(boxes...))
	d.Body = body.String()

	type spec struct {
		Kind  string    `json:"kind"`
		Names [2]string `json:"names"`
		Total int       `json:"total"`
	}
	specs := map[string]spec{}
	for _, t := range tasks {
		specs[t.Key] = spec{Kind: t.Kind, Names: t.Names, Total: t.Values[0] + t.Values[1]}
	}
	d.Set("PRICES", specs)
	d.AddScript(renderJS)
	return d
}
