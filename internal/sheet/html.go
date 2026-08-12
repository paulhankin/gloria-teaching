package sheet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// BaseCSS ist das gemeinsame Layout aller Arbeitsblaetter:
// A4 quer, feste Seitenhoehe, Kopfzeile, Loesungsblatt.
const BaseCSS = `
  @page { size: A4 landscape; margin: 10mm 12mm; }
  * { box-sizing: border-box; }
  body { margin: 0; background: #fff; color: #1f3550;
         font: 15pt/1.35 'Patrick Hand', 'Comic Sans MS', cursive; }
  .page { width: 273mm; height: 186mm; max-height: 186mm; overflow: hidden;
          display: flex; flex-direction: column;
          break-after: page; page-break-after: always; }
  .page:last-child { break-after: auto; page-break-after: auto; }
  .kopf { display: flex; justify-content: space-between; align-items: baseline;
          flex: 0 0 auto; padding-bottom: 1.5mm; margin-bottom: 3mm;
          border-bottom: 2.5px solid #e8548c; }
  h1 { margin: 0; color: #e8548c; font: 700 24pt 'Caveat', cursive; }
  .namenszeile { color: #7a869a; font-size: 11pt; }
  .linie { display: inline-block; min-width: 24mm; margin-left: 2mm;
           border-bottom: 2px dotted #b6c0cf; }
  .kasten { padding: 2.5mm 3mm; border: 2px dashed #d8dfe8; border-radius: 8px; }
  .schreiblinien div { height: 8.7mm; border-bottom: 1.5px dotted #ccd5e0; }

  .loesung { height: auto; max-height: none; overflow: visible; }
  .loesung h1 { color: #2e8b57; }
  .lspalten { columns: 2; column-gap: 9mm; }
  .lbox { break-inside: avoid; margin-bottom: 3.5mm; padding: 2.5mm 3.5mm;
          border: 2.5px solid #bfe3cd; border-radius: 10px; background: #f5fff9; }
  .lbox h3 { margin: 0 0 1mm; color: #2e8b57; font: 700 18pt 'Caveat', cursive; }
  .lbox p { margin: 0 0 1mm; font-size: 12pt; }
`

// NameZeile ist die Standard-Kopfzeile rechts.
const NameZeile = `<span class="namenszeile">Name: ____________________ &nbsp; Datum: __________</span>`

// Page baut eine Arbeitsblattseite mit Kopfzeile.
func Page(titel, inhalt string) string {
	return page("", titel, NameZeile, inhalt)
}

// SolutionPage baut das Loesungsblatt (waechst ueber eine Seite hinaus).
// Die boxen werden zweispaltig gesetzt.
func SolutionPage(boxen ...string) string {
	inner := `  <div class="lspalten">
` + strings.Join(boxen, "") + `  </div>
`
	return page(" loesung", "L&ouml;sungen &#10003;",
		`<span class="namenszeile">f&uuml;r Eltern / Lehrperson</span>`, inner)
}

// SolutionBox ist ein Kasten auf dem Loesungsblatt.
func SolutionBox(titel string, zeilen ...string) string {
	return fmt.Sprintf("    <div class=\"lbox\">\n      <h3>%s</h3>\n%s    </div>\n",
		titel, strings.Join(zeilen, ""))
}

func page(cls, titel, rechts, inhalt string) string {
	return fmt.Sprintf("<div class=\"page%s\">\n  <div class=\"kopf\"><h1>%s</h1>%s</div>\n%s</div>\n",
		cls, titel, rechts, inhalt)
}

// Lines liefert n leere Schreiblinien.
func Lines(n int) string {
	return `<div class="schreiblinien">` + strings.Repeat("<div></div>", n) + `</div>`
}

// JSON serialisiert v fuer die Einbettung in <script> (ohne HTML-Escaping,
// aber </script> wird unschaedlich gemacht).
func JSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	s := strings.TrimRight(buf.String(), "\n")
	return strings.ReplaceAll(s, "</script", `<\/script`)
}
