package site

import (
	"html/template"
	"strings"
)

// PublicData is the published worksheet page for one user.
type PublicData struct {
	OwnerUsername  string
	ViewerUsername string
	Worksheets     []Worksheet
}

// PublicIndex renders one user's public worksheets.
func PublicIndex(d PublicData) string {
	var b strings.Builder
	if err := publicIndexTmpl.Execute(&b, d); err != nil {
		panic(err)
	}
	return b.String()
}

var publicIndexTmpl = template.Must(template.New("public-index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.OwnerUsername}}'s worksheets</title>
<style>
  * { box-sizing:border-box; }
  body { margin:0; padding:36px 20px 64px; color:#172033; font:15px/1.45 system-ui,sans-serif; }
  .wrap { max-width:900px; margin:0 auto; }
  header { display:flex; justify-content:space-between; align-items:baseline; gap:20px; padding-bottom:18px; border-bottom:1px solid #d8dee8; }
  h1 { margin:0; font-size:26px; line-height:1.2; }
  header p, .empty, .meta, .date { color:#667085; }
  header p { margin:4px 0 0; }
  nav { display:flex; gap:12px; align-items:center; white-space:nowrap; }
  a { color:#175cd3; }
  table { width:100%; margin-top:28px; border-collapse:collapse; border-top:1px solid #172033; }
  th { padding:9px 10px; border-bottom:1px solid #d8dee8; color:#667085; font-size:12px; text-align:left; text-transform:uppercase; letter-spacing:.04em; }
  td { padding:14px 10px; border-bottom:1px solid #d8dee8; vertical-align:top; }
  .title { font-weight:650; }
  .date { white-space:nowrap; font-size:13px; }
  .actions { text-align:right; white-space:nowrap; }
  .actions a { display:block; }
  .actions a + a { margin-top:4px; }
  .empty { margin-top:28px; }
  @media (max-width:650px) {
    body { padding:24px 12px 48px; }
    header { display:block; }
    nav { margin-top:10px; }
    table, tbody, tr, td { display:block; }
    thead { display:none; }
    tr { padding:10px 0; border-bottom:1px solid #d8dee8; }
    td { padding:3px 4px; border:0; }
    .actions { margin-top:6px; text-align:left; }
  }
</style>
</head>
<body><div class="wrap">
<header>
  <div><h1>{{.OwnerUsername}}'s worksheets</h1><p>Published worksheets available to signed-in users</p></div>
  <nav><span>{{.ViewerUsername}}</span><a href="/">My worksheets</a></nav>
</header>
{{if .Worksheets}}
<table>
<thead><tr><th>Worksheet</th><th>Updated</th><th>Details</th><th></th></tr></thead>
<tbody>{{range .Worksheets}}
<tr>
  <td><a class="title" href="/worksheets/{{$.OwnerUsername}}/sheet/{{.Name}}">{{.Title}}</a></td>
  <td class="date">{{if .Date}}{{.Date}}{{else}}—{{end}}</td>
  <td class="meta">{{.Meta}}</td>
  <td class="actions"><a href="/{{.OutputPath}}/index.pdf{{if .Version}}?v={{.Version}}{{end}}">Worksheet PDF</a><a href="/{{.OutputPath}}/solutions.pdf{{if .Version}}?v={{.Version}}{{end}}">Solutions PDF</a></td>
</tr>
{{end}}</tbody>
</table>
{{else}}<p class="empty">This user has no published worksheets.</p>{{end}}
</div></body></html>`))
