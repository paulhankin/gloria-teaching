package sheet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// BaseCSS is the shared layout of every worksheet:
// A4 landscape, fixed page height, header, solution page.
const BaseCSS = `
  @page { size: A4 landscape; margin: 10mm 12mm; }
  * { box-sizing: border-box; }
  body { margin: 0; background: #fff; color: #1f3550;
         font: 15pt/1.35 'Patrick Hand', 'Comic Sans MS', cursive; }
  .page { width: 273mm; height: 186mm; max-height: 186mm; overflow: hidden;
          display: flex; flex-direction: column;
          break-after: page; page-break-after: always; }
  .page:last-child { break-after: auto; page-break-after: auto; }
  .header { display: flex; justify-content: space-between; align-items: baseline;
            flex: 0 0 auto; padding-bottom: 1.5mm; margin-bottom: 3mm;
            border-bottom: 2.5px solid #e8548c; }
  h1 { margin: 0; color: #e8548c; font: 700 24pt 'Caveat', cursive; }
  .name-line { color: #7a869a; font-size: 11pt; }
  .blank { display: inline-block; min-width: 24mm; margin-left: 2mm;
           border-bottom: 2px dotted #b6c0cf; }
  .box { padding: 2.5mm 3mm; border: 2px dashed #d8dfe8; border-radius: 8px; }
  .writing-lines div { height: 8.7mm; border-bottom: 1.5px dotted #ccd5e0; }

  .solution { height: auto; max-height: none; overflow: visible; }
  .solution h1 { color: #2e8b57; }
  .solution-cols { columns: 2; column-gap: 9mm; }
  .solution-box { break-inside: avoid; margin-bottom: 3.5mm; padding: 2.5mm 3.5mm;
          border: 2.5px solid #bfe3cd; border-radius: 10px; background: #f5fff9; }
  .solution-box h3 { margin: 0 0 1mm; color: #2e8b57; font: 700 18pt 'Caveat', cursive; }
  .solution-box p { margin: 0 0 1mm; font-size: 12pt; }
`

// NameLine is the standard header line on the right (German, for the pupils).
const NameLine = `<span class="name-line">Name: ____________________ &nbsp; Datum: __________</span>`

// Page builds a worksheet page with a header.
func Page(title, content string) string {
	return page("", title, NameLine, content)
}

// SolutionPage builds the solution page (may grow beyond one page).
// The boxes are laid out in two columns.
func SolutionPage(boxes ...string) string {
	inner := `  <div class="solution-cols">
` + strings.Join(boxes, "") + `  </div>
`
	return page(" solution", "L&ouml;sungen &#10003;",
		`<span class="name-line">f&uuml;r Eltern / Lehrperson</span>`, inner)
}

// SolutionBox is a single box on the solution page.
func SolutionBox(title string, lines ...string) string {
	return fmt.Sprintf("    <div class=\"solution-box\">\n      <h3>%s</h3>\n%s    </div>\n",
		title, strings.Join(lines, ""))
}

func page(cls, title, right, content string) string {
	return fmt.Sprintf("<div class=\"page%s\">\n  <div class=\"header\"><h1>%s</h1>%s</div>\n%s</div>\n",
		cls, title, right, content)
}

// Lines returns n empty writing lines.
func Lines(n int) string {
	return `<div class="writing-lines">` + strings.Repeat("<div></div>", n) + `</div>`
}

// JSON serialises v for embedding in <script> (without HTML escaping,
// but </script> is made harmless).
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
