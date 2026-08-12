// Package zahlenfolgen baut das Arbeitsblatt "Zahlenfolgen" (Ordnungszahlen).
package zahlenfolgen

import (
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"lernmaterial/internal/sheet"
)

//go:embed render.js
var renderJS string

const css = `
  .haelften { display: flex; flex-direction: column; gap: 5mm; flex: 1 1 auto; min-height: 0; }
  .haelfte { flex: 1 1 50%; min-height: 0; display: flex; }
  .haelfte + .haelfte { border-top: 2px dashed #d8dfe8; padding-top: 5mm; }
  .aufgabe { width: 100%; display: flex; flex-direction: column; }
  .aufgabe h2 { margin: 0 0 1mm; color: #e8548c; font: 700 21pt 'Caveat', cursive; }
  .aufgabe h2 small { font-size: 15pt; color: #7a869a; font-weight: 400; }
  .story { margin: 0 0 1mm; }
  .unten { display: flex; flex-direction: column; flex: 1 1 auto; min-height: 0; }
  .bild { flex: 1 1 auto; min-height: 0; position: relative; margin-bottom: 1mm; }
  .bild svg { position: absolute; top: 0; left: 0; width: 100%; height: 100%; }
  .text { flex: 0 0 auto; }
  ol.fragen { margin: 0; padding-left: 6mm; column-count: 2; column-gap: 8mm; }
  ol.fragen li { margin-bottom: 2mm; break-inside: avoid; }
  .linie { min-width: 22mm; }
  .lbox .folge { margin-bottom: 2mm; color: #5a6b7a; font-size: 11pt; }
`

func init() {
	sheet.Register(sheet.Worksheet{
		Fach:  "mathe",
		Name:  "zahlenfolgen",
		Titel: "Zahlenfolgen",
		Meta:  "Mathe, jünger · 3 Aufgabenblätter + Lösungen · Ordnungszahlen",
		Build: build,
	})
}

// frageText liefert Fragestellung und Loesungstext.
func (a aufgabe) frageText(f frage) (q, ans string) {
	z := a.Zahlen
	switch f.Art {
	case stelle:
		return fmt.Sprintf("Welche Zahl steht an der <b>%d. Stelle</b>?", f.A),
			fmt.Sprintf("Die %d. Zahl ist <b>%d</b>.", f.A, z[f.A-1])
	case welcheStelle:
		i := slices.Index(z, f.A) + 1
		return fmt.Sprintf("An welcher <b>Stelle</b> steht die Zahl <b>%d</b>?", f.A),
			fmt.Sprintf("Die %d steht an der <b>%d. Stelle</b>.", f.A, i)
	case plus:
		x, y := z[f.A-1], z[f.B-1]
		return fmt.Sprintf("Rechne: <b>%d. Zahl + %d. Zahl</b> = ?", f.A, f.B),
			fmt.Sprintf("%d. Zahl = %d, %d. Zahl = %d &rarr; %d + %d = <b>%d</b>",
				f.A, x, f.B, y, x, y, x+y)
	default:
		x, y := z[f.A-1], z[f.B-1]
		return fmt.Sprintf("Rechne: <b>%d. Zahl &minus; %d. Zahl</b> = ?", f.A, f.B),
			fmt.Sprintf("%d. Zahl = %d, %d. Zahl = %d &rarr; %d &minus; %d = <b>%d</b>",
				f.A, x, f.B, y, x, y, x-y)
	}
}

func build() *sheet.Doc {
	d := &sheet.Doc{
		Titel: "Mathe-Arbeitsblatt: Zahlenfolgen",
		CSS:   css,
		Rough: true,
	}

	var body strings.Builder
	for i := 0; i < len(aufgaben); i += 2 {
		var halves strings.Builder
		for _, a := range aufgaben[i:min(i+2, len(aufgaben))] {
			var lis strings.Builder
			for _, f := range a.Fragen {
				q, _ := a.frageText(f)
				fmt.Fprintf(&lis, "            <li>%s<span class=\"linie\"></span></li>\n", q)
			}
			fmt.Fprintf(&halves, `    <div class="haelfte">
      <div class="aufgabe">
        <h2>%s &nbsp;<small>%s</small></h2>
        <p class="story">%s</p>
        <div class="unten">
          <div class="bild" data-folge="%s"></div>
          <div class="text">
            <ol class="fragen">
%s            </ol>
          </div>
        </div>
      </div>
    </div>
`, a.Titel, a.Name, a.Story, a.Key, lis.String())
		}
		body.WriteString(sheet.Page(
			fmt.Sprintf("Zahlenfolgen &ndash; Blatt %d", i/2+1),
			fmt.Sprintf("  <div class=\"haelften\">\n%s  </div>\n", halves.String())))
	}

	var boxen []string
	for _, a := range aufgaben {
		zeilen := []string{fmt.Sprintf("      <p class=\"folge\">Folge: %s</p>\n", joinInts(a.Zahlen))}
		for k, f := range a.Fragen {
			_, ans := a.frageText(f)
			zeilen = append(zeilen, fmt.Sprintf("      <p>%c) %s</p>\n", "abcd"[k], ans))
		}
		boxen = append(boxen, sheet.SolutionBox(a.Titel+" &ndash; "+a.Name, zeilen...))
	}
	body.WriteString(sheet.SolutionPage(boxen...))
	d.Body = body.String()

	// render.js erwartet Strings als Zahlen (wegen Textbreite).
	type spec struct {
		Typ       string   `json:"typ"`
		Zahlen    []string `json:"zahlen"`
		Kopffarbe string   `json:"kopffarbe,omitempty"`
	}
	folgen := map[string]spec{}
	for _, a := range aufgaben {
		s := spec{Typ: a.Typ, Kopffarbe: a.Kopffarbe}
		for _, z := range a.Zahlen {
			s.Zahlen = append(s.Zahlen, strconv.Itoa(z))
		}
		folgen[a.Key] = s
	}
	d.Set("FOLGEN", folgen)
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
