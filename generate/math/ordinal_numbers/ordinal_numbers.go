// Package ordinalnumbers builds the "Ordinalzahlen" worksheet.
package ordinalnumbers

import (
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"learningmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

const css = `
  .halves { display: flex; flex-direction: column; gap: 5mm; flex: 1 1 auto; min-height: 0; }
  .half { flex: 1 1 50%; min-height: 0; display: flex; }
  .half + .half { border-top: 2px dashed #d8dfe8; padding-top: 5mm; }
  .task { width: 100%; display: flex; flex-direction: column; }
  .task h2 { margin: 0 0 1mm; color: #e8548c; font: 700 21pt 'Caveat', cursive; }
  .task h2 small { font-size: 15pt; color: #7a869a; font-weight: 400; }
  .story { margin: 0 0 1mm; }
  .lower { display: flex; flex-direction: column; flex: 1 1 auto; min-height: 0; }
  .figure { flex: 1 1 auto; min-height: 0; position: relative; margin-bottom: 1mm; }
  .figure svg { position: absolute; top: 0; left: 0; width: 100%; height: 100%; }
  .text { flex: 0 0 auto; }
  ol.questions { margin: 0; padding-left: 6mm; column-count: 2; column-gap: 8mm; }
  ol.questions li { margin-bottom: 2mm; break-inside: avoid; }
  .blank { min-width: 22mm; }
  .solution-box .sequence { margin-bottom: 2mm; color: #5a6b7a; font-size: 11pt; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "ordinal_numbers",
		Title:   "Ordinalzahlen",
		Date:    "12 Aug 2026",
		Meta:    "Jüngere Primarstufe · 3 Aufgabenblätter + Lösungen",
		Build:   build,
	})
}

// questionText returns the question and the solution text (both in German).
func (t task) questionText(q question) (text, answer string) {
	n := t.Numbers
	switch q.Kind {
	case atPosition:
		return fmt.Sprintf("Welche Zahl steht an der <b>%d. Stelle</b>?", q.A),
			fmt.Sprintf("Die %d. Zahl ist <b>%d</b>.", q.A, n[q.A-1])
	case whichPosition:
		i := slices.Index(n, q.A) + 1
		return fmt.Sprintf("An welcher <b>Stelle</b> steht die Zahl <b>%d</b>?", q.A),
			fmt.Sprintf("Die %d steht an der <b>%d. Stelle</b>.", q.A, i)
	case plus:
		x, y := n[q.A-1], n[q.B-1]
		return fmt.Sprintf("Rechne: <b>%d. Zahl + %d. Zahl</b> = ?", q.A, q.B),
			fmt.Sprintf("%d. Zahl = %d, %d. Zahl = %d &rarr; %d + %d = <b>%d</b>",
				q.A, x, q.B, y, x, y, x+y)
	default:
		x, y := n[q.A-1], n[q.B-1]
		return fmt.Sprintf("Rechne: <b>%d. Zahl &minus; %d. Zahl</b> = ?", q.A, q.B),
			fmt.Sprintf("%d. Zahl = %d, %d. Zahl = %d &rarr; %d &minus; %d = <b>%d</b>",
				q.A, x, q.B, y, x, y, x-y)
	}
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Arbeitsblatt: Ordinalzahlen",
		CSS:   css,
		Rough: true,
	}

	var body strings.Builder
	for i := 0; i < len(tasks); i += 2 {
		var halves strings.Builder
		for _, t := range tasks[i:min(i+2, len(tasks))] {
			var lis strings.Builder
			for _, q := range t.Questions {
				text, _ := t.questionText(q)
				fmt.Fprintf(&lis, "            <li>%s<span class=\"blank\"></span></li>\n", text)
			}
			fmt.Fprintf(&halves, `    <div class="half">
      <div class="task">
        <h2>%s &nbsp;<small>%s</small></h2>
        <p class="story">%s</p>
        <div class="lower">
          <div class="figure" data-sequence="%s"></div>
          <div class="text">
            <ol class="questions">
%s            </ol>
          </div>
        </div>
      </div>
    </div>
`, t.Title, t.Name, t.Story, t.Key, lis.String())
		}
		body.WriteString(sheet.Page(
			fmt.Sprintf("Ordinalzahlen &ndash; Blatt %d", i/2+1),
			fmt.Sprintf("  <div class=\"halves\">\n%s  </div>\n", halves.String())))
	}

	var boxes []string
	for _, t := range tasks {
		lines := []string{fmt.Sprintf("      <p class=\"sequence\">Folge: %s</p>\n", joinInts(t.Numbers))}
		for k, q := range t.Questions {
			_, answer := t.questionText(q)
			lines = append(lines, fmt.Sprintf("      <p>%c) %s</p>\n", "abcd"[k], answer))
		}
		boxes = append(boxes, sheet.SolutionBox(t.Title+" &ndash; "+t.Name, lines...))
	}
	d.Body = body.String()
	d.Solutions = sheet.SolutionPage(boxes...)

	// render.js expects the numbers as strings (for text width).
	type spec struct {
		Kind      string   `json:"kind"`
		Numbers   []string `json:"numbers"`
		HeadColor string   `json:"headColor,omitempty"`
	}
	sequences := map[string]spec{}
	for _, t := range tasks {
		s := spec{Kind: t.Kind, HeadColor: t.HeadColor}
		for _, n := range t.Numbers {
			s.Numbers = append(s.Numbers, strconv.Itoa(n))
		}
		sequences[t.Key] = s
	}
	d.Set("SEQUENCES", sequences)
	d.AddScript(renderJS)
	return d
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return strings.Join(parts, ", ")
}
