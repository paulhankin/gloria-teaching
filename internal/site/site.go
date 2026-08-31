// Package site renders the front page: the worksheet index plus the UI for
// worksheet requests (new worksheets and change requests).
package site

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"

	"learningmaterial/internal/sheet"
	"learningmaterial/internal/store"
)

type Worksheet struct {
	Username   string                 `json:"username"`
	Subject    string                 `json:"subject"`
	Name       string                 `json:"name"`
	Title      string                 `json:"title"`
	Date       string                 `json:"date"`
	Meta       string                 `json:"meta"`
	Version    string                 `json:"version,omitempty"`
	Owner      string                 `json:"-"`
	Visibility store.Visibility       `json:"-"`
	Finished   bool                   `json:"-"`
	Shares     []store.WorksheetShare `json:"-"`
}

// Revision is a Git-backed version of one worksheet.
type Revision struct {
	Commit  string
	Short   string
	Subject string
	Date    string
	Current bool
}

func (w Worksheet) Path() string { return w.Subject + "/" + w.Name }

// OutputPath is the generated-file location below the output root.
func (w Worksheet) OutputPath() string { return w.Username + "/" + w.Name }

// Private reports whether the worksheet can have explicit shares.
func (w Worksheet) Private() bool { return w.Visibility == store.VisibilityPrivate }

// Data is everything the front page needs.
type Data struct {
	Worksheets []Worksheet
	Revisions  map[string][]Revision // worksheet path -> newest first
	Requests   []store.Request       // newest first
	Admin      bool
	// Static marks the offline copy written by cmd/generate: no forms,
	// no admin controls (there is no server to talk to).
	Static bool
	// User is the effective account username on the live site.
	User string
	// Actor is the username that signed in; it differs from User while impersonating.
	Actor string
	// CanImpersonate enables the admin-only account switcher.
	CanImpersonate bool
	// Users lists account emails available to the account switcher.
	Users []string
	// Flash is an optional confirmation message shown at the top.
	Flash string
	// Log holds recent pipeline events (admin only, newest first).
	Log []string
	// Tags is the owner's navigation tree (top-level tags with children).
	Tags []store.Tag
	// WorksheetTags maps a worksheet path to its assigned tag IDs.
	WorksheetTags map[string]map[int64]bool
	// ActiveTagID selects the tag whose worksheets are shown; 0 = Home (all).
	ActiveTagID int64
	// ActiveTagName is the heading for a selected tag.
	ActiveTagName string
	// Manage renders the tag-management view instead of the worksheet list.
	Manage bool
	// FinishedView renders only the finished-worksheets list.
	FinishedView bool
	// PublicView renders only the public worksheets.
	PublicView bool
	// Lang is the UI language: "en" or "de" (defaults to English).
	Lang string
	// RequestPath is the current request path + query, so the language picker
	// can redirect back to the same page.
	RequestPath string
}

// T returns the UI string for key in the current language (d.Lang).
// Worksheet content itself stays German; only the surrounding site UI is
// translated.
func (d Data) T(key string) string { return translate(d.Lang, key) }

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

// ActiveWorksheets returns the worksheets still being worked on.
func (d Data) ActiveWorksheets() []Worksheet {
	var out []Worksheet
	for _, w := range d.Worksheets {
		if !w.Finished {
			out = append(out, w)
		}
	}
	return out
}

// FinishedWorksheets returns the worksheets filed away as finished.
func (d Data) FinishedWorksheets() []Worksheet {
	var out []Worksheet
	for _, w := range d.Worksheets {
		if w.Finished {
			out = append(out, w)
		}
	}
	return out
}

// activeTagIDs returns the selected tag plus its descendants, so clicking a
// top-level tag also matches worksheets tagged with one of its sub-tags.
func (d Data) activeTagIDs() map[int64]bool {
	ids := map[int64]bool{d.ActiveTagID: true}
	var walk func(tags []store.Tag)
	walk = func(tags []store.Tag) {
		for _, t := range tags {
			if t.ID == d.ActiveTagID {
				for _, c := range t.Children {
					ids[c.ID] = true
				}
			}
			walk(t.Children)
		}
	}
	walk(d.Tags)
	return ids
}

// ShownWorksheets returns the active worksheets for the current view: all on
// Home, otherwise those tagged with the selected tag or one of its sub-tags.
func (d Data) ShownWorksheets() []Worksheet {
	if d.ActiveTagID == 0 {
		return d.ActiveWorksheets()
	}
	ids := d.activeTagIDs()
	var out []Worksheet
	for _, w := range d.ActiveWorksheets() {
		for id := range d.WorksheetTags[w.Path()] {
			if ids[id] {
				out = append(out, w)
				break
			}
		}
	}
	return out
}

// PublicWorksheets returns the worksheets visible to anyone (not private),
// in the same order as the index.
func (d Data) PublicWorksheets() []Worksheet {
	var out []Worksheet
	for _, w := range d.Worksheets {
		if !w.Private() {
			out = append(out, w)
		}
	}
	return out
}

// ShownFinishedWorksheets is ShownWorksheets for the finished list.
func (d Data) ShownFinishedWorksheets() []Worksheet {
	if d.ActiveTagID == 0 {
		return d.FinishedWorksheets()
	}
	ids := d.activeTagIDs()
	var out []Worksheet
	for _, w := range d.FinishedWorksheets() {
		for id := range d.WorksheetTags[w.Path()] {
			if ids[id] {
				out = append(out, w)
				break
			}
		}
	}
	return out
}

// Heading is the main panel title for the current view.
func (d Data) Heading() string {
	if d.Manage {
		return d.T("nav.manage")
	}
	if d.FinishedView {
		return d.T("section.finished")
	}
	if d.PublicView {
		return d.T("section.public")
	}
	if d.ActiveTagID == 0 {
		return d.T("section.myworksheets")
	}
	return d.ActiveTagName
}

