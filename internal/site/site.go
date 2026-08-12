// Package site renders the front page: the worksheet index plus the UI for
// worksheet requests (new worksheets and change requests).
package site

import (
	"html/template"
	"strings"

	"learningmaterial/internal/sheet"
	"learningmaterial/internal/store"
)

// Data is everything the front page needs.
type Data struct {
	Worksheets []sheet.Worksheet
	Requests   []store.Request // newest first; empty unless Admin
	Admin      bool
	// Static marks the offline copy written by cmd/generate: no forms,
	// no admin controls (there is no server to talk to).
	Static bool
	// Flash is an optional confirmation message shown at the top.
	Flash string
}

// ChangeRequests returns the change requests for one worksheet.
func (d Data) ChangeRequests(w sheet.Worksheet) []store.Request {
	var out []store.Request
	for _, r := range d.Requests {
		if r.Kind == store.KindChange && r.Worksheet == w.Path() {
			out = append(out, r)
		}
	}
	return out
}

// NewRequests returns the requests for new worksheets.
func (d Data) NewRequests() []store.Request {
	var out []store.Request
	for _, r := range d.Requests {
		if r.Kind == store.KindNew {
			out = append(out, r)
		}
	}
	return out
}

// Fonts is the embedded webfont CSS.
func (d Data) Fonts() template.CSS { return template.CSS(sheet.Asset("fonts.css")) }

// Index renders the front page.
func Index(d Data) string {
	var b strings.Builder
	if err := indexTmpl.Execute(&b, d); err != nil {
		panic(err)
	}
	return b.String()
}

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

  /* request UI */
  .requests { margin-top:16px; border-top:2px dashed #dfe5ee; padding-top:12px; }
  .requests h3 { margin:0 0 8px; font-size:14px; text-transform:uppercase;
                 letter-spacing:.06em; color:#6b7a8d; }
  ul.reqlist { list-style:none; margin:0 0 12px; padding:0; }
  ul.reqlist li { border-left:4px solid #2f9fd0; background:#f6fbfe; border-radius:0 8px 8px 0;
                  padding:8px 12px; margin-bottom:8px; display:flex; gap:12px;
                  align-items:flex-start; }
  ul.reqlist .body { white-space:pre-wrap; flex:1; }
  ul.reqlist .who { display:block; color:#6b7a8d; font-size:13px; margin-top:4px; }
  .empty { color:#98a3b3; font-size:14px; margin:0 0 12px; }
  details.ask summary { cursor:pointer; font-weight:600; color:#2f9fd0; font-size:15px; }
  form.ask { margin-top:10px; display:flex; flex-direction:column; gap:8px; }
  form.ask textarea, form.ask input[type=text] {
    font:15px/1.4 system-ui,sans-serif; padding:8px 10px; border:2px solid #cfd8e3;
    border-radius:8px; width:100%; box-sizing:border-box; background:#fff; color:inherit; }
  form.ask textarea { min-height:80px; resize:vertical; }
  form.ask button { align-self:flex-start; padding:8px 18px; border:0; border-radius:999px;
    background:#2f9fd0; color:#fff; font-weight:600; font-size:15px; cursor:pointer; }
  button.del { border:0; background:none; color:#e8548c; cursor:pointer; font-size:13px;
    font-weight:600; padding:0; }
  .newcard { border-style:dashed; }
  .adminbar { display:flex; justify-content:space-between; align-items:center; gap:12px;
    margin:34px 0 0; color:#6b7a8d; font-size:14px; }
  .adminbar form { margin:0; }
  .adminbar button { border:2px solid #6b7a8d; background:none; color:#6b7a8d; cursor:pointer;
    border-radius:999px; padding:5px 14px; font-size:13px; font-weight:600; }
  .flash { background:#eafaf1; border:2px solid #bfe3cd; color:#2e8b57; border-radius:10px;
    padding:10px 14px; margin:0 0 20px; font-size:15px; }
`

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Learning material</title>
<style>
{{.Fonts}}
` + indexCSS + `
</style>
</head>
<body>
<div class="wrap">
  <h1>Learning material</h1>
  <p class="sub">Printable worksheets (A4 landscape).</p>
{{if .Flash}}  <p class="flash">{{.Flash}}</p>{{end}}
{{range .Worksheets}}
  <div class="card">
    <h2>{{.Title}}</h2>
    <p class="meta">{{.Meta}}</p>
    <div class="links">
      <a class="btn pdf" href="{{.Path}}/index.pdf"><span>Download PDF</span></a>
      <a class="btn html" href="{{.Path}}/index.html"><span>View in browser</span></a>
    </div>
{{if not $.Static}}
    <div class="requests">
      {{if $.Admin}}
      <h3>Change requests</h3>
      {{$rs := $.ChangeRequests .}}
      {{if $rs}}
      <ul class="reqlist">
        {{range $rs}}
        <li>
          <div class="body">{{.Body}}<span class="who">{{if .Author}}{{.Author}} &middot; {{end}}{{.CreatedAt.Format "2 Jan 2006, 15:04"}}</span></div>
          <form method="POST" action="/requests/delete"><input type="hidden" name="id" value="{{.ID}}">
            <button class="del" type="submit">Delete</button></form>
        </li>
        {{end}}
      </ul>
      {{else}}<p class="empty">No change requests yet.</p>{{end}}
      {{end}}
      <details class="ask">
        <summary>Request a change to this worksheet</summary>
        <form class="ask" method="POST" action="/requests">
          <input type="hidden" name="kind" value="change">
          <input type="hidden" name="worksheet" value="{{.Path}}">
          <textarea name="body" required placeholder="What should change on this worksheet?"></textarea>
          <input type="text" name="author" placeholder="Your name (optional)">
          <button type="submit">Send request</button>
        </form>
      </details>
    </div>
{{end}}
  </div>
{{end}}

{{if not .Static}}
  <div class="card newcard">
    <h2>New worksheet</h2>
    <p class="meta">Missing a topic? Describe the worksheet you would like.</p>
    {{if .Admin}}
    <div class="requests" style="border-top:0;padding-top:0">
      <h3>Requested worksheets</h3>
      {{$rs := .NewRequests}}
      {{if $rs}}
      <ul class="reqlist">
        {{range $rs}}
        <li>
          <div class="body">{{.Body}}<span class="who">{{if .Author}}{{.Author}} &middot; {{end}}{{.CreatedAt.Format "2 Jan 2006, 15:04"}}</span></div>
          <form method="POST" action="/requests/delete"><input type="hidden" name="id" value="{{.ID}}">
            <button class="del" type="submit">Delete</button></form>
        </li>
        {{end}}
      </ul>
      {{else}}<p class="empty">No worksheet requests yet.</p>{{end}}
    </div>
    {{end}}
    <details class="ask">
      <summary>Request a new worksheet</summary>
      <form class="ask" method="POST" action="/requests">
        <input type="hidden" name="kind" value="new">
        <textarea name="body" required placeholder="Which subject, topic and level? What should the tasks look like?"></textarea>
        <input type="text" name="author" placeholder="Your name (optional)">
        <button type="submit">Send request</button>
      </form>
    </details>
  </div>

  <div class="adminbar">
    <span>{{if .Admin}}Admin mode is on — requests are visible.{{else}}Admin mode is off.{{end}}</span>
    <form method="POST" action="/admin">
      <input type="hidden" name="admin" value="{{if .Admin}}off{{else}}on{{end}}">
      <button type="submit">{{if .Admin}}Disable admin mode{{else}}Enable admin mode{{end}}</button>
    </form>
  </div>
{{end}}
</div>
</body>
</html>
`))
