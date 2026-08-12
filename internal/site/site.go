// Package site renders the front page: the worksheet index plus the UI for
// worksheet requests (new worksheets and change requests).
package site

import (
	"html/template"
	"strings"

	"learningmaterial/internal/sheet"
	"learningmaterial/internal/store"
)

type Worksheet struct {
	Subject string `json:"subject"`
	Name    string `json:"name"`
	Title   string `json:"title"`
	Meta    string `json:"meta"`
}

func (w Worksheet) Path() string { return w.Subject + "/" + w.Name }

// Data is everything the front page needs.
type Data struct {
	Worksheets []Worksheet
	Requests   []store.Request // newest first; empty unless Admin
	Admin      bool
	// Static marks the offline copy written by cmd/generate: no forms,
	// no admin controls (there is no server to talk to).
	Static bool
	// Flash is an optional confirmation message shown at the top.
	Flash string
	// Log holds recent pipeline events (admin only, newest first).
	Log []string
}

// Open reports whether the request is still in the pipeline.
func (d Data) openRequests(kind store.Kind, worksheet string) []store.Request {
	var out []store.Request
	for _, r := range d.Requests {
		if r.Kind != kind || r.Worksheet != worksheet || !r.Status.Open() {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ChangeRequests returns the open change requests for one worksheet.
func (d Data) ChangeRequests(w Worksheet) []store.Request {
	return d.openRequests(store.KindChange, w.Path())
}

// NewRequests returns the open requests for new worksheets.
func (d Data) NewRequests() []store.Request {
	return d.openRequests(store.KindNew, "")
}

// CompletedRequests returns finished work items, newest first. Keeping these
// visible in admin mode makes an approval auditable after the review card is
// replaced by the published worksheet.
func (d Data) CompletedRequests() []store.Request {
	var out []store.Request
	for _, r := range d.Requests {
		if r.Status == store.StatusDone || r.Status == store.StatusRejected {
			out = append(out, r)
		}
	}
	return out
}

// Busy reports whether anything is queued or being worked on (used to
// auto-refresh the page).
func (d Data) Busy() bool {
	for _, r := range d.Requests {
		if r.Status == store.StatusQueued || r.Status == store.StatusWorking {
			return true
		}
	}
	return false
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

  /* work items */
  ul.reqlist li { display:block; }
  .item { border-left-color:#98a3b3; }
  .item .head { display:flex; gap:10px; align-items:center; flex-wrap:wrap; }
  .tag { font-size:12px; font-weight:700; text-transform:uppercase; letter-spacing:.06em;
    border-radius:999px; padding:3px 10px; color:#fff; background:#98a3b3; }
  .tag.queued  { background:#98a3b3; }
  .tag.working { background:#f0a132; }
  .tag.review  { background:#2f9fd0; }
  .tag.failed  { background:#e8548c; }
  .tag.done    { background:#2e8b57; }
  .tag.rejected { background:#6b7a8d; }
  .item .note { color:#6b7a8d; font-size:13px; margin:6px 0 0; white-space:pre-wrap; }
  .item .body { margin-top:6px; }
  .actions { display:flex; gap:8px; flex-wrap:wrap; margin-top:10px; align-items:center; }
  .actions form { margin:0; }
  .actions button { border:0; border-radius:999px; padding:6px 14px; font-size:14px;
    font-weight:600; color:#fff; cursor:pointer; background:#98a3b3; }
  .actions button.ok { background:#2e8b57; }
  .actions button.no { background:#e8548c; }
  .actions a.prev { font-size:14px; font-weight:600; color:#2f9fd0; text-decoration:none; }
  form.refine { margin-top:8px; display:flex; gap:8px; }
  form.refine input[type=text] { flex:1; font:14px/1.4 system-ui,sans-serif; padding:6px 10px;
    border:2px solid #cfd8e3; border-radius:8px; background:#fff; color:inherit; }
  form.refine button { border:0; border-radius:999px; padding:6px 14px; font-size:14px;
    font-weight:600; color:#fff; background:#2f9fd0; cursor:pointer; }
  pre.log { background:#f4f6f9; border-radius:10px; padding:10px 14px; font-size:12px;
    color:#6b7a8d; max-height:220px; overflow:auto; margin:10px 0 0; }
`

// itemTmpl renders one work item with its status and the admin actions.
const itemTmpl = `
{{define "item"}}
<li class="item">
  <div class="head">
    <span class="tag {{.Status}}">{{.Status}}</span>
    <span class="who">#{{.ID}} &middot; {{if .Author}}{{.Author}} &middot; {{end}}{{.CreatedAt.Format "2 Jan 2006, 15:04"}}</span>
  </div>
  <div class="body">{{.Body}}</div>
  {{if .Note}}<p class="note">{{.Note}}</p>{{end}}
  {{if or (eq .Status "queued") (eq .Status "working") (eq .Status "review") (eq .Status "failed")}}
  <div class="actions">
    {{if .HasPreview}}
    <a class="prev" href="/preview/{{.ID}}/" target="_blank">Open preview</a>
    {{end}}
    {{if eq .Status "review"}}
    <form method="POST" action="/work/approve"><input type="hidden" name="id" value="{{.ID}}">
      <button class="ok" type="submit">Approve &amp; push</button></form>
    {{end}}
    {{if eq .Status "failed"}}
    <form method="POST" action="/work/retry"><input type="hidden" name="id" value="{{.ID}}">
      <button type="submit">Retry</button></form>
    {{end}}
    <form method="POST" action="/work/reject"><input type="hidden" name="id" value="{{.ID}}">
      <button class="no" type="submit">Reject</button></form>
  </div>
  {{end}}
  {{if or (eq .Status "review") (eq .Status "failed")}}
  <form class="refine" method="POST" action="/work/refine">
    <input type="hidden" name="id" value="{{.ID}}">
    <input type="text" name="body" required placeholder="Request a refinement">
    <button type="submit">Refine</button>
  </form>
  {{end}}
</li>
{{end}}
`

var indexTmpl = template.Must(template.New("index").Parse(itemTmpl + `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
{{if .Busy}}<meta http-equiv="refresh" content="10">{{end}}
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
        {{range $rs}}{{template "item" .}}{{end}}
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
        {{range $rs}}{{template "item" .}}{{end}}
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

  {{if .Admin}}
  {{$completed := .CompletedRequests}}
  {{if $completed}}
  <div class="card">
    <h2>Recent completed work</h2>
    <p class="meta">Approved and rejected requests remain here after a restart.</p>
    <ul class="reqlist">
      {{range $completed}}{{template "item" .}}{{end}}
    </ul>
  </div>
  {{end}}
  {{end}}

  <div class="adminbar">
    <span>{{if .Admin}}Admin mode is on — requests are visible.{{else}}Admin mode is off.{{end}}</span>
    <form method="POST" action="/admin">
      <input type="hidden" name="admin" value="{{if .Admin}}off{{else}}on{{end}}">
      <button type="submit">{{if .Admin}}Disable admin mode{{else}}Enable admin mode{{end}}</button>
    </form>
    {{if .Admin}}
    <form method="POST" action="/work/rebuild"><button type="submit">Rebuild site</button></form>
    {{end}}
  </div>
  {{if and .Admin .Log}}<pre class="log">{{range .Log}}{{.}}
{{end}}</pre>{{end}}
{{end}}
</div>
</body>
</html>
`))