// SectionHeading titles the worksheet table below the page heading. On Home
// it reads "My worksheets"; on a category view it repeats the category.
func (d Data) SectionHeading() string {
	if d.FinishedView {
		return d.T("section.finished")
	}
	if d.ActiveTagID == 0 {
		return d.T("section.myworksheets")
	}
	return d.ActiveTagName
}

// TagChecked reports whether a worksheet carries the given tag.
func (d Data) TagChecked(worksheet string, tagID int64) bool {
	return d.WorksheetTags[worksheet][tagID]
}

// tagCheck is one labelled checkbox in a worksheet's tag form.
type tagCheck struct {
	ID      int64
	Name    string
	Checked bool
}

// tagForm pairs a worksheet with the tag checkboxes for the manage view.
type tagForm struct {
	Title  string
	Path   string
	Checks []tagCheck
}

// TagForms builds the per-worksheet tag assignment forms for the manage view.
func (d Data) TagForms() []tagForm {
	var flat []store.Tag
	var walk func(tags []store.Tag, depth int)
	walk = func(tags []store.Tag, depth int) {
		for _, t := range tags {
			name := t.Name
			if depth > 0 {
				name = "— " + name
			}
			flat = append(flat, store.Tag{ID: t.ID, Name: name})
			walk(t.Children, depth+1)
		}
	}
	walk(d.Tags, 0)
	out := make([]tagForm, 0, len(d.Worksheets))
	for _, w := range d.Worksheets {
		f := tagForm{Title: w.Title, Path: w.Path()}
		for _, t := range flat {
			f.Checks = append(f.Checks, tagCheck{ID: t.ID, Name: t.Name, Checked: d.TagChecked(w.Path(), t.ID)})
		}
		out = append(out, f)
	}
	return out
}

// rowSet carries the shared page data plus the worksheets of one index table
// into the worksheet-rows template.
type rowSet struct {
	Static bool
	Data   Data
	Rows   []Worksheet
}

// User returns the effective account username (needed inside range loops).
func (s rowSet) User() string { return s.Data.User }

// row wraps one worksheet with the page data for the per-row lookups.
type row struct {
	Worksheet
	data Data
}

// Revisions returns the worksheet's revision history.
func (r row) Revisions() []Revision { return r.data.Revisions[r.Path()] }

// ChangeRequests returns the open change requests for the worksheet.
func (r row) ChangeRequests() []store.Request { return r.data.ChangeRequests(r.Worksheet) }

