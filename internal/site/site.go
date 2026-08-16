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
	Subject    string                 `json:"subject"`
	Name       string                 `json:"name"`
	Title      string                 `json:"title"`
	Date       string                 `json:"date"`
	Meta       string                 `json:"meta"`
	Version    string                 `json:"version,omitempty"`
	Owner      string                 `json:"-"`
	Visibility store.Visibility       `json:"-"`
	Shares     []store.WorksheetShare `json:"-"`
}

func (w Worksheet) Path() string { return w.Subject + "/" + w.Name }

// Private reports whether the worksheet can have explicit shares.
func (w Worksheet) Private() bool { return w.Visibility == store.VisibilityPrivate }

// Data is everything the front page needs.
type Data struct {
	Worksheets []Worksheet
	Requests   []store.Request // newest first
	Admin      bool
	// Static marks the offline copy written by cmd/generate: no forms,
	// no admin controls (there is no server to talk to).
	Static bool
	// User is the effective account email on the live site.
	User string
	// Actor is the account that signed in; it differs from User while impersonating.
	Actor string
	// CanImpersonate enables the admin-only account switcher.
	CanImpersonate bool
	// Users lists account emails available to the account switcher.
	Users []string
	// Flash is an optional confirmation message shown at the top.
	Flash string
	// Log holds recent pipeline events (admin only, newest first).
	Log []string
}

func (d Data) openRequests(kind store.Kind, worksheet string) []store.Request {
	var out []store.Request
	for _, r := range d.Requests {
		if r.Kind == kind && r.Worksheet == worksheet && r.Status.Open() {
			out = append(out, r)
		}
	}
	return out
}

