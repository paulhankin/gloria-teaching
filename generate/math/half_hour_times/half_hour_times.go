// Package halfhourtimes builds a worksheet about reading half-hour times.
package halfhourtimes

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
  .tip { margin: 0 0 3mm; padding: 2mm 3mm; border: 2px dashed #bfe3cd;
         border-radius: 8px; background: #f5fff9; }
  .tip b { color: #2e8b57; }
  .clock-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 4mm 7mm;
                flex: 1 1 auto; min-height: 0; }
  .clock-task { display: flex; flex-direction: column; align-items: center; min-height: 0;
                padding: 1.5mm 2mm; border: 2px dashed #d8dfe8; border-radius: 10px; }
  .task-number { align-self: flex-start; color: #e8548c; font: 700 17pt 'Caveat', cursive; }
  .clock { position: relative; width: 100%; flex: 1 1 auto; min-height: 28mm; }
  .clock svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .answers { width: 100%; text-align: center; line-height: 1.8; white-space: nowrap; }
  .answers .blank { min-width: 21mm; margin-left: 1mm; }
  .drawing-prompt { margin: 0; color: #1f3550; font-weight: 700; font-size: 15pt; }
  .draw-note { margin: 0; color: #7a869a; font-size: 10.5pt; }
  .later-list { display: grid; grid-template-columns: 1fr 1fr; gap: 3.5mm 9mm;
                flex: 1 1 auto; min-height: 0; }
  .later-task { display: grid; grid-template-columns: 8mm 36mm 1fr 39mm; align-items: center;
                min-height: 0; padding: 1mm 2mm; border-bottom: 1.5px dotted #ccd5e0; }
  .later-task .clock { width: 35mm; height: 31mm; min-height: 0; }
  .arrow { text-align: center; color: #e8548c; font: 700 16pt 'Caveat', cursive; line-height: 1.1; }
  .later-answer { text-align: center; white-space: nowrap; }
  .later-answer .blank { min-width: 25mm; }
  .solution-clock { display: inline-block; position: relative; width: 19mm; height: 19mm;
                    vertical-align: middle; margin-right: 2mm; }
  .solution-clock svg { position: absolute; inset: 0; width: 100%; height: 100%; }
  .solution-row { display: inline-flex; align-items: center; width: 49%; margin-bottom: 2mm; }
  .solution-box p { line-height: 1.2; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Subject: "math",
		Name:    "half_hour_times",
		Title:   "Uhrzeiten: halbe Stunden",
		Date:    "15 Aug 2026",
		Meta:    "1.–2. Klasse · halbe Stunden lesen und einzeichnen · 3 Aufgabenblätter + Lösungen",
		Build:   build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Title: "Mathe-Arbeitsblatt: Uhrzeiten – halbe Stunden",
		CSS:   css,
		Rough: true,
	}

	d.Body = readingPage() + drawingPage() + laterPage()
	d.Solutions = solutionPages()

	clocks := map[string]clockTime{}
	for _, task := range readingTasks {
		clocks[task.Key] = task.Time
	}
	for _, task := range drawingTasks {
		clocks[task.Key] = task.Time
	}
	for _, task := range laterTasks {
		clocks[task.Key] = task.Start
		clocks[task.Key+"-answer"] = addHalfHour(task.Start)
	}
	d.Set("CLOCKS", clocks)
	d.AddScript(renderJS)
	return d
}

func readingPage() string {
	var tasks strings.Builder
	for i, task := range readingTasks {
		fmt.Fprintf(&tasks, `    <div class="clock-task">
      <span class="task-number">%d.</span>
      <div class="clock" data-clock="%s"></div>
      <div class="answers"><span class="blank"></span> Uhr</div>
    </div>
`, i+1, task.Key)
	}
	content := fmt.Sprintf(`  <p class="tip"><b>Merke:</b> Bei einer halben Stunde zeigt der lange Minutenzeiger auf die <b>6</b>. Der kurze Stundenzeiger steht genau zwischen zwei Zahlen.</p>
  <div class="clock-grid">
%s  </div>
`, tasks.String())
	return sheet.Page("Uhrzeiten &ndash; 1. Lies die Uhren", content)
}

func drawingPage() string {
	var tasks strings.Builder
	for i, task := range drawingTasks {
		fmt.Fprintf(&tasks, `    <div class="clock-task">
      <span class="task-number">%d.</span>
      <p class="drawing-prompt">%s</p>
      <div class="clock" data-clock="%s" data-blank="true"></div>
      <p class="draw-note">Zeichne beide Zeiger ein.</p>
    </div>
`, i+1, task.Prompt, task.Key)
	}
	content := fmt.Sprintf(`  <p class="tip"><b>Tipp:</b> Zeichne zuerst den langen Minutenzeiger zur <b>6</b>. Setze danach den kurzen Stundenzeiger auf die Hälfte zwischen den beiden passenden Zahlen.</p>
  <div class="clock-grid">
%s  </div>
`, tasks.String())
	return sheet.Page("Uhrzeiten &ndash; 2. Zeichne die Zeiger", content)
}

func laterPage() string {
	var tasks strings.Builder
	for i, task := range laterTasks {
		fmt.Fprintf(&tasks, `    <div class="later-task">
      <span class="task-number">%d.</span>
      <div class="clock" data-clock="%s"></div>
      <div class="arrow">+ eine halbe<br>Stunde &rarr;</div>
      <div class="later-answer"><span class="blank"></span><br>Uhr</div>
    </div>
`, i+1, task.Key)
	}
	content := fmt.Sprintf(`  <p class="tip"><b>Eine halbe Stunde später:</b> Zähle auf der Uhr 30 Minuten weiter. Nach <b>8:30 Uhr</b> kommt zum Beispiel <b>9:00 Uhr</b>.</p>
  <div class="later-list">
%s  </div>
`, tasks.String())
	return sheet.Page("Uhrzeiten &ndash; 3. Eine halbe Stunde später", content)
}

func solutionPages() string {
	var reading, drawing, later strings.Builder
	for i, task := range readingTasks {
		fmt.Fprintf(&reading, "      <p>%d) <b>%s Uhr</b> &nbsp;·&nbsp; halb %s</p>\n",
			i+1, digitalTime(task.Time), germanHour(nextHour(task.Time.Hour)))
	}
	for i, task := range drawingTasks {
		fmt.Fprintf(&drawing, `      <span class="solution-row"><span class="solution-clock" data-clock="%s"></span>%d) <b>%s Uhr</b></span>
`, task.Key, i+1, digitalTime(task.Time))
	}
	for i, task := range laterTasks {
		answer := addHalfHour(task.Start)
		fmt.Fprintf(&later, `      <span class="solution-row"><span class="solution-clock" data-clock="%s-answer"></span>%d) <b>%s Uhr</b></span>
`, task.Key, i+1, digitalTime(answer))
	}
	return sheet.SolutionPage(
		sheet.SolutionBox("1. Lies die Uhren", reading.String()),
		sheet.SolutionBox("2. Zeichne die Zeiger", drawing.String()),
		sheet.SolutionBox("3. Eine halbe Stunde später", later.String()),
	)
}

func addHalfHour(t clockTime) clockTime {
	if t.Minute == 0 {
		return clockTime{Hour: t.Hour, Minute: 30}
	}
	return clockTime{Hour: nextHour(t.Hour), Minute: 0}
}

func nextHour(hour int) int {
	if hour == 12 {
		return 1
	}
	return hour + 1
}

func digitalTime(t clockTime) string {
	return fmt.Sprintf("%d:%02d", t.Hour, t.Minute)
}

func germanHour(hour int) string {
	return []string{"", "eins", "zwei", "drei", "vier", "fünf", "sechs", "sieben", "acht", "neun", "zehn", "elf", "zwölf"}[hour]
}
