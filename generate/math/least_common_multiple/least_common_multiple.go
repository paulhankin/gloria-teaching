// Package leastcommonmultiple builds the least common multiple worksheet.
package leastcommonmultiple

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
  .intro { margin: 0 0 2.5mm; }
  .tip { padding: 2mm 3mm; border-left: 4px solid #e0a01e; background: #fff9e8; border-radius: 5px; }
  .figure { position: relative; min-height: 0; }
  .figure svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .starter-layout { display: flex; flex-direction: column; gap: 4mm; flex: 1; min-height: 0; }
  .sequence-list { display: grid; gap: 3mm; margin-top: 3mm; }
  .sequence { padding: 2.5mm 3mm; border: 2px dashed #d8dfe8; border-radius: 8px; font-size: 16pt; }
  .sequence-number { display: inline-block; width: 8mm; color: #7a869a; }
  .starter-figure { flex: 1; min-height: 36mm; }
  .listing-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4mm 7mm; flex: 1; min-height: 0; }
  .listing-card { padding: 2.5mm 3mm; border: 2px dashed #d8dfe8; border-radius: 9px; }
  .multiple-row { display: grid; grid-template-columns: 18mm 1fr; align-items: end; margin: 2mm 0; }
  .write-line { height: 7mm; border-bottom: 1.5px dotted #aeb9c8; }
  .lcm-row { margin-top: 2.5mm; padding-top: 1.5mm; border-top: 1px solid #e4e9ef; font-weight: 700; }
  .answer-box { display: inline-block; width: 18mm; height: 7mm; border: 2px dashed #e0a01e; border-radius: 5px; vertical-align: middle; }
  .practice-layout { display: flex; flex-direction: column; gap: 4mm; flex: 1; min-height: 0; }
  .practice-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 4mm; align-content: start; }
  .practice-card { padding: 3mm; text-align: center; border: 2px dashed #d8dfe8; border-radius: 9px; font-size: 16pt; }
  .practice-card .answer-box { margin-left: 2mm; }
  .strategy { margin-top: 5mm; padding: 3mm; border: 2px solid #bfe3cd; border-radius: 9px; background: #f5fff9; }
  .practice-figure { flex: 1; min-height: 42mm; }
  .story-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 5mm 7mm; flex: 1; min-height: 0; }
  .story-card { display: flex; flex-direction: column; min-height: 0; padding: 3mm; border: 2px dashed #d8dfe8; border-radius: 10px; }
  .story-card p { margin: 0 0 1.5mm; }
  .story-question { font-weight: 700; }
  .story-lines { flex: 1; min-height: 0; }
  .story-lines div { height: 7mm; border-bottom: 1.5px dotted #ccd5e0; }
  .solution-box .calculation { color: #65778a; }
  .solution-box .compact { line-height: 1.25; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "least_common_multiple",
		Title:   "Das kgV entdecken",
		Date:    "15 Aug 2026",
		Meta:    "4. Klasse · kleinstes gemeinsames Vielfaches · 4 Aufgabenblätter + Lösungen",
		Build:   build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Arbeitsblatt: Das kgV entdecken",
		CSS:   css,
		Rough: true,
	}
	d.Body = sequencePage() + listingPage() + practicePage() + wordProblemPage()
	d.Solutions = skillSolutions() + wordProblemSolutions()
	d.AddScript(renderJS)
	return d
}

func sequencePage() string {
	var rows strings.Builder
	for i, task := range sequences {
		fmt.Fprintf(&rows, `      <div class="sequence"><span class="sequence-number">%d.</span><b>%s</b> %s</div>
`, i+1, task.Start, task.Missing)
	}
	content := fmt.Sprintf(`  <p class="intro"><b>Vielfache</b> entstehen, wenn du eine Zahl mit 1, 2, 3, 4, … multiplizierst. Zum Beispiel: Vielfache von 3 sind 3, 6, 9, 12, …</p>
  <div class="starter-layout">
    <div>
      <div class="tip"><b>Auftrag:</b> Finde den Abstand und setze jede Vielfachreihe fort.</div>
      <div class="sequence-list">
%s      </div>
      <div class="strategy"><b>Merke:</b> Eine Zahl hat unendlich viele Vielfache. Die Reihe hört also nie auf.</div>
    </div>
    <div class="figure starter-figure" data-figure="jumps"></div>
  </div>
`, rows.String())
	return sheet.Page("Das kgV &ndash; 1. Vielfache erkennen", content)
}

func listingPage() string {
	var cards strings.Builder
	for i, task := range listingTasks {
		fmt.Fprintf(&cards, `    <div class="listing-card">
      <h3>%d. Vielfache von %d und %d</h3>
      <div class="multiple-row"><b>V(%d):</b><div class="write-line"></div></div>
      <div class="multiple-row"><b>V(%d):</b><div class="write-line"></div></div>
      <div class="multiple-row"><span></span><div class="write-line"></div></div>
      <div class="lcm-row">kgV(%d, %d) = <span class="answer-box"></span></div>
    </div>
`, i+1, task.A, task.B, task.A, task.B, task.A, task.B)
	}
	content := fmt.Sprintf(`  <p class="intro">Schreibe die Vielfachen beider Zahlen auf. Kreise gemeinsame Vielfache ein. Das <b>kleinste gemeinsame Vielfache</b> ist das <b>kgV</b>.</p>
  <div class="tip" style="margin-bottom:3mm"><b>Beispiel:</b> V(2): 2, 4, <u>6</u>, 8, … &nbsp; V(3): 3, <u>6</u>, 9, … &nbsp; Deshalb: <b>kgV(2, 3) = 6</b></div>
  <div class="listing-grid">
%s  </div>
`, cards.String())
	return sheet.Page("Das kgV &ndash; 2. Gemeinsame Vielfache finden", content)
}

func practicePage() string {
	var cards strings.Builder
	for i, task := range practiceTasks {
		fmt.Fprintf(&cards, `      <div class="practice-card"><span class="sequence-number">%d.</span>kgV(%d, %d) = <span class="answer-box"></span></div>
`, i+1, task.A, task.B)
	}
	content := fmt.Sprintf(`  <p class="intro">Bestimme das kgV. Du darfst die Vielfachreihen klein daneben notieren.</p>
  <div class="practice-layout">
    <div>
      <div class="practice-grid">
%s      </div>
      <div class="strategy"><b>Schlau entdeckt?</b> Ist eine Zahl bereits ein Vielfaches der anderen, ist die grössere Zahl das kgV. Beispiel: <b>kgV(3, 6) = 6</b>.</div>
    </div>
    <div class="figure practice-figure" data-figure="meeting"></div>
  </div>
`, cards.String())
	return sheet.Page("Das kgV &ndash; 3. Jetzt wirst du kgV-Profi", content)
}

func wordProblemPage() string {
	var cards strings.Builder
	for i, task := range wordProblems {
		fmt.Fprintf(&cards, `    <div class="story-card">
      <h2>%d. %s</h2>
      <p>%s</p>
      <p class="story-question">%s</p>
      <div class="story-lines"><div></div><div></div></div>
      <p><b>Antwort:</b> <span class="blank" style="min-width:55mm"></span></p>
    </div>
`, i+1, task.Title, task.Story, task.Question)
	}
	content := fmt.Sprintf(`  <p class="intro">Wenn zwei Dinge regelmässig geschehen, verrät das kgV, wann sie wieder gleichzeitig stattfinden. Schreibe Rechnung und Antwort auf.</p>
  <div class="story-grid">
%s  </div>
`, cards.String())
	return sheet.Page("Das kgV &ndash; 4. Gleichzeitig!", content)
}

func skillSolutions() string {
	var sequenceAnswers, listingAnswers, practiceAnswers strings.Builder
	for i, task := range sequences {
		fmt.Fprintf(&sequenceAnswers, "      <p class=\"compact\">%d) %s</p>\n", i+1, task.Answer)
	}
	for i, task := range listingTasks {
		fmt.Fprintf(&listingAnswers, "      <p class=\"compact\">%d) V(%d) und V(%d): erstes gemeinsames Vielfaches <b>%d</b>; kgV(%d, %d) = <b>%d</b></p>\n", i+1, task.A, task.B, task.Answer, task.A, task.B, task.Answer)
	}
	for i, task := range practiceTasks {
		fmt.Fprintf(&practiceAnswers, "      <p class=\"compact\">%d) kgV(%d, %d) = <b>%d</b></p>\n", i+1, task.A, task.B, task.Answer)
	}
	return sheet.SolutionPage(
		sheet.SolutionBox("1. Vielfachreihen", sequenceAnswers.String()),
		sheet.SolutionBox("2. Gemeinsame Vielfache", listingAnswers.String()),
		sheet.SolutionBox("3. kgV-Profi", practiceAnswers.String()),
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
