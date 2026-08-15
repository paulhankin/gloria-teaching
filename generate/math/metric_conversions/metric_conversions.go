// Package metricconversions builds the metric length conversion worksheet.
package metricconversions

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
  h2 { margin: 0 0 1.5mm; color: #e8548c; font: 700 21pt 'Caveat', cursive; }
  h3 { margin: 0 0 1mm; color: #e8548c; font: 700 17pt 'Caveat', cursive; }
  .intro { margin: 0 0 2mm; }
  .figure { position: relative; flex: 0 0 51mm; min-height: 0; margin: 0 0 3mm; }
  .figure svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .exercise-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 2.3mm 10mm; }
  .exercise { padding: 1.5mm 2.5mm; border-bottom: 1.5px dotted #ccd5e0; white-space: nowrap; }
  .exercise-number { display: inline-block; width: 7mm; color: #7a869a; }
  .answer-blank { display: inline-block; min-width: 17mm; height: 5.5mm;
                  border-bottom: 2px dotted #aeb9c8; vertical-align: baseline; }
  .answer-blank.short { min-width: 10mm; }
  .answer-blank.wide { min-width: 28mm; }
  .comparison-layout { display: grid; grid-template-columns: 1.2fr .8fr; gap: 8mm; flex: 1 1 auto; min-height: 0; }
  .compare-list { display: grid; grid-template-columns: 1fr 1fr; gap: 2.5mm 7mm; }
  .compare-row { display: grid; grid-template-columns: 1fr 13mm 1fr; align-items: end;
                 padding: 2mm; border-bottom: 1.5px dotted #ccd5e0; text-align: center; }
  .sign-box { height: 8mm; border: 2px dashed #e0a01e; border-radius: 5px; }
  .ordering { margin: 0 0 3mm; padding: 2.5mm 3mm; border: 2px dashed #d8dfe8; border-radius: 8px; }
  .ordering p { margin: 0 0 1.5mm; }
  .order-line { height: 8mm; border-bottom: 1.5px dotted #aeb9c8; }
  .side-figure { position: relative; flex: 1 1 auto; min-height: 35mm; }
  .side-figure svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .story-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 5mm 7mm; flex: 1 1 auto; min-height: 0; }
  .story-card { display: flex; flex-direction: column; min-height: 0; padding: 3mm;
                border: 2px dashed #d8dfe8; border-radius: 10px; }
  .story-card p { margin: 0 0 1.5mm; }
  .story-question { font-weight: 700; }
  .story-lines { flex: 1 1 auto; min-height: 0; }
  .story-lines div { height: 7.5mm; border-bottom: 1.5px dotted #ccd5e0; }
  .solution-box .reason { color: #65778a; font-size: 10.5pt; }
  .solution-box .compact { line-height: 1.25; }
  .solution-box .calculation { color: #65778a; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "metric_conversions",
		Title:   "Längen umwandeln",
		Date:    "15 Aug 2026",
		Meta:    "4. Klasse · mm, cm, dm, m und km · 4 Aufgabenblätter + Lösungen",
		Build:   build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Arbeitsblatt: Längen umwandeln",
		CSS:   css,
		Rough: true,
	}

	d.Body = conversionPage() + composedPage() + comparisonPage() + wordProblemPage()
	d.Solutions = conversionSolutions() + comparisonSolutions() + wordProblemSolutions()
	d.AddScript(renderJS)
	return d
}

func conversionPage() string {
	var exercises strings.Builder
	for i, task := range directConversions {
		fmt.Fprintf(&exercises, "    <div class=\"exercise\"><span class=\"exercise-number\">%d.</span>%s</div>\n", i+1, task.Prompt)
	}
	content := fmt.Sprintf(`  <p class="intro">Nutze die Längentreppe. Zwischen m, dm, cm und mm bedeutet ein Schritt nach rechts <b>· 10</b>, ein Schritt nach links <b>: 10</b>. Zwischen km und m rechnest du mit <b>1000</b>.</p>
  <div class="figure" data-figure="ladder"></div>
  <div class="exercise-grid">
%s  </div>
`, exercises.String())
	return sheet.Page("Längen umwandeln &ndash; 1. Die Längentreppe", content)
}

func composedPage() string {
	var exercises strings.Builder
	for i, task := range composedConversions {
		fmt.Fprintf(&exercises, "      <div class=\"exercise\"><span class=\"exercise-number\">%d.</span>%s</div>\n", i+1, task.Prompt)
	}
	content := fmt.Sprintf(`  <p class="intro">Zerlege oder bündle die Längen. Achte besonders auf fehlende Stellen: <b>6 m 4 cm = 6 m 0 dm 4 cm</b>.</p>
  <div class="exercise-grid">
%s  </div>
`, exercises.String())
	return sheet.Page("Längen umwandeln &ndash; 2. Zerlegen und zusammensetzen", content)
}

func comparisonPage() string {
	var rows strings.Builder
	for _, task := range comparisons {
		fmt.Fprintf(&rows, "      <div class=\"compare-row\"><span>%s</span><span class=\"sign-box\"></span><span>%s</span></div>\n", task.Left, task.Right)
	}
	var orders strings.Builder
	for i, task := range orderings {
		fmt.Fprintf(&orders, `      <div class="ordering">
        <h3>%d. Von kurz nach lang</h3>
        <p>%s</p><div class="order-line"></div>
      </div>
`, i+1, task.Values)
	}
	content := fmt.Sprintf(`  <p class="intro">Wandle zuerst in dieselbe Einheit um. Setze dann <b>&lt;</b>, <b>&gt;</b> oder <b>=</b> ein.</p>
  <div class="comparison-layout">
    <div>
      <div class="compare-list">
%s      </div>
    </div>
    <div style="display:flex;flex-direction:column;min-height:0">
%s      <div class="side-figure" data-figure="compare"></div>
    </div>
  </div>
`, rows.String(), orders.String())
	return sheet.Page("Längen umwandeln &ndash; 3. Vergleichen und ordnen", content)
}

func wordProblemPage() string {
	var cards strings.Builder
	for i, task := range wordProblems {
		fmt.Fprintf(&cards, `    <div class="story-card">
      <h2>%d. %s</h2>
      <p>%s</p>
      <p class="story-question">%s</p>
      <div class="story-lines"><div></div><div></div><div></div></div>
      <p><b>Antwort:</b> <span class="blank" style="min-width:55mm"></span></p>
    </div>
`, i+1, task.Title, task.Story, task.Question)
	}
	content := fmt.Sprintf(`  <p class="intro">Wandle die Angaben zuerst in eine passende gemeinsame Einheit um. Schreibe Rechnung und Antwort auf.</p>
  <div class="story-grid">
%s  </div>
`, cards.String())
	return sheet.Page("Längen umwandeln &ndash; 4. Längen im Alltag", content)
}

func conversionSolutions() string {
	var direct, composed strings.Builder
	for i, task := range directConversions {
		fmt.Fprintf(&direct, "      <p class=\"compact\">%d) %s</p>\n", i+1, task.Answer)
	}
	for i, task := range composedConversions {
		fmt.Fprintf(&composed, "      <p class=\"compact\">%d) %s</p>\n", i+1, task.Answer)
	}
	return sheet.SolutionPage(
		sheet.SolutionBox("1. Die Längentreppe", direct.String()),
		sheet.SolutionBox("2. Zerlegen und zusammensetzen", composed.String()),
	)
}

func comparisonSolutions() string {
	var signs, orders strings.Builder
	for i, task := range comparisons {
		fmt.Fprintf(&signs, "      <p class=\"compact\">%d) %s <b>%s</b> %s <span class=\"reason\">(%s)</span></p>\n", i+1, task.Left, task.Sign, task.Right, task.Reason)
	}
	for i, task := range orderings {
		fmt.Fprintf(&orders, "      <p class=\"compact\">%d) %s</p>\n", i+1, task.Answer)
	}
	return sheet.SolutionPage(
		sheet.SolutionBox("3a. Vergleichen", signs.String()),
		sheet.SolutionBox("3b. Von kurz nach lang", orders.String()),
	)
}

func wordProblemSolutions() string {
	var boxes []string
	for i, task := range wordProblems {
		boxes = append(boxes, sheet.SolutionBox(
			fmt.Sprintf("4.%d %s", i+1, task.Title),
			fmt.Sprintf("      <p class=\"calculation\">%s</p>\n", task.Calculation),
			fmt.Sprintf("      <p>%s</p>\n", task.Answer),
		))
	}
	return sheet.SolutionPage(boxes...)
}
