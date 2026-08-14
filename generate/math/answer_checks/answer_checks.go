// Package answerchecks builds the "Kann das stimmen?" checking dossier.
package answerchecks

import (
	_ "embed"
	"fmt"
	"html"
	"strings"

	"learningmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

const css = `
  body { font-size: 14pt; }
  h2 { margin: 0 0 2mm; color: #e8548c; font: 700 22pt 'Caveat', cursive; }
  h3 { margin: 0 0 1.5mm; color: #e8548c; font: 700 19pt 'Caveat', cursive; }
  .story { margin: 0 0 2mm; }
  .question-area { flex: 0 0 76mm; min-height: 0; padding: 3mm 4mm;
                   border: 2px dashed #d8dfe8; border-radius: 10px; }
  .response-area { flex: 1 1 auto; min-height: 0; margin-top: 5mm; display: flex; gap: 7mm; }
  .response-box { flex: 1 1 50%; min-width: 0; padding: 3mm 4mm;
                  border: 2px dashed #d8dfe8; border-radius: 10px; }
  .response-box h3 { margin-bottom: 2mm; }
  .written-wrap { height: 100%; display: flex; align-items: center; justify-content: center; gap: 18mm; }
  .written { width: 42mm; font: 700 28pt/1.12 'Courier New', monospace;
             text-align: right; color: #1f3550; }
  .written .line { border-bottom: 2.5px solid #1f3550; padding-bottom: 1mm; }
  .written .answer-blank { min-height: 12mm; }
  .answer-row { margin: 3mm 0; font-size: 16pt; }
  .answer-row .blank { min-width: 42mm; }
  .figure { position: relative; height: 43mm; margin: 1mm 0 2mm; }
  .figure svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .price-question { display: grid; grid-template-columns: 3fr 2fr; gap: 8mm; height: 100%; }
  .price-text { min-width: 0; }
  .logic-question { display: grid; grid-template-columns: 3fr 2fr; gap: 8mm; height: 100%; }
  .logic-question .figure { height: 47mm; }
  .question { margin: 2mm 0; font-weight: 700; font-size: 17pt; }
  .writing-lines { margin-top: 1mm; }
  .check-list { margin: 1mm 0 0; padding-left: 6mm; }
  .check-list li { margin-bottom: 1mm; }
  .check-list li::marker { content: "✓  "; color: #2e8b57; }
  .solution-box .check-list { font-size: 11.5pt; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "answer_checks",
		Title:   "Kann das stimmen?",
		Date:    "14 Aug 2026",
		Meta:    "3./4. Klasse · 9 Seiten + Lösungen · Aufgaben lösen und selbst prüfen",
		Build:   build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Dossier: Kann das stimmen?",
		CSS:   css,
		Rough: true,
	}

	var body strings.Builder
	for i, task := range arithmeticTasks {
		body.WriteString(arithmeticPage(i+1, task))
	}
	for i, task := range priceTasks {
		body.WriteString(pricePage(i+1, task))
	}
	for i, task := range logicTasks {
		body.WriteString(logicPage(i+1, task))
	}
	d.Body = body.String()
	d.Solutions = firstSolutionPage() + logicSolutionPage()

	type pictureSpec struct {
		Kind   string   `json:"kind"`
		Labels []string `json:"labels"`
	}
	pictures := map[string]pictureSpec{}
	for _, task := range priceTasks {
		pictures[task.Key] = pictureSpec{Kind: task.Kind, Labels: task.Names[:]}
	}
	for _, task := range logicTasks {
		pictures[task.Key] = pictureSpec{Kind: task.Kind, Labels: task.Labels}
	}
	d.Set("CHECK_PICTURES", pictures)
	d.AddScript(renderJS)
	return d
}

func arithmeticPage(number int, task arithmeticTask) string {
	content := fmt.Sprintf(`  <div class="question-area">
    <div class="written-wrap">
      <div class="written">
        <div>%d</div><div class="line">%s %d</div><div class="answer-blank">&nbsp;</div>
      </div>
      <div>
        <h2>1. Löse die Aufgabe.</h2>
        <p>Schreibe dein Ergebnis unter den Strich.</p>
      </div>
    </div>
  </div>
  <div class="response-area">
    <div class="response-box">
      <h3>1. Wie hast du gerechnet?</h3>
      %s
    </div>
    <div class="response-box">
      <h3>2. Wie hast du deine Lösung überprüft?</h3>
      %s
    </div>
  </div>
`, task.Top, task.Operation, task.Bottom, sheet.Lines(8), sheet.Lines(8))
	return sheet.Page(fmt.Sprintf("Kann das stimmen? &ndash; Lösen und prüfen, Aufgabe %d/4", number), content)
}

func pricePage(number int, task priceTask) string {
	content := fmt.Sprintf(`  <div class="question-area">
    <div class="price-question">
      <div class="price-text">
        <h2>Aufgabe %d: %s</h2>
        <p class="story">%s</p>
        <p class="question">%s</p>
        <div class="answer-row">%s: <span class="blank"></span> Fr.</div>
        <div class="answer-row">%s: <span class="blank"></span> Fr.</div>
      </div>
      <div class="figure" data-picture="%s"></div>
    </div>
  </div>
  <div class="response-area">
    <div class="response-box">
      <h3>1. Wie hast du die Antwort gefunden?</h3>
      %s
    </div>
    <div class="response-box">
      <h3>2. Wie hast du deine Lösung überprüft?</h3>
      %s
    </div>
  </div>
`, number, task.Title, task.Story, task.Question, task.Names[0], task.Names[1], task.Key, sheet.Lines(8), sheet.Lines(8))
	return sheet.Page(fmt.Sprintf("Kann das stimmen? &ndash; Warm-up II, Aufgabe %d/2", number), content)
}

func logicPage(number int, task logicTask) string {
	content := fmt.Sprintf(`  <div class="question-area">
    <div class="logic-question">
      <div>
        <h2>Knobelaufgabe %d: %s</h2>
        <p class="story">%s</p>
        <p class="question">%s</p>
        <p><b>Meine Antwort:</b> <span class="blank" style="min-width:60mm"></span></p>
      </div>
      <div class="figure" data-picture="%s"></div>
    </div>
  </div>
  <div class="response-area">
    <div class="response-box">
      <h3>1. Wie hast du die Antwort gefunden?</h3>
      %s
    </div>
    <div class="response-box">
      <h3>2. Wie hast du deine Lösung überprüft?</h3>
      %s
    </div>
  </div>
`, number, task.Title, task.Story, task.Question, task.Key, sheet.Lines(8), sheet.Lines(8))
	return sheet.Page(fmt.Sprintf("Kann das stimmen? &ndash; Knobeln und prüfen %d/3", number), content)
}

func firstSolutionPage() string {
	var arithmetic strings.Builder
	for i, task := range arithmeticTasks {
		if task.Operation == "−" {
			fmt.Fprintf(&arithmetic,
				"      <p>%d) %d &minus; %d = <b>%d</b>. Probe: %d + %d = %d &#10004;</p>\n",
				i+1, task.Top, task.Bottom, task.Correct, task.Correct, task.Bottom, task.Top)
		} else {
			fmt.Fprintf(&arithmetic,
				"      <p>%d) %d + %d = <b>%d</b>. Probe: %d &minus; %d = %d &#10004;</p>\n",
				i+1, task.Top, task.Bottom, task.Correct, task.Correct, task.Bottom, task.Top)
		}
	}

	boxes := []string{sheet.SolutionBox("Lösen und prüfen &ndash; Gegenrechnungen", arithmetic.String())}
	for i, task := range priceTasks {
		boxes = append(boxes, sheet.SolutionBox(
			fmt.Sprintf("Warm-up II, Aufgabe %d &ndash; %s", i+1, task.Title),
			fmt.Sprintf("      <p>%s: <b>%d Franken</b>; %s: <b>%d Franken</b>.</p>\n", task.Names[0], task.Values[0], task.Names[1], task.Values[1]),
			fmt.Sprintf("      <p><b>Probe:</b> %s &#10004;</p>\n", task.CheckText),
		))
	}
	return sheet.SolutionPage(boxes...)
}

func logicSolutionPage() string {
	var boxes []string
	for i, task := range logicTasks {
		var checks strings.Builder
		checks.WriteString("      <ul class=\"check-list\">\n")
		for _, item := range task.CheckItems {
			fmt.Fprintf(&checks, "        <li>%s</li>\n", item)
		}
		checks.WriteString("      </ul>\n")
		boxes = append(boxes, sheet.SolutionBox(
			fmt.Sprintf("Knobelaufgabe %d &ndash; %s", i+1, html.EscapeString(task.Title)),
			fmt.Sprintf("      <p>%s</p>\n", task.Answer),
			fmt.Sprintf("      <p><b>Lösungsweg:</b> %s</p>\n", task.Reasoning),
			"      <p><b>Probe aller Aussagen:</b></p>\n",
			checks.String(),
		))
	}
	return sheet.SolutionPage(boxes...)
}
