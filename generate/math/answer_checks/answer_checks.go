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
  h3 { margin: 0 0 1.5mm; color: #e8548c; font: 700 18pt 'Caveat', cursive; }
  .intro { margin: 0 0 3mm; color: #596b7f; }
  .tip { padding: 2mm 3mm; border-left: 4px solid #2f9fd0; background: #f1f9fd;
         border-radius: 0 8px 8px 0; font-size: 12.5pt; }
  .task-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 5mm 7mm; flex: 1; }
  .arithmetic-card { display: grid; grid-template-columns: 34mm 1fr; gap: 4mm;
                     padding: 3mm 4mm; border: 2px dashed #d8dfe8; border-radius: 10px; }
  .written { align-self: center; width: 30mm; font: 700 20pt/1.08 'Courier New', monospace;
             text-align: right; color: #1f3550; }
  .written .line { border-bottom: 2px solid #1f3550; padding-bottom: 1mm; }
  .written .claim { padding-top: 1mm; color: #e8548c; }
  .decision { font-weight: 700; margin-bottom: 2mm; }
  .workline { display: block; margin: 3.5mm 0; border-bottom: 1.5px dotted #b6c0cf; }
  .price-columns { display: flex; gap: 7mm; flex: 1; min-height: 0; }
  .price-card { flex: 1; min-width: 0; display: flex; flex-direction: column;
                padding: 3mm 4mm; border: 2px dashed #d8dfe8; border-radius: 10px; }
  .story { margin: 0 0 2mm; }
  .figure { position: relative; flex: 0 0 42mm; margin: 1mm 0 2mm; }
  .figure svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .answer-row { margin: 1mm 0; }
  .answer-row .blank { min-width: 27mm; }
  .check-box { margin-top: auto; padding: 2.5mm 3mm; border: 2px solid #bfe3cd;
               border-radius: 8px; background: #f5fff9; }
  .logic-layout { display: grid; grid-template-columns: 3fr 2fr; gap: 7mm; flex: 1; min-height: 0; }
  .logic-main { display: flex; flex-direction: column; min-width: 0; }
  .logic-main .figure { flex-basis: 39mm; }
  .logic-question { margin: 1mm 0 2mm; font-weight: 700; font-size: 16pt; }
  .strategy { margin-top: auto; }
  .logic-check { display: flex; flex-direction: column; min-width: 0; }
  .logic-check .box { flex: 1; }
  .check-list { margin: 1mm 0 0; padding-left: 6mm; }
  .check-list li { margin-bottom: 3mm; }
  .check-list li::marker { content: "□  "; color: #2e8b57; }
  .small-note { color: #6a7888; font-size: 11.5pt; }
  .solution-box .check-list { font-size: 11.5pt; }
  .solution-box .check-list li { margin-bottom: 1mm; }
  .solution-box .warning { color: #b05a28; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "answer_checks",
		Title:   "Kann das stimmen?",
		Meta:    "Mathe, 3./4. Klasse · 5-seitiges Dossier + Lösungen · Antworten prüfen",
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
	body.WriteString(arithmeticPage())
	body.WriteString(pricePage())
	for i, task := range logicTasks {
		body.WriteString(logicPage(i+1, task))
	}
	body.WriteString(firstSolutionPage())
	body.WriteString(logicSolutionPage())
	d.Body = body.String()

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

func arithmeticPage() string {
	var cards strings.Builder
	for i, task := range arithmeticTasks {
		checkHint := "Addiere die beiden unteren Zahlen."
		if task.Operation == "+" {
			checkHint = "Subtrahiere von der Summe die zweite Zahl."
		}
		fmt.Fprintf(&cards, `    <div class="arithmetic-card">
      <div class="written">
        <div>%d</div><div class="line">%s %d</div><div class="claim">%d</div>
      </div>
      <div>
        <div class="decision">%d. Kann das stimmen? &nbsp; □ Ja &nbsp; □ Nein</div>
        <div class="small-note">Probe: %s</div>
        <span class="workline"></span><span class="workline"></span>
        <div>Falls nein, richtiges Ergebnis: <span class="blank"></span></div>
      </div>
    </div>
`, task.Top, task.Operation, task.Bottom, task.Claim, i+1, checkHint)
	}
	content := fmt.Sprintf(`  <p class="intro"><b>Mathematikerinnen und Mathematiker rechnen nicht nur:</b> Sie prüfen auch, ob ein Ergebnis wirklich stimmen kann.</p>
  <div class="tip"><b>Gegenrechnung:</b> Bei einer Subtraktion prüfst du mit Addition. Bei einer Addition prüfst du mit Subtraktion.</div>
  <div style="height:4mm"></div>
  <div class="task-grid">
%s  </div>
`, cards.String())
	return sheet.Page("Kann das stimmen? &ndash; Warm-up I: Gegenrechnung", content)
}

func pricePage() string {
	var cards strings.Builder
	for i, task := range priceTasks {
		fmt.Fprintf(&cards, `    <div class="price-card">
      <h2>Aufgabe %d: %s</h2>
      <p class="story">%s</p>
      <div class="figure" data-picture="%s"></div>
      <p><b>%s</b></p>
      <div class="answer-row">%s: <span class="blank"></span> Fr.</div>
      <div class="answer-row">%s: <span class="blank"></span> Fr.</div>
      <div class="check-box"><b>Meine Probe:</b> Addiere beide Preise. Erhältst du wirklich den Gesamtpreis?<br><span class="workline"></span></div>
    </div>
`, i+1, task.Title, task.Story, task.Key, task.Question, task.Names[0], task.Names[1])
	}
	return sheet.Page("Kann das stimmen? &ndash; Warm-up II: Preisrätsel", fmt.Sprintf(`  <p class="intro">Löse jedes Rätsel. <b>Die Probe gehört zur Antwort.</b></p>
  <div class="price-columns">
%s  </div>
`, cards.String()))
}

func logicPage(number int, task logicTask) string {
	content := fmt.Sprintf(`  <div class="logic-layout">
    <div class="logic-main">
      <h2>Knobelaufgabe %d: %s</h2>
      <p class="story">%s</p>
      <div class="figure" data-picture="%s"></div>
      <p class="logic-question">%s</p>
      <div class="box strategy"><b>Mein Lösungsweg:</b>%s</div>
    </div>
    <div class="logic-check">
      <h3>Stimmt meine Antwort?</h3>
      <div class="box">
        <p>Trage deine gefundenen Werte ein. Gehe danach <b>jede Aussage im Text</b> einzeln durch.</p>
        <ul class="check-list">
          <li>Alle Werte sind verschieden.</li>
          <li>Die vorgegebenen Werte kommen genau einmal vor.</li>
          <li>Jeder Satz im Text passt zu meiner Lösung.</li>
          <li>Meine Antwort passt zur Frage.</li>
        </ul>
        <p><b>Meine Antwort:</b></p>
        <span class="workline"></span><span class="workline"></span>
      </div>
    </div>
  </div>
`, number, task.Title, task.Story, task.Key, task.Question, sheet.Lines(5))
	return sheet.Page(fmt.Sprintf("Kann das stimmen? &ndash; Knobeln und prüfen %d/3", number), content)
}

func firstSolutionPage() string {
	var arithmetic strings.Builder
	for i, task := range arithmeticTasks {
		if task.Operation == "−" {
			status := "Ja"
			if task.Claim != task.Correct {
				status = "Nein"
			}
			fmt.Fprintf(&arithmetic, "      <p>%d) <b>%s</b>: %d + %d = %d", i+1, status, task.Bottom, task.Claim, task.Bottom+task.Claim)
			if task.Claim != task.Correct {
				fmt.Fprintf(&arithmetic, ", nicht %d. Richtig: %d &minus; %d = <b>%d</b>.", task.Top, task.Top, task.Bottom, task.Correct)
			} else {
				fmt.Fprintf(&arithmetic, " &#10004;</p>\n")
				continue
			}
			arithmetic.WriteString("</p>\n")
		} else {
			status := "Ja"
			if task.Claim != task.Correct {
				status = "Nein"
			}
			fmt.Fprintf(&arithmetic, "      <p>%d) <b>%s</b>: %d &minus; %d = %d", i+1, status, task.Claim, task.Bottom, task.Claim-task.Bottom)
			if task.Claim != task.Correct {
				fmt.Fprintf(&arithmetic, ", nicht %d. Richtig: %d + %d = <b>%d</b>.</p>\n", task.Top, task.Top, task.Bottom, task.Correct)
			} else {
				arithmetic.WriteString(" &#10004;</p>\n")
			}
		}
	}

	boxes := []string{sheet.SolutionBox("Warm-up I &ndash; Gegenrechnungen", arithmetic.String())}
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