// RequestTitle gives a work item a short, recognisable heading.
func (d Data) RequestTitle(r store.Request) string {
	if r.Kind == store.KindNew {
		return d.T("request.new")
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

// colorTitle renders the site name letter by letter with real 3D thickness:
// several copies of the letter are stacked at 1px offsets to form a solid
// extruded body, with a hatched edge and a light pastel front face on top.
// Each letter is a pastel crayon colour with a gentle rotation and lift.
// Spaces and emoji pass through untouched.
func colorTitle(s string) template.HTML {
	const depth = 7
	var b strings.Builder
	i := 0
	for _, r := range s {
		switch {
		case r == ' ':
			b.WriteString(`<span class="sp"></span>`)
			continue
		case r > 127:
			b.WriteString(`<span class="wand">` + string(r) + `</span>`)
			continue
		}
		cls := i%8 + 1
		ch := string(r)
		fmt.Fprintf(&b, `<span class="lt c%d">`, cls)
		// Solid 3D body: stacked copies offset down-right, back to front.
		for k := depth; k >= 1; k-- {
			fmt.Fprintf(&b, `<span class="d" style="--k:%d" aria-hidden="true">%s</span>`, k, ch)
		}
		// Hatched sheen on the depth, then the pastel front face and a light
		// pencil-stroke wash over it.
		fmt.Fprintf(&b, `<span class="hatch" style="--k:%d" aria-hidden="true">%s</span>`, depth, ch)
		fmt.Fprintf(&b, `<span class="face">%s</span>`, ch)
		fmt.Fprintf(&b, `<span class="shade" aria-hidden="true">%s</span></span>`, ch)
		i++
	}
	return template.HTML(b.String())
}

// rainbowText renders s letter by letter, each in the next rainbow colour, so
// the tagline looks hand-coloured. Spaces and emoji pass through untouched.
var rainbowColors = []string{"#e2574c", "#f0932b", "#e5c100", "#3da35d", "#2f8fd6", "#7a5fd0", "#d64f9b"}

func rainbowText(s string) template.HTML {
	var b strings.Builder
	i := 0
	for _, r := range s {
		if r == ' ' {
			b.WriteString(" ")
			continue
		}
		if r > 127 {
			b.WriteString(string(r))
			continue
		}
		fmt.Fprintf(&b, `<span style="color:%s">%s</span>`, rainbowColors[i%len(rainbowColors)], template.HTMLEscapeString(string(r)))
		i++
	}
	return template.HTML(b.String())
}

// paintStroke is a brush stroke with paint splashes that underlines the
// tagline, as if the paintbrush just swept the words onto the page. The
// colours echo the rainbow letters.
const paintStrokeSVG = `<svg class="paint-stroke" viewBox="0 0 360 26" fill="none" aria-hidden="true"><path d="M6 16 C 60 9, 150 20, 230 13 C 280 9, 320 14, 352 12" stroke="#f0932b" stroke-width="7" stroke-linecap="round" opacity=".45"/><path d="M4 18 C 70 12, 160 22, 250 15 C 300 11, 330 15, 356 13" stroke="#2f8fd6" stroke-width="3" stroke-linecap="round" opacity=".5"/><circle cx="40" cy="6" r="3" fill="#e2574c" opacity=".6"/><circle cx="300" cy="4" r="2.4" fill="#3da35d" opacity=".6"/><circle cx="330" cy="22" r="2.8" fill="#7a5fd0" opacity=".6"/><circle cx="120" cy="23" r="2" fill="#e2574c" opacity=".55"/><circle cx="210" cy="3" r="1.8" fill="#f0932b" opacity=".6"/></svg>`

func statusLabel(lang string, s store.Status) string {
	switch s {
	case store.StatusQueued:
		return translate(lang, "status.queued")
	case store.StatusWorking:
		return translate(lang, "status.working")
	case store.StatusReview:
		return translate(lang, "status.review")
	case store.StatusFailed:
		return translate(lang, "status.failed")
	case store.StatusDone:
		return translate(lang, "status.done")
	case store.StatusRejected:
		return translate(lang, "status.rejected")
	default:
		return string(s)
	}
}

func statusHelp(lang string, s store.Status) string {
	switch s {
	case store.StatusQueued:
		return translate(lang, "statushelp.queued")
	case store.StatusWorking:
		return translate(lang, "statushelp.working")
	case store.StatusReview:
		return translate(lang, "statushelp.review")
	case store.StatusFailed:
		return translate(lang, "statushelp.failed")
	case store.StatusDone:
		return translate(lang, "statushelp.done")
	case store.StatusRejected:
		return translate(lang, "statushelp.rejected")
	default:
		return ""
	}
}

const indexCSS = `
  :root { --ink:#172033; --muted:#667085; --line:#d8dee8; --link:#175cd3;
    --queued:#667085; --working:#b54708; --review:#175cd3; --failed:#b42318;
    --done:#067647; --rejected:#667085; }
  * { box-sizing:border-box; }
  /* Graph-paper (kariert) background, like a school notebook page. */
  body { margin:0; color:var(--ink);
    font:15px/1.45 system-ui,sans-serif;
    background-color:#ffffff;
    background-image:
      linear-gradient(to right, rgba(150,190,235,.14) 1px, transparent 1px),
      linear-gradient(to bottom, rgba(150,190,235,.14) 1px, transparent 1px);
    background-size:26px 26px; }
  .layout { display:flex; min-height:100vh; }
  .sidebar { width:230px; flex-shrink:0; border-right:1px solid var(--line);
    background:#fff;
    padding:20px 0; position:sticky; top:0; height:100vh; overflow-y:auto; }
  .sidebar .brand { padding:0 18px 16px; font-weight:700; font-size:16px;
    border-bottom:1px solid var(--line); margin-bottom:10px; color:var(--ink); }
  .nav-item { display:flex; align-items:center; gap:10px; padding:9px 18px;
    color:var(--ink); text-decoration:none; font-size:14px; }
  .nav-item svg { width:17px; height:17px; flex-shrink:0; color:var(--muted); }
  .nav-item:hover { background:#f4f6f9; }
  .nav-item.active { background:#eaf1fd; color:var(--link); font-weight:650; }
  .nav-item.active svg { color:var(--link); }
  .nav-item.sub { padding-left:36px; font-size:13px; color:var(--muted); }
  .nav-item.sub.active { color:var(--link); }
  .nav-section { padding:14px 18px 5px; color:var(--muted); font-size:11px;
    text-transform:uppercase; letter-spacing:.05em; font-weight:650; }
  .main { flex:1; min-width:0; padding:30px 30px 64px; }
  .wrap { max-width:1040px; margin:0 auto; }
  /* Slim top bar holding the account controls, above the full-width title. */
  .topbar { display:flex; justify-content:flex-end; align-items:center;
    padding:2px 0 14px; }
  header { display:flex; justify-content:center; text-align:center;
    padding:6px 6px 22px; border-bottom:2px solid var(--line); }
  /* Playful, colour-pencil brand. The name is drawn letter by letter in a
     hand-drawn face with a crayon palette and a gentle wobble. */
  .brand-title { font-family:"Arial Black","Helvetica Neue",Arial,sans-serif;
    font-size:min(62px,6.2vw); line-height:.9; margin:0; font-weight:900; letter-spacing:1.5px;
    text-transform:uppercase; white-space:nowrap;
    display:flex; align-items:center; justify-content:center;
    padding:0 4px 16px 4px; }
  .brand-gap { display:inline-block; width:.28em; }
  .brand-logo { height:1.5em; width:auto; flex-shrink:0;
    transform:translateY(.06em); }
  .brand-title .lt { position:relative; display:inline-block;
    -webkit-text-stroke:1.6px #20242e; }
  .brand-title .lt .d { position:absolute; left:calc(var(--k)*1px); top:calc(var(--k)*1px);
    z-index:0; color:#20242e; -webkit-text-stroke:0; }
  .brand-title .lt .hatch { position:absolute; left:calc(var(--k)*1px); top:calc(var(--k)*1px);
    z-index:1; -webkit-text-stroke:0; color:transparent;
    background:repeating-linear-gradient(45deg, #3a3f4d 0 2px, transparent 2px 4.5px);
    -webkit-background-clip:text; background-clip:text; opacity:.75; }
  .brand-title .lt .shade { display:none; }
  /* Monochrome 3D: white faces, a black outline, and a black 3D body whose
     depth is shaded with diagonal pencil hatching in dark grey. */
  .brand-title .lt .face { position:relative; z-index:2; color:#fff; }
  /* Hand-drawn squiggle underline, like a crayon stroke gone wavy. */
  .brand-sub { margin:14px 0 0; font-size:30px; font-weight:800;
    font-family:Caveat,"Segoe Print","Bradley Hand",cursive; letter-spacing:.5px; }
  /* The tagline sits on a paint stroke with splashes, with a brush at the end
     as if it just painted the words. */
  .paint-line { position:relative; display:inline-block; padding:0 6px 14px 6px; }
  .paint-line .paint-stroke { position:absolute; left:0; right:0; bottom:0;
    width:100%; height:26px; }
  .paintbrush { display:inline-block; font-size:30px; margin-left:6px;
    transform:rotate(-28deg) translateY(-6px); }
  .account { display:flex; align-items:center; justify-content:flex-end; gap:10px; color:var(--muted); font-size:13px; flex-wrap:wrap; }
  .account form { margin:0; }
  .account button, .account .impersonate-username { padding:5px 9px; }
  .account .impersonate-username { width:230px; border:1px solid #98a2b3; border-radius:3px; background:#fff; color:inherit; font:inherit; }
  .impersonation { color:#b54708; font-weight:650; }
  /* Language picker (EN/DE) at the far right of the top bar. */
  .lang-picker { display:flex; gap:2px; margin-left:6px; }
  .lang-picker .lang { padding:5px 8px; border:1px solid #c6cfdd; background:#fff;
    color:var(--muted); cursor:pointer; font-weight:650; font-size:12px;
    display:inline-flex; align-items:center; gap:5px; }
  .lang-picker .lang .flag { width:18px; height:12px; border-radius:2px;
    box-shadow:0 0 0 1px rgba(0,0,0,.12); flex-shrink:0; }
  .lang-picker .lang:first-child { border-radius:5px 0 0 5px; }
  .lang-picker .lang:last-child { border-radius:0 5px 5px 0; }
  .lang-picker .lang.on { background:#2f6fae; border-color:#2f6fae; color:#fff; }
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
  .row-actions form { margin:6px 0 0; }
  button.finish { width:100%; color:#fff; background:var(--done); border-color:var(--done); }
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
  .finished-heading { margin-top:6px; }
  .finished-heading summary { font-size:18px; font-weight:650; color:var(--ink); }
  .finished-heading .count { font-size:15px; font-weight:400; }
  .revision-list { width:100%; max-width:720px; margin:8px 0 2px; padding:0; list-style:none; }
  .revision-list li { display:flex; gap:10px; align-items:center; padding:7px 0; border-top:1px solid var(--line); }
  .revision-list .revision-info { flex:1; min-width:0; }
  .revision-list code { font-size:12px; }
  .revision-list .current { color:var(--done); font-size:12px; font-weight:700; }
  .revision-list form { margin:0; }
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
  table.active { border-top:3px solid var(--working); }
  table.active td:first-child { width:190px; }
  .request-body { margin-top:5px; white-space:pre-wrap; }
  .request-note { margin:7px 0 0; color:var(--muted); font-size:13px; white-space:pre-wrap; }
  .actions { display:flex; gap:7px; flex-wrap:wrap; margin-top:10px; align-items:center; }
  .actions form { margin:0; }
  .actions button.ok { color:#fff; background:var(--done); border-color:var(--done); }
  .actions button.no { color:var(--failed); border-color:#f0b4ae; }
  .actions a { font-size:13px; font-weight:600; }
  form.refine { display:flex; gap:7px; margin-top:8px; }
  form.refine input { flex:1; }
  .new-request { margin:34px 0 22px; padding:0 0 18px; border-bottom:1px solid var(--line); }
  .new-request h2 { margin:0 0 10px; }
  .adminbar { display:flex; gap:8px; align-items:center; flex-wrap:wrap; margin-top:34px;
    padding-top:14px; border-top:1px solid var(--line); color:var(--muted); font-size:13px; }
  .adminbar form { margin:0; }
  h3 { font-size:15px; margin:26px 0 8px; }
  form.tag-add { display:flex; gap:7px; flex-wrap:wrap; max-width:620px; }
  form.tag-add input[type=text] { flex:1; min-width:200px; }
  form.tag-add select { padding:9px 10px; border:1px solid #98a2b3; border-radius:3px;
    background:#fff; color:inherit; font:14px/1.4 system-ui,sans-serif; }
  .tag-manage-list { list-style:none; padding:0; margin:8px 0; max-width:640px; }
  .tag-manage-list li { padding:7px 0; border-top:1px solid var(--line); }
  .tag-manage-list ul { list-style:none; padding-left:26px; margin:4px 0 0; }
  .tag-manage-list .tag-name { font-weight:650; display:inline-block; min-width:140px; }
  .tag-manage-list .tag-name.sub { color:var(--muted); font-weight:500; }
  form.tag-rename, form.tag-delete { display:inline-flex; gap:6px; margin-left:8px; }
  form.tag-rename input[type=text] { width:150px; padding:4px 7px; }
  form.tag-rename button, form.tag-delete button { padding:4px 9px; font-size:13px; }
  button.no { color:var(--failed); border-color:#f0b4ae; }
  .tag-worksheet { border-top:1px solid var(--line); padding:10px 0; max-width:720px; }
  .tag-assign { display:flex; gap:14px; flex-wrap:wrap; align-items:center; margin-top:5px; }
  .tag-check { display:inline-flex; gap:6px; align-items:center; font-size:14px; color:var(--ink); }
  pre.log { background:#f7f8fa; border:1px solid var(--line); padding:10px; font-size:12px;
    color:var(--muted); max-height:220px; overflow:auto; }
  @media (max-width:720px) {
    .layout { display:block; }
    .sidebar { width:auto; height:auto; position:static; border-right:0;
      border-bottom:1px solid var(--line); padding:12px 0; }
    .main { padding:20px 12px 48px; }
    .topbar { justify-content:flex-start; }
    .brand-title { font-size:clamp(26px,9vw,40px); white-space:normal; }
    table, tbody, tr, td { display:block; }
    thead { display:none; }
    tr { border-bottom:1px solid var(--line); padding:10px 0; }
    td { border:0; padding:3px 4px; }
    .row-actions { text-align:left; margin-top:5px; }
    .worksheet-main { border-bottom:0; padding-bottom:0; }
    .worksheet-request { padding-top:0; }
    table.active td:first-child { width:auto; margin-bottom:5px; }
  }
`

const worksheetRows = `
{{define "worksheet-rows"}}
{{$static := .Static}}
{{range $i, $w := .Rows}}
{{with $r := row $ $w}}
{{$worksheet := $r.Path}}
<tr{{if not $static}} class="worksheet-main"{{end}}>
  <td><div class="title">{{$r.Title}}</div>{{if not $static}}<span class="privacy">{{if $r.Private}}{{T . "settings.private"}}{{else}}{{T . "settings.public"}}{{end}} · {{T . "label.owner"}}: {{$r.Owner}}</span>{{end}}</td>
  <td class="date">{{if $r.Date}}{{$r.Date}}{{else}}—{{end}}</td>
  <td class="meta">{{$r.Meta}}</td>
  {{if not $static}}<td>
    {{$rs := $r.ChangeRequests}}
    {{if $rs}}{{range $rs}}<span class="status {{.Status}}">{{statusLabel $.Data.Lang .Status}}</span><br>{{end}}{{else}}<span class="meta">{{T . "link.current"}}</span>{{end}}
  </td>{{end}}
  <td class="row-actions"><a class="pdf" href="{{$r.OutputPath}}/index.pdf{{if $r.Version}}?v={{$r.Version}}{{end}}">{{T . "link.worksheetpdf"}}</a><a class="pdf" href="{{$r.OutputPath}}/solutions.pdf{{if $r.Version}}?v={{$r.Version}}{{end}}">{{T . "link.solutionspdf"}}</a>{{if not $static}}<form method="POST" action="/worksheets/finished"><input type="hidden" name="worksheet" value="{{$r.Path}}"><input type="hidden" name="finished" value="{{if $r.Finished}}off{{else}}on{{end}}"><button class="finish" type="submit">{{if $r.Finished}}{{T . "action.moveback"}}{{else}}{{T . "action.markfinished"}}{{end}}</button></form>{{end}}</td>
</tr>
{{if not $static}}
<tr class="worksheet-request"><td colspan="5">
  {{if $r.Finished}}
  {{else}}
  {{$revisions := $r.Revisions}}
  {{if $revisions}}<details><summary>{{T . "history.title"}}</summary>
    <ul class="revision-list">{{range $revisions}}<li>
      <div class="revision-info"><code>{{.Short}}</code> · {{.Date}} · {{.Subject}}</div>
      {{if .Current}}<span class="current">{{T . "link.current"}}</span>{{else}}<form method="POST" action="/worksheets/revert"><input type="hidden" name="worksheet" value="{{$worksheet}}"><input type="hidden" name="commit" value="{{.Commit}}"><button type="submit">{{T . "history.revert"}}</button></form>{{end}}
    </li>{{end}}</ul>
  </details>{{end}}
  <details><summary>{{T . "settings.title"}}</summary>
    <div class="sharing">
      <form class="visibility" method="POST" action="/worksheets/visibility">
        <input type="hidden" name="worksheet" value="{{$r.Path}}">
        <label for="visibility-{{$r.Name}}">{{T . "settings.access"}}</label>
        <select id="visibility-{{$r.Name}}" name="visibility">
          <option value="private"{{if $r.Private}} selected{{end}}>{{T . "settings.private"}}</option>
          <option value="public"{{if not $r.Private}} selected{{end}}>{{T . "settings.public"}}</option>
        </select>
        <button type="submit">{{T . "settings.save"}}</button>
      </form>
      {{if $r.Private}}
      <form class="share-form" method="POST" action="/worksheets/shares">
        <input type="hidden" name="worksheet" value="{{$r.Path}}">
        <input type="email" name="email" required placeholder="{{T . "ph.useremail"}}">
        <select name="permission" aria-label="Permission"><option value="view">{{T . "settings.canview"}}</option><option value="edit">{{T . "settings.canedit"}}</option></select>
        <button type="submit">{{T . "settings.share"}}</button>
      </form>
      {{if $r.Shares}}<ul class="share-list">{{range $r.Shares}}<li><span class="email">{{.Email}}</span><span>{{if eq .Permission "edit"}}Can edit{{else}}Can view{{end}}</span><form method="POST" action="/worksheets/shares/delete"><input type="hidden" name="worksheet" value="{{$worksheet}}"><input type="hidden" name="email" value="{{.Email}}"><button type="submit">{{T . "settings.remove"}}</button></form></li>{{end}}</ul>{{end}}
      {{else}}<span class="meta">{{T . "settings.listed"}} <a href="/worksheets/{{$.User}}/index">{{T . "settings.publicpage"}}</a>.</span>{{end}}
    </div>
  </details>
  <details><summary>{{T . "settings.requestupdate"}}</summary>
    <form class="ask" method="POST" action="/requests">
      <input type="hidden" name="kind" value="change"><input type="hidden" name="worksheet" value="{{$r.Path}}">
      <textarea name="body" required placeholder="{{T . "ph.whatchange"}}"></textarea>
      <button type="submit">{{T . "button.sendrequest"}}</button>
    </form>
  </details>
  {{end}}
</td></tr>
{{end}}
{{end}}
{{end}}
{{end}}
`

const manageView = `
{{define "manage"}}
<section aria-labelledby="manage-heading">
<h2 id="manage-heading">{{T . "tags.manage"}}</h2>
<p class="meta">Categories appear in the left-hand menu. Create a top-level category such as "First Grade", then add sub-categories like "Mathematics" beneath it. Tag worksheets to file them under a category.</p>

<h3>{{T . "tags.add"}}</h3>
<form class="ask tag-add" method="POST" action="/tags">
  <input type="text" name="name" required placeholder="{{T . "ph.catname"}}">
  <select name="parent" aria-label="Parent category">
    <option value="0">{{T . "tags.toplevel"}}</option>
    {{range .Tags}}<option value="{{.ID}}">{{.Name}}</option>{{end}}
  </select>
  <button type="submit">{{T . "tags.addbutton"}}</button>
</form>

{{if .Tags}}
<h3>{{T . "tags.yours"}}</h3>
<ul class="tag-manage-list">
{{range .Tags}}
  <li>
    <span class="tag-name">{{.Name}}</span>
    <form class="tag-rename" method="POST" action="/tags/rename"><input type="hidden" name="id" value="{{.ID}}"><input type="text" name="name" value="{{.Name}}" required><button type="submit">{{T . "tags.rename"}}</button></form>
    <form class="tag-delete" method="POST" action="/tags/delete" onsubmit="return confirm('Delete category &quot;{{.Name}}&quot; and its sub-categories? Worksheets keep their other categories.');"><input type="hidden" name="id" value="{{.ID}}"><button type="submit" class="no">{{T . "tags.delete"}}</button></form>
    {{if .Children}}<ul>{{range .Children}}
      <li><span class="tag-name sub">{{.Name}}</span>
        <form class="tag-rename" method="POST" action="/tags/rename"><input type="hidden" name="id" value="{{.ID}}"><input type="text" name="name" value="{{.Name}}" required><button type="submit">{{T . "tags.rename"}}</button></form>
        <form class="tag-delete" method="POST" action="/tags/delete"><input type="hidden" name="id" value="{{.ID}}"><button type="submit" class="no">{{T . "tags.delete"}}</button></form>
      </li>{{end}}</ul>{{end}}
  </li>
{{end}}
</ul>
{{end}}

<h3>{{T . "tags.tagworksheets"}}</h3>
{{if .Tags}}
<p class="meta">{{T . "tags.tick"}}</p>
{{range .TagForms}}
<div class="tag-worksheet">
  <div class="title">{{.Title}}</div>
  <form method="POST" action="/worksheets/tags" class="tag-assign">
    <input type="hidden" name="worksheet" value="{{.Path}}">
    {{range .Checks}}<label class="tag-check"><input type="checkbox" name="tag" value="{{.ID}}"{{if .Checked}} checked{{end}}> {{.Name}}</label>{{end}}
    <button type="submit">{{T . "settings.save"}}</button>
  </form>
</div>
{{end}}
{{else}}<p class="meta">{{T . "tags.createfirst"}}</p>{{end}}
</section>
{{end}}
`

const requestActions = `
{{define "actions"}}
  <div class="actions">
    {{if eq .Status "failed"}}
    <form method="POST" action="/work/retry"><input type="hidden" name="id" value="{{.ID}}"><button type="submit">{{T . "action.retry"}}</button></form>
    {{end}}
    <form method="POST" action="/work/reject"><input type="hidden" name="id" value="{{.ID}}"><button class="no" type="submit">{{T . "action.reject"}}</button></form>
  </div>
  {{if and .Admin (eq .Status "failed")}}
  <form class="refine" method="POST" action="/work/refine">
    <input type="hidden" name="id" value="{{.ID}}"><input type="text" name="body" required placeholder="{{T . "ph.refine"}}">
    <button type="submit">{{T . "action.refine"}}</button>
  </form>
  {{end}}
{{end}}
`

// Inline SVG icons for the sidebar navigation (Feather-style strokes).
const (
	iconHomeSVG  = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path><polyline points="9 22 9 12 15 12 15 22"></polyline></svg>`
	iconGlobeSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>`
	iconCheckSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>`
	iconCogSVG   = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>`
	// flagEN is the Union Jack; flagDE is half German tricolour, half Swiss flag.
	flagENSVG = `<svg class="flag" viewBox="0 0 60 40" aria-hidden="true"><rect width="60" height="40" fill="#012169"/><path d="M0,0 L60,40 M60,0 L0,40" stroke="#fff" stroke-width="8"/><path d="M0,0 L60,40 M60,0 L0,40" stroke="#C8102E" stroke-width="4"/><path d="M30,0 V40 M0,20 H60" stroke="#fff" stroke-width="13"/><path d="M30,0 V40 M0,20 H60" stroke="#C8102E" stroke-width="8"/></svg>`
	flagDESVG = `<svg class="flag" viewBox="0 0 60 40" aria-hidden="true"><rect x="0" y="0" width="30" height="13.33" fill="#000"/><rect x="0" y="13.33" width="30" height="13.33" fill="#DD0000"/><rect x="0" y="26.66" width="30" height="13.34" fill="#FFCC00"/><rect x="30" y="0" width="30" height="40" fill="#DA291C"/><rect x="42" y="9" width="6" height="22" fill="#fff"/><rect x="34" y="17" width="22" height="6" fill="#fff"/><line x1="30" y1="0" x2="30" y2="40" stroke="#999" stroke-width="1"/></svg>`
)

// brandLogoImg is the Teacher's Friend logo (the handshake image), embedded
// and served at /logo.png. The URL carries a content fingerprint so a new logo
// busts the browser cache automatically. Static copies inline it as a data URL.
var brandLogoSrc = "/logo.png?v=" + func() string {
	sum := sha256.Sum256(sheet.AssetBytes("logo.png"))
	return fmt.Sprintf("%x", sum[:6])
}()

func brandLogoImg(static bool) template.HTML {
	src := brandLogoSrc
	if static {
		src = "data:image/png;base64," + base64.StdEncoding.EncodeToString(sheet.AssetBytes("logo.png"))
	}
	return template.HTML(`<img class="brand-logo" src="` + src + `" alt="Teacher's Friend logo: two hands shaking beneath a heart">`)
}

// reqActions bundles a request with whether the viewer may use the
// admin-only actions (refine), so the actions template can decide.
type reqActions struct {
	store.Request
	Admin bool
}

var indexTmpl = template.Must(template.New("index").Funcs(template.FuncMap{
	"statusLabel": statusLabel,
	"statusHelp":  statusHelp,
	"iconHome":    func() template.HTML { return template.HTML(iconHomeSVG) },
	"iconGlobe":   func() template.HTML { return template.HTML(iconGlobeSVG) },
	"iconCheck":   func() template.HTML { return template.HTML(iconCheckSVG) },
	"iconCog":     func() template.HTML { return template.HTML(iconCogSVG) },
	"flagEN":      func() template.HTML { return template.HTML(flagENSVG) },
	"flagDE":      func() template.HTML { return template.HTML(flagDESVG) },
	"colorTitle":  colorTitle,
	"rainbowText": rainbowText,
	"paintStroke": func() template.HTML { return template.HTML(paintStrokeSVG) },
	// T translates a UI key into the page's language: {{T . "nav.home"}}. It
	// accepts Data, row or rowSet (the context inside table rows), all of which
	// carry the language.
	"T": func(v any, key string) string {
		switch d := v.(type) {
		case Data:
			return d.T(key)
		case row:
			return d.data.T(key)
		case rowSet:
			return d.Data.T(key)
		}
		return translate("en", key)
	},
	"brandLogo": brandLogoImg,
	"rowSet": func(d Data, ws []Worksheet) rowSet {
		return rowSet{Static: d.Static, Data: d, Rows: ws}
	},
	"reqActions": func(d Data, r store.Request) reqActions {
		return reqActions{r, d.Admin}
	},
	"row": func(s rowSet, w Worksheet) row {
		return row{Worksheet: w, data: s.Data}
	},
}).Parse(worksheetRows + requestActions + manageView + `<!DOCTYPE html>
<html lang="{{if eq .Lang "de"}}de{{else}}en{{end}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
{{if .Busy}}<meta http-equiv="refresh" content="10">{{end}}
<title>{{T . "page.title"}}</title>
<style>{{.Fonts}}` + indexCSS + `</style>
</head>
<body>
{{if .Static}}
<div class="wrap">
<header><div class="brand-block"><h1 class="brand-title">{{colorTitle "Teacher's"}}<span class="brand-gap"></span>{{brandLogo .Static}}<span class="brand-gap"></span>{{colorTitle "Friend"}}</h1><p class="brand-sub"><span class="paint-line">{{rainbowText "Unleash your creativity!"}}<span class="paintbrush" aria-hidden="true">🖌️</span>{{paintStroke}}</span></p></div></header>
{{else}}
<div class="layout">
<nav class="sidebar" aria-label="Worksheet navigation">
  <div class="brand">{{if .User}}{{.User}}{{else}}{{T . "page.title"}}{{end}}</div>
  <a class="nav-item{{if and (not .Manage) (not .FinishedView) (not .PublicView) (eq .ActiveTagID 0)}} active{{end}}" href="/">{{iconHome}} {{T . "nav.home"}}</a>
  <a class="nav-item{{if .PublicView}} active{{end}}" href="/?public=1">{{iconGlobe}} {{T . "nav.public"}}</a>
  <a class="nav-item{{if .FinishedView}} active{{end}}" href="/?finished=1">{{iconCheck}} {{T . "nav.finished"}}</a>
  <a class="nav-item{{if .Manage}} active{{end}}" href="/?manage=1">{{iconCog}} {{T . "nav.manage"}}</a>
  {{if .Tags}}
  <div class="nav-section">{{T . "section.myworksheets"}}</div>
  {{range .Tags}}
  <a class="nav-item{{if eq $.ActiveTagID .ID}} active{{end}}" href="/?tag={{.ID}}">{{.Name}}</a>
  {{range .Children}}<a class="nav-item sub{{if eq $.ActiveTagID .ID}} active{{end}}" href="/?tag={{.ID}}">{{.Name}}</a>{{end}}
  {{end}}
  {{end}}
</nav>
<div class="main"><div class="wrap">
{{if .User}}<div class="topbar"><div class="account">{{if .Impersonating}}<span class="impersonation">{{T . "topbar.viewingas"}} {{.User}}</span>{{end}}{{if .CanImpersonate}}<form method="POST" action="/account/impersonate"><input type="hidden" name="next" value="/"><input class="impersonate-username" type="text" name="username" list="impersonation-users" required placeholder="{{T . "ph.username"}}" aria-label="{{T . "ph.viewasuser"}}"><datalist id="impersonation-users">{{range .Users}}<option value="{{.}}">{{end}}</datalist><button type="submit">{{T . "topbar.viewas"}}</button></form>{{if .Impersonating}}<form method="POST" action="/account/impersonate"><input type="hidden" name="next" value="/"><input type="hidden" name="username" value="{{.Actor}}"><button type="submit">{{T . "topbar.stop"}}</button></form>{{end}}{{end}}<a href="/worksheets/{{.User}}/index">{{T . "topbar.publicpage"}}</a><form method="POST" action="/account/sign-out"><button type="submit">{{T . "topbar.signout"}}</button></form><form class="lang-picker" method="POST" action="/language"><input type="hidden" name="next" value="{{$.RequestPath}}"><button type="submit" name="lang" value="en" class="lang{{if ne .Lang "de"}} on{{end}}" aria-label="English">{{flagEN}}EN</button><button type="submit" name="lang" value="de" class="lang{{if eq .Lang "de"}} on{{end}}" aria-label="Deutsch (Schweiz)">{{flagDE}}DE</button></form></div></div>{{end}}
<header><div class="brand-block"><h1 class="brand-title">{{colorTitle "Teacher's"}}<span class="brand-gap"></span>{{brandLogo .Static}}<span class="brand-gap"></span>{{colorTitle "Friend"}}</h1><p class="brand-sub"><span class="paint-line">{{rainbowText "Unleash your creativity!"}}<span class="paintbrush" aria-hidden="true">🖌️</span>{{paintStroke}}</span></p></div></header>
{{end}}
{{if .Flash}}<div class="flash">{{.Flash}}</div>{{end}}

{{if and .Manage (not .Static)}}
{{template "manage" .}}
{{else}}

{{if and .FinishedView (not .Static)}}
<section aria-labelledby="worksheets-heading">
<h2 id="worksheets-heading">{{T . "section.finished"}} <span class="count">({{len .FinishedWorksheets}})</span></h2>
{{if .FinishedWorksheets}}
<table class="worksheets">
<thead><tr><th>{{T . "col.worksheet"}}</th><th>{{T . "col.updated"}}</th><th>{{T . "col.details"}}</th><th>{{T . "col.state"}}</th><th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . .FinishedWorksheets}}</tbody>
</table>
{{else}}<p class="meta">{{T . "empty.finished"}}</p>{{end}}
</section>
{{else if and .PublicView (not .Static)}}
<section aria-labelledby="worksheets-heading">
<h2 id="worksheets-heading">{{T . "section.public"}} <span class="count">({{len .PublicWorksheets}})</span></h2>
<p class="meta">{{T . "public.visible"}} <a href="/worksheets/{{.User}}/index">{{T . "settings.publicpage"}}</a>.</p>
{{if .PublicWorksheets}}
<table class="worksheets">
<thead><tr><th>{{T . "col.worksheet"}}</th><th>{{T . "col.updated"}}</th><th>{{T . "col.details"}}</th><th>{{T . "col.state"}}</th><th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . .PublicWorksheets}}</tbody>
</table>
{{else}}<p class="meta">{{T . "empty.public"}}</p>{{end}}
</section>
{{else}}

{{if and (not .Static) (not .FinishedView)}}
<section class="new-request">
<h2>{{T . "prompt.newrequest"}}</h2>
<form class="ask" method="POST" action="/requests">
  <input type="hidden" name="kind" value="new">
  <textarea name="body" required placeholder="{{T . "ph.newworksheet"}}"></textarea>
  <button type="submit">{{T . "button.sendrequest"}}</button>
</form>
</section>
{{end}}

{{if not .Static}}
{{$active := .ActiveRequests}}
{{if $active}}
<section aria-labelledby="updates-heading">
<h2 id="updates-heading">{{T . "updates.inprogress"}} <span class="count">({{len $active}})</span></h2>
<table class="active">
<thead><tr><th>{{T . "col.state"}}</th><th>{{T . "col.request"}}</th><th>{{T . "col.lastupdate"}}</th></tr></thead>
<tbody>
{{range $active}}
<tr id="request-{{.ID}}">
  <td><span class="status {{.Status}}">{{statusLabel $.Lang .Status}}</span><span class="status-help">{{statusHelp $.Lang .Status}}</span></td>
  <td><div class="title">{{$.RequestTitle .}}</div><div class="request-body">{{.Body}}</div>
    <div class="request-meta">Request #{{.ID}}{{if .Author}} · {{.Author}}{{else if .Requester}} · {{.Requester}}{{end}} · submitted {{.CreatedAt.Format "2 Jan 2006, 15:04"}}</div>
    {{if .Note}}<p class="request-note">{{.Note}}</p>{{end}}
    {{template "actions" (reqActions $ .)}}
  </td>
  <td class="date">{{.UpdatedAt.Format "2 Jan 2006, 15:04"}}</td>
</tr>
{{end}}
</tbody></table>
</section>
{{end}}
{{end}}

<section aria-labelledby="worksheets-heading">
<h2 id="worksheets-heading">{{.SectionHeading}} <span class="count">({{len .ShownWorksheets}})</span></h2>
<table class="worksheets">
<thead><tr><th>{{T . "col.worksheet"}}</th><th>{{T . "col.updated"}}</th><th>{{T . "col.details"}}</th>{{if not .Static}}<th>{{T . "col.state"}}</th>{{end}}<th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . .ShownWorksheets}}</tbody>
</table>
{{$finished := .ShownFinishedWorksheets}}
{{if $finished}}
<details class="finished-heading"{{if .Static}} open{{end}}><summary>{{T . "section.finished"}} <span class="count">({{len $finished}})</span></summary>
<table class="worksheets">
<thead><tr><th>{{T . "col.worksheet"}}</th><th>{{T . "col.updated"}}</th><th>{{T . "col.details"}}</th>{{if not .Static}}<th>{{T . "col.state"}}</th>{{end}}<th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . $finished}}</tbody>
</table>
</details>
{{end}}
</section>

{{if not .Static}}
{{if .Admin}}
{{$completed := .CompletedRequests}}{{if $completed}}
<section><h2>{{T . "updates.recent"}}</h2>
<table><thead><tr><th>{{T . "col.state"}}</th><th>{{T . "col.request"}}</th><th>{{T . "col.date"}}</th></tr></thead><tbody>
{{range $completed}}<tr><td><span class="status {{.Status}}">{{statusLabel $.Lang .Status}}</span></td>
<td><div class="title">{{$.RequestTitle .}}</div><div class="request-body">{{.Body}}</div>{{if .Note}}<p class="request-note">{{.Note}}</p>{{end}}</td>
<td class="date">{{.UpdatedAt.Format "2 Jan 2006, 15:04"}}</td></tr>{{end}}
</tbody></table></section>
{{end}}{{end}}

<div class="adminbar">
<span>{{if .Admin}}Admin controls are on.{{else}}Admin controls are off.{{end}}</span>
<form method="POST" action="/admin"><input type="hidden" name="admin" value="{{if .Admin}}off{{else}}on{{end}}"><button type="submit">{{if .Admin}}Turn off admin controls{{else}}Turn on admin controls{{end}}</button></form>
{{if .Admin}}<form method="POST" action="/work/rebuild"><button type="submit">{{T . "action.rebuild"}}</button></form>{{end}}
</div>
{{if and .Admin .Log}}<pre class="log">{{range .Log}}{{.}}
{{end}}</pre>{{end}}
{{end}}
{{end}}{{/* end FinishedView else (Home/category list) */}}
{{end}}{{/* end not Manage */}}
</div></div>
</div></body></html>
`))
