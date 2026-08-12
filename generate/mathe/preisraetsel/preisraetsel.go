// Package preisraetsel baut das Arbeitsblatt "Preisrätsel"
// (Gesamtpreis, Preisunterschied, Hälfte, Doppeltes, Dreifaches).
package preisraetsel

import (
	_ "embed"
	"fmt"
	"strings"

	"lernmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

type aufgabe struct {
	Key   string
	Typ   string // Motiv fuer render.js
	Titel string
	Text  string
	Frage string
	Namen [2]string
	Werte [2]int
}

const css = `
  .spalten { display: flex; gap: 8mm; flex: 1 1 auto; min-height: 0; }
  .spalte { display: flex; flex-direction: column; flex: 1 1 50%; min-width: 0; }
  .spalte + .spalte { padding-left: 8mm; border-left: 2px dashed #d8dfe8; }
  h2 { margin: 0 0 3mm; color: #e8548c; font: 700 24pt 'Caveat', cursive; }
  h3 { margin: 0 0 2mm; color: #e8548c; font: 700 20pt 'Caveat', cursive; }
  .story { margin: 0 0 3mm; }
  .bild { flex: 0 0 auto; margin: 0 0 3mm; line-height: 0; }
  .bild svg { display: block; width: 100%; height: auto; }
  .frage { margin: 0 0 3mm; font-weight: 700; }
  .antworten { margin: 0 0 6mm; font-size: 15pt; }
  .antwort { display: block; margin-bottom: 5mm; white-space: nowrap; }
  .linie { min-width: 34mm; border-bottom-color: #aab5c4; }
  .rechnung { flex: 1 1 auto; min-height: 40mm; }
  .begruendung { flex: 1 1 auto; min-height: 55mm; }
  .schreiblinien { margin-top: 1mm; }
  .lbox p { font-size: 12.5pt; }
  .probe { color: #65778a; font-size: 11.5pt !important; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Fach:  "mathe",
		Name:  "preisraetsel",
		Titel: "Preisrätsel",
		Meta:  "Mathe, Deutsch · 8 Aufgabenblätter + Lösungen · Summe, Differenz, Vielfache",
		Build: build,
	})
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Titel: "Mathe-Arbeitsblatt: Preisrätsel",
		CSS:   css,
		Rough: true,
	}

	var body strings.Builder
	for nr, a := range aufgaben {
		nr++
		inhalt := fmt.Sprintf(`  <div class="spalten">
    <div class="spalte">
      <h2>Aufgabe %d &nbsp; <small>%s</small></h2>
      <p class="story">%s</p>
      <div class="bild" data-preis="%s"></div>
      <p class="frage">%s</p>
      <div class="kasten rechnung">Rechnung:</div>
    </div>
    <div class="spalte">
      <h3>Meine Antwort</h3>
      <div class="antworten">
        <span class="antwort">%s: <span class="linie"></span> Fr.</span>
        <span class="antwort">%s: <span class="linie"></span> Fr.</span>
      </div>
      <h3>Warum meine Antwort stimmt</h3>
      <div class="kasten begruendung">
        %s
      </div>
    </div>
  </div>
`, nr, a.Titel, a.Text, a.Key, a.Frage, a.Namen[0], a.Namen[1], sheet.Lines(12))
		body.WriteString(sheet.Page(fmt.Sprintf("Preisrätsel &ndash; Blatt %d", nr), inhalt))
	}

	var boxen []string
	for nr, a := range aufgaben {
		nr++
		x, y := a.Werte[0], a.Werte[1]
		boxen = append(boxen, sheet.SolutionBox(
			fmt.Sprintf("Aufgabe %d &ndash; %s", nr, a.Titel),
			fmt.Sprintf("      <p>%s: <b>%d Franken</b> &nbsp; · &nbsp; %s: <b>%d Franken</b></p>\n",
				a.Namen[0], x, a.Namen[1], y),
			fmt.Sprintf("      <p class=\"probe\">Probe: %d + %d = %d Franken.</p>\n", x, y, x+y),
		))
	}
	body.WriteString(sheet.SolutionPage(boxen...))
	d.Body = body.String()

	type spec struct {
		Typ   string    `json:"typ"`
		Namen [2]string `json:"namen"`
		Summe int       `json:"summe"`
	}
	specs := map[string]spec{}
	for _, a := range aufgaben {
		specs[a.Key] = spec{Typ: a.Typ, Namen: a.Namen, Summe: a.Werte[0] + a.Werte[1]}
	}
	d.Set("PREISE", specs)
	d.AddScript(renderJS)
	return d
}