// ActiveRequests returns all unfinished work, newest first.
func (d Data) ActiveRequests() []store.Request {
	var out []store.Request
	for _, r := range d.Requests {
		if r.Status.Open() {
			out = append(out, r)
		}
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

// CompletedRequests returns finished work items, newest first.
func (d Data) CompletedRequests() []store.Request {
	var out []store.Request
	for _, r := range d.Requests {
		if r.Status == store.StatusDone || r.Status == store.StatusRejected {
			out = append(out, r)
		}
	}
	return out
}

// RequestTitle gives a work item a short, recognisable heading.
func (d Data) RequestTitle(r store.Request) string {
	if r.Kind == store.KindNew {
		return "New worksheet"
	}
	for _, w := range d.Worksheets {
		if w.Path() == r.Worksheet {
			return w.Title
		}
	}
	return r.Worksheet
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

// Impersonating reports whether an administrator is viewing the site as another user.
func (d Data) Impersonating() bool {
	return d.Actor != "" && !strings.EqualFold(d.Actor, d.User)
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

func statusLabel(s store.Status) string {
	switch s {
	case store.StatusQueued:
		return "Queued"
	case store.StatusWorking:
		return "Work in progress"
	case store.StatusReview:
		return "Ready for review"
	case store.StatusFailed:
		return "Needs attention"
	case store.StatusDone:
		return "Published"
	case store.StatusRejected:
		return "Rejected"
	default:
		return string(s)
	}
}

func statusHelp(s store.Status) string {
	switch s {
	case store.StatusQueued:
		return "Waiting to start"
	case store.StatusWorking:
		return "The worksheet is being updated now"
	case store.StatusReview:
		return "The update is finished and awaiting approval"
	case store.StatusFailed:
		return "The update could not be completed"
	case store.StatusDone:
		return "The update is live"
	case store.StatusRejected:
		return "The update was not published"
	default:
		return ""
	}
}

const indexCSS = `
  :root { --ink:#172033; --muted:#667085; --line:#d8dee8; --link:#175cd3;
    --queued:#667085; --working:#b54708; --review:#175cd3; --failed:#b42318;
    --done:#067647; --rejected:#667085; }
  * { box-sizing:border-box; }
  body { margin:0; padding:36px 20px 64px; background:#fff; color:var(--ink);
    font:15px/1.45 system-ui,sans-serif; }
  .wrap { max-width:1040px; margin:0 auto; }
  header { display:flex; justify-content:space-between; align-items:baseline; gap:20px;
    padding-bottom:18px; border-bottom:1px solid var(--line); }
  h1 { font-size:26px; line-height:1.2; margin:0; }
  header p { color:var(--muted); margin:4px 0 0; }
  .account { display:flex; align-items:center; justify-content:flex-end; gap:10px; color:var(--muted); font-size:13px; flex-wrap:wrap; }
  .account form { margin:0; }
  .account button, .account select { padding:5px 9px; }
  .account select { border:1px solid #98a2b3; border-radius:3px; background:#fff; color:inherit; }
  .impersonation { color:#b54708; font-weight:650; }
  h2 { font-size:18px; margin:30px 0 10px; }
  .count { color:var(--muted); font-weight:400; }
  .flash { border:1px solid #86c9a8; color:#05603a; padding:10px 12px; margin:18px 0 0; }
  table { width:100%; border-collapse:collapse; border-top:1px solid var(--ink); }
  th { color:var(--muted); font-size:12px; text-align:left; text-transform:uppercase;
    letter-spacing:.04em; font-weight:600; padding:9px 10px; border-bottom:1px solid var(--line); }
  td { padding:13px 10px; border-bottom:1px solid var(--line); vertical-align:top; }
  .title { font-weight:650; }
  .meta, .date, .request-meta { color:var(--muted); font-size:13px; }
  .date { white-space:nowrap; }
  a { color:var(--link); }
  a.pdf { font-weight:650; white-space:nowrap; }
  .row-actions { text-align:right; white-space:nowrap; }
  .row-actions a { display:block; }
  .row-actions a + a { margin-top:4px; }
  .worksheet-main td { border-bottom:0; padding-bottom:5px; }
  .worksheet-request td { padding-top:0; }
  .worksheet-request details { width:100%; }
  .worksheet-request form.ask { max-width:none; }
  .sharing { display:flex; gap:18px; flex-wrap:wrap; align-items:flex-start; padding:8px 0 5px; }
  .sharing form { margin:0; }
  .sharing .visibility { display:flex; gap:7px; align-items:center; }
  .sharing select, .sharing input[type=email] { padding:7px 9px; border:1px solid #98a2b3;
    border-radius:3px; background:#fff; color:inherit; font:14px/1.4 system-ui,sans-serif; }
  .sharing input[type=email] { min-width:240px; }
  .share-form { display:flex; gap:7px; flex-wrap:wrap; }
  .share-list { width:100%; max-width:620px; margin:4px 0 0; padding:0; list-style:none; }
  .share-list li { display:flex; gap:8px; align-items:center; padding:6px 0; border-top:1px solid var(--line); }
  .share-list .email { flex:1; }
  .privacy { display:inline-block; margin-top:4px; color:var(--muted); font-size:12px; }
  details { margin-top:6px; }
  summary { color:var(--link); cursor:pointer; font-size:13px; }
  form.ask { max-width:620px; margin:10px 0 4px; display:grid; gap:8px; }
  textarea, input[type=text] { width:100%; padding:9px 10px; border:1px solid #98a2b3;
    border-radius:3px; background:#fff; color:inherit; font:14px/1.4 system-ui,sans-serif; }
  textarea { min-height:82px; resize:vertical; }
  button { border:1px solid #344054; border-radius:3px; background:#fff; color:#344054;
    padding:7px 11px; font-weight:600; cursor:pointer; }
  form.ask button { justify-self:start; color:#fff; background:#175cd3; border-color:#175cd3; }
  .status { display:inline-flex; align-items:center; gap:7px; color:var(--queued); font-weight:700; }
  .status::before { content:""; width:9px; height:9px; background:currentColor; border-radius:50%; }
  .status.working { color:var(--working); }
  .status.review { color:var(--review); }
  .status.failed { color:var(--failed); }
  .status.done { color:var(--done); }
  .status.rejected { color:var(--rejected); }
  .status-help { display:block; margin-top:2px; color:var(--muted); font-size:12px; }
  .active { border-top:3px solid var(--working); }
  .active td:first-child { width:190px; }
  .request-body { margin-top:5px; white-space:pre-wrap; }
  .request-note { margin:7px 0 0; color:var(--muted); font-size:13px; white-space:pre-wrap; }
  .actions { display:flex; gap:7px; flex-wrap:wrap; margin-top:10px; align-items:center; }
  .actions form { margin:0; }
  .actions button.ok { color:#fff; background:var(--done); border-color:var(--done); }
  .actions button.no { color:var(--failed); border-color:#f0b4ae; }
  .actions a { font-size:13px; font-weight:600; }
  form.refine { display:flex; gap:7px; margin-top:8px; }
  form.refine input { flex:1; }
  .new-request { margin:18px 0 0; padding:14px 0; border-top:1px solid var(--line);
    border-bottom:1px solid var(--line); }
  .new-request h2 { margin:0 0 4px; }
  .new-request p { color:var(--muted); margin:0 0 6px; }
  .adminbar { display:flex; gap:8px; align-items:center; flex-wrap:wrap; margin-top:34px;
    padding-top:14px; border-top:1px solid var(--line); color:var(--muted); font-size:13px; }
  .adminbar form { margin:0; }
  pre.log { background:#f7f8fa; border:1px solid var(--line); padding:10px; font-size:12px;
    color:var(--muted); max-height:220px; overflow:auto; }
  @media (max-width:720px) {
    body { padding:24px 12px 48px; }
    header { display:block; }
    .account { margin-top:10px; }
    header p { margin-top:4px; }
    table, tbody, tr, td { display:block; }
    thead { display:none; }
    tr { border-bottom:1px solid var(--line); padding:10px 0; }
    td { border:0; padding:3px 4px; }
    .row-actions { text-align:left; margin-top:5px; }
    .worksheet-main { border-bottom:0; padding-bottom:0; }
    .worksheet-request { padding-top:0; }
    .active td:first-child { width:auto; margin-bottom:5px; }
  }
`

const requestActions = `
{{define "actions"}}
  <div class="actions">
    {{if .HasPreview}}<a href="/preview/{{.ID}}/" target="_blank">Open preview</a>{{end}}
    {{if eq .Status "review"}}
    <form method="POST" action="/work/approve"><input type="hidden" name="id" value="{{.ID}}"><button class="ok" type="submit">Approve &amp; publish</button></form>
    {{end}}
    {{if eq .Status "failed"}}
    <form method="POST" action="/work/retry"><input type="hidden" name="id" value="{{.ID}}"><button type="submit">Retry</button></form>
    {{end}}
    <form method="POST" action="/work/reject"><input type="hidden" name="id" value="{{.ID}}"><button class="no" type="submit">Reject</button></form>
  </div>
  {{if or (eq .Status "review") (eq .Status "failed")}}
  <form class="refine" method="POST" action="/work/refine">
    <input type="hidden" name="id" value="{{.ID}}"><input type="text" name="body" required placeholder="Describe the refinement">
    <button type="submit">Refine</button>
  </form>
  {{end}}
{{end}}
`

var indexTmpl = template.Must(template.New("index").Funcs(template.FuncMap{
	"statusLabel": statusLabel,
	"statusHelp":  statusHelp,
}).Parse(requestActions + `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
{{if .Busy}}<meta http-equiv="refresh" content="10">{{end}}
<title>Learning material</title>
<style>{{.Fonts}}` + indexCSS + `</style>
</head>
<body><div class="wrap">
<header><div><h1>Worksheets</h1><p>PDF worksheets for printing</p></div>{{if .User}}<div class="account">{{if .Impersonating}}<span class="impersonation">Viewing as {{.User}}</span>{{else}}<span>{{.User}}</span>{{end}}{{if .CanImpersonate}}<form method="POST" action="/account/impersonate"><input type="hidden" name="next" value="/"><select name="email" aria-label="View site as user">{{range .Users}}<option value="{{.}}"{{if eq . $.User}} selected{{end}}>{{.}}</option>{{end}}</select><button type="submit">View as</button></form>{{if .Impersonating}}<form method="POST" action="/account/impersonate"><input type="hidden" name="next" value="/"><input type="hidden" name="email" value="{{.Actor}}"><button type="submit">Stop impersonating</button></form>{{end}}{{end}}<form method="POST" action="/account/sign-out"><button type="submit">Sign out</button></form></div>{{end}}</header>
{{if .Flash}}<div class="flash">{{.Flash}}</div>{{end}}

{{if not .Static}}
{{$active := .ActiveRequests}}
{{if $active}}
<section aria-labelledby="updates-heading">
<h2 id="updates-heading">Updates in progress <span class="count">({{len $active}})</span></h2>
<table class="active">
<thead><tr><th>State</th><th>Request</th><th>Last update</th></tr></thead>
<tbody>
{{range $active}}
<tr id="request-{{.ID}}">
  <td><span class="status {{.Status}}">{{statusLabel .Status}}</span><span class="status-help">{{statusHelp .Status}}</span></td>
  <td><div class="title">{{$.RequestTitle .}}</div><div class="request-body">{{.Body}}</div>
    <div class="request-meta">Request #{{.ID}}{{if .Author}} · {{.Author}}{{end}} · submitted {{.CreatedAt.Format "2 Jan 2006, 15:04"}}</div>
    {{if .Note}}<p class="request-note">{{.Note}}</p>{{end}}
    {{if $.Admin}}{{template "actions" .}}{{end}}
  </td>
  <td class="date">{{.UpdatedAt.Format "2 Jan 2006, 15:04"}}</td>
</tr>
{{end}}
</tbody></table>
</section>
{{end}}
{{end}}

<section aria-labelledby="worksheets-heading">
<h2 id="worksheets-heading">Available worksheets <span class="count">({{len .Worksheets}})</span></h2>
<table class="worksheets">
<thead><tr><th>Worksheet</th><th>Updated</th><th>Details</th>{{if not .Static}}<th>State</th>{{end}}<th></th></tr></thead>
<tbody>
{{range .Worksheets}}
{{$worksheet := .Path}}
<tr{{if not $.Static}} class="worksheet-main"{{end}}>
  <td><div class="title">{{.Title}}</div>{{if not $.Static}}<span class="privacy">{{if .Private}}Private{{else}}Public{{end}} · owner: {{.Owner}}</span>{{end}}</td>
  <td class="date">{{if .Date}}{{.Date}}{{else}}—{{end}}</td>
  <td class="meta">{{.Meta}}</td>
  {{if not $.Static}}<td>
    {{$rs := $.ChangeRequests .}}
    {{if $rs}}{{range $rs}}<span class="status {{.Status}}">{{statusLabel .Status}}</span><br>{{end}}{{else}}<span class="meta">Current</span>{{end}}
  </td>{{end}}
  <td class="row-actions"><a class="pdf" href="{{.Path}}/index.pdf{{if .Version}}?v={{.Version}}{{end}}">Worksheet PDF</a><a class="pdf" href="{{.Path}}/solutions.pdf{{if .Version}}?v={{.Version}}{{end}}">Solutions PDF</a></td>
</tr>
{{if not $.Static}}
<tr class="worksheet-request"><td colspan="5">
  <details><summary>Sharing settings</summary>
    <div class="sharing">
      <form class="visibility" method="POST" action="/worksheets/visibility">
        <input type="hidden" name="worksheet" value="{{.Path}}">
        <label for="visibility-{{.Name}}">Access</label>
        <select id="visibility-{{.Name}}" name="visibility">
          <option value="private"{{if .Private}} selected{{end}}>Private</option>
          <option value="public"{{if not .Private}} selected{{end}}>Public</option>
        </select>
        <button type="submit">Save</button>
      </form>
      {{if .Private}}
      <form class="share-form" method="POST" action="/worksheets/shares">
        <input type="hidden" name="worksheet" value="{{.Path}}">
        <input type="email" name="email" required placeholder="Existing user's email">
        <select name="permission" aria-label="Permission"><option value="view">Can view</option><option value="edit">Can edit</option></select>
        <button type="submit">Share</button>
      </form>
      {{if .Shares}}<ul class="share-list">{{range .Shares}}<li><span class="email">{{.Email}}</span><span>{{if eq .Permission "edit"}}Can edit{{else}}Can view{{end}}</span><form method="POST" action="/worksheets/shares/delete"><input type="hidden" name="worksheet" value="{{$worksheet}}"><input type="hidden" name="email" value="{{.Email}}"><button type="submit">Remove</button></form></li>{{end}}</ul>{{end}}
      {{else}}<span class="meta">Public discovery is not available yet.</span>{{end}}
    </div>
  </details>
  <details><summary>Request an update</summary>
    <form class="ask" method="POST" action="/requests">
      <input type="hidden" name="kind" value="change"><input type="hidden" name="worksheet" value="{{.Path}}">
      <textarea name="body" required placeholder="What should change?"></textarea>
      <input type="text" name="author" placeholder="Your name (optional)"><button type="submit">Send request</button>
    </form>
  </details>
</td></tr>
{{end}}
{{end}}
</tbody></table>
</section>

{{if not .Static}}
<section class="new-request">
<h2>Request a new worksheet</h2><p>Include the subject, level and type of tasks.</p>
<details><summary>Open request form</summary>
<form class="ask" method="POST" action="/requests">
  <input type="hidden" name="kind" value="new">
  <textarea name="body" required placeholder="Which subject, topic and level? What should the tasks look like?"></textarea>
  <input type="text" name="author" placeholder="Your name (optional)"><button type="submit">Send request</button>
</form></details>
</section>

{{if .Admin}}
{{$completed := .CompletedRequests}}{{if $completed}}
<section><h2>Recent completed work</h2>
<table><thead><tr><th>State</th><th>Request</th><th>Date</th></tr></thead><tbody>
{{range $completed}}<tr><td><span class="status {{.Status}}">{{statusLabel .Status}}</span></td>
<td><div class="title">{{$.RequestTitle .}}</div><div class="request-body">{{.Body}}</div>{{if .Note}}<p class="request-note">{{.Note}}</p>{{end}}</td>
<td class="date">{{.UpdatedAt.Format "2 Jan 2006, 15:04"}}</td></tr>{{end}}
</tbody></table></section>
{{end}}{{end}}

<div class="adminbar">
<span>{{if .Admin}}Admin controls are on.{{else}}Admin controls are off.{{end}}</span>
<form method="POST" action="/admin"><input type="hidden" name="admin" value="{{if .Admin}}off{{else}}on{{end}}"><button type="submit">{{if .Admin}}Turn off admin controls{{else}}Turn on admin controls{{end}}</button></form>
{{if .Admin}}<form method="POST" action="/work/rebuild"><button type="submit">Rebuild PDFs</button></form>{{end}}
</div>
{{if and .Admin .Log}}<pre class="log">{{range .Log}}{{.}}
{{end}}</pre>{{end}}
{{end}}
</div></body></html>
`))
