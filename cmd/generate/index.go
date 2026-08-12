package main

import (
	"fmt"
	"strings"

	"learningmaterial/internal/sheet"
)

const indexCSS = `
  :root { --ink:#1f3550; }
  body { margin:0; padding:40px 20px 60px; background:#fdfcfa; color:var(--ink);
         font:16px/1.5 system-ui, sans-serif; }
  .wrap { max-width:720px; margin:0 auto; }
  h1 { font:700 56px 'Caveat',cursive; margin:0 0 6px; }
  .sub { color:#6b7a8d; margin:0 0 34px; }
  .card { border:2px solid var(--ink); border-radius:14px; padding:18px 20px;
          margin-bottom:22px; background:#fff; }
  h2 { font:700 34px 'Caveat',cursive; margin:0 0 4px; }
  .meta { color:#6b7a8d; font-size:14px; margin:0 0 14px; }
  .links { display:flex; flex-wrap:wrap; gap:10px; }
  a.btn { display:inline-block; text-decoration:none; padding:8px 16px; border-radius:999px;
          border:2px solid currentColor; font-weight:600; font-size:15px; }
  a.pdf { color:#e8548c; }
  a.html { color:#2f9fd0; }
  a.btn:hover { background:currentColor; }
  a.btn:hover span { color:#fff; }
`

// indexHTML builds the index page linking to every worksheet.
func indexHTML(ws []sheet.Worksheet) string {
	var cards strings.Builder
	for _, w := range ws {
		fmt.Fprintf(&cards, `  <div class="card">
    <h2>%s</h2>
    <p class="meta">%s</p>
    <div class="links">
      <a class="btn pdf" href="%s/index.pdf"><span>Download PDF</span></a>
      <a class="btn html" href="%s/index.html"><span>View in browser</span></a>
    </div>
  </div>
`, w.Title, w.Meta, w.Path(), w.Path())
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Learning material</title>
<style>
%s
%s
</style>
</head>
<body>
<div class="wrap">
  <h1>Learning material</h1>
  <p class="sub">Printable worksheets (A4 landscape).</p>
%s</div>
</body>
</html>
`, sheet.Asset("fonts.css"), indexCSS, cards.String())
}
