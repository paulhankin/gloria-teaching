// Package site renders the front page: the worksheet index plus the UI for
// worksheet requests (new worksheets and change requests).
package site

import (
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
		return "Manage"
	}
	if d.FinishedView {
		return "Finished worksheets"
	}
	if d.PublicView {
		return "Public worksheets"
	}
	if d.ActiveTagID == 0 {
		return "Worksheets"
	}
	return d.ActiveTagName
}

// SectionHeading titles the worksheet table below the page heading. On Home
// it reads "Available worksheets"; on a category view it repeats the category.
func (d Data) SectionHeading() string {
	if d.FinishedView {
		return "Finished worksheets"
	}
	if d.ActiveTagID == 0 {
		return "Available worksheets"
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
		rot := []string{"-2deg", "1.5deg", "-1deg", "2deg"}[i%4]
		lift := []string{"0", "-1.5px", "0.5px", "-1px"}[i%4]
		ch := string(r)
		fmt.Fprintf(&b, `<span class="lt c%d" style="--rot:%s;--lift:%s">`, cls, rot, lift)
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

func statusLabel(s store.Status) string {
	switch s {
	case store.StatusQueued:
		return "Queued"
	case store.StatusWorking:
		return "Work in progress"
	case store.StatusReview:
		return "Ready to publish"
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
		return "The worksheet is being updated now, in a disposable isolated workspace"
	case store.StatusReview:
		return "The update is finished and will be published automatically"
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
    font-size:min(50px,5vw); line-height:.9; margin:0; font-weight:900; letter-spacing:1.5px;
    text-transform:uppercase; white-space:nowrap;
    display:flex; align-items:center; justify-content:center;
    transform:rotate(-1deg); padding:0 4px 16px 4px; }
  .brand-gap { display:inline-block; width:.28em; }
  .brand-logo { height:1.05em; width:auto; flex-shrink:0;
    transform:rotate(2deg) translateY(.04em); }
  /* Colour-pencil 3D face: white letters with a coloured outline and a light,
     partial pencil-stroke wash on the face, over a solid 3D body whose depth
     is shaded with diagonal hatching in the same colour. */
  .brand-title .lt { position:relative; display:inline-block;
    transform:rotate(var(--rot,0deg)) translateY(var(--lift,0));
    -webkit-text-stroke:1.6px var(--edge); }
  .brand-title .lt .d { position:absolute; left:calc(var(--k)*1px); top:calc(var(--k)*1px);
    z-index:0; color:var(--edge); -webkit-text-stroke:0; }
  .brand-title .lt .hatch { position:absolute; left:calc(var(--k)*1px); top:calc(var(--k)*1px);
    z-index:1; -webkit-text-stroke:0; color:transparent;
    background:repeating-linear-gradient(45deg, var(--edge) 0 2px, transparent 2px 4.5px);
    -webkit-background-clip:text; background-clip:text; opacity:.75; }
  /* The face is white with a few quick pencil strokes that do not fully colour it in. */
  .brand-title .lt .face { position:relative; z-index:2; color:#fff; }
  .brand-title .lt .shade { position:absolute; left:0; top:0; z-index:3; pointer-events:none;
    -webkit-text-stroke:0; color:transparent;
    background:repeating-linear-gradient(58deg, var(--fill) 0 2.5px, transparent 2.5px 6.5px);
    -webkit-background-clip:text; background-clip:text; opacity:.4; }
  .brand-title .c1 { --fill:#f19a8f; --edge:#c0392b; }
  .brand-title .c2 { --fill:#f6c478; --edge:#b97f22; }
  .brand-title .c3 { --fill:#f2d96b; --edge:#9c8a1e; }
  .brand-title .c4 { --fill:#93d6a0; --edge:#3e8e52; }
  .brand-title .c5 { --fill:#8fc1ec; --edge:#2f6fae; }
  .brand-title .c6 { --fill:#c9a9ec; --edge:#7a4ea6; }
  .brand-title .c7 { --fill:#f4a9cf; --edge:#b9447f; }
  .brand-title .c8 { --fill:#8bd8d0; --edge:#2f8f86; }
  .brand-title .sp { display:inline-block; width:.55em; }
  .brand-title .wand { font-size:.5em; vertical-align:34px; transform:rotate(10deg); display:inline-block; margin-left:8px; -webkit-text-stroke:0; }
  /* Hand-drawn squiggle underline, like a crayon stroke gone wavy. */
  .brand-under { display:block; margin-top:0; transform:rotate(-.6deg); }
  .brand-sub { margin:6px 0 0; font-size:19px; font-weight:600;
    font-family:"Caveat","Patrick Hand","Segoe Print",cursive; }
  .brand-sub .doodle { font-size:16px; padding:0 7px; }
  .account { display:flex; align-items:center; justify-content:flex-end; gap:10px; color:var(--muted); font-size:13px; flex-wrap:wrap; }
  .account form { margin:0; }
  .account button, .account .impersonate-username { padding:5px 9px; }
  .account .impersonate-username { width:230px; border:1px solid #98a2b3; border-radius:3px; background:#fff; color:inherit; font:inherit; }
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
    .brand-under { width:80%; height:auto; }
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
  <td><div class="title">{{$r.Title}}</div>{{if not $static}}<span class="privacy">{{if $r.Private}}Private{{else}}Public{{end}} · owner: {{$r.Owner}}</span>{{end}}</td>
  <td class="date">{{if $r.Date}}{{$r.Date}}{{else}}—{{end}}</td>
  <td class="meta">{{$r.Meta}}</td>
  {{if not $static}}<td>
    {{$rs := $r.ChangeRequests}}
    {{if $rs}}{{range $rs}}<span class="status {{.Status}}">{{statusLabel .Status}}</span><br>{{end}}{{else}}<span class="meta">Current</span>{{end}}
  </td>{{end}}
  <td class="row-actions"><a class="pdf" href="{{$r.OutputPath}}/index.pdf{{if $r.Version}}?v={{$r.Version}}{{end}}">Worksheet PDF</a><a class="pdf" href="{{$r.OutputPath}}/solutions.pdf{{if $r.Version}}?v={{$r.Version}}{{end}}">Solutions PDF</a>{{if not $static}}<form method="POST" action="/worksheets/finished"><input type="hidden" name="worksheet" value="{{$r.Path}}"><input type="hidden" name="finished" value="{{if $r.Finished}}off{{else}}on{{end}}"><button class="finish" type="submit">{{if $r.Finished}}Move back to active{{else}}Mark as finished{{end}}</button></form>{{end}}</td>
</tr>
{{if not $static}}
<tr class="worksheet-request"><td colspan="5">
  {{if $r.Finished}}
  {{else}}
  {{$revisions := $r.Revisions}}
  {{if $revisions}}<details><summary>Revision history</summary>
    <ul class="revision-list">{{range $revisions}}<li>
      <div class="revision-info"><code>{{.Short}}</code> · {{.Date}} · {{.Subject}}</div>
      {{if .Current}}<span class="current">Current</span>{{else}}<form method="POST" action="/worksheets/revert"><input type="hidden" name="worksheet" value="{{$worksheet}}"><input type="hidden" name="commit" value="{{.Commit}}"><button type="submit">Revert to this version</button></form>{{end}}
    </li>{{end}}</ul>
  </details>{{end}}
  <details><summary>Worksheet settings</summary>
    <div class="sharing">
      <form class="visibility" method="POST" action="/worksheets/visibility">
        <input type="hidden" name="worksheet" value="{{$r.Path}}">
        <label for="visibility-{{$r.Name}}">Access</label>
        <select id="visibility-{{$r.Name}}" name="visibility">
          <option value="private"{{if $r.Private}} selected{{end}}>Private</option>
          <option value="public"{{if not $r.Private}} selected{{end}}>Public</option>
        </select>
        <button type="submit">Save</button>
      </form>
      {{if $r.Private}}
      <form class="share-form" method="POST" action="/worksheets/shares">
        <input type="hidden" name="worksheet" value="{{$r.Path}}">
        <input type="email" name="email" required placeholder="Existing user's email">
        <select name="permission" aria-label="Permission"><option value="view">Can view</option><option value="edit">Can edit</option></select>
        <button type="submit">Share</button>
      </form>
      {{if $r.Shares}}<ul class="share-list">{{range $r.Shares}}<li><span class="email">{{.Email}}</span><span>{{if eq .Permission "edit"}}Can edit{{else}}Can view{{end}}</span><form method="POST" action="/worksheets/shares/delete"><input type="hidden" name="worksheet" value="{{$worksheet}}"><input type="hidden" name="email" value="{{.Email}}"><button type="submit">Remove</button></form></li>{{end}}</ul>{{end}}
      {{else}}<span class="meta">Listed on <a href="/worksheets/{{$.User}}/index">your public page</a>.</span>{{end}}
    </div>
  </details>
  <details><summary>Request an update</summary>
    <form class="ask" method="POST" action="/requests">
      <input type="hidden" name="kind" value="change"><input type="hidden" name="worksheet" value="{{$r.Path}}">
      <textarea name="body" required placeholder="What should change?"></textarea>
      <button type="submit">Send request</button>
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
<h2 id="manage-heading">Manage categories</h2>
<p class="meta">Categories appear in the left-hand menu. Create a top-level category such as "First Grade", then add sub-categories like "Mathematics" beneath it. Tag worksheets to file them under a category.</p>

<h3>Add a category</h3>
<form class="ask tag-add" method="POST" action="/tags">
  <input type="text" name="name" required placeholder="Category name">
  <select name="parent" aria-label="Parent category">
    <option value="0">Top level (no parent)</option>
    {{range .Tags}}<option value="{{.ID}}">{{.Name}}</option>{{end}}
  </select>
  <button type="submit">Add category</button>
</form>

{{if .Tags}}
<h3>Your categories</h3>
<ul class="tag-manage-list">
{{range .Tags}}
  <li>
    <span class="tag-name">{{.Name}}</span>
    <form class="tag-rename" method="POST" action="/tags/rename"><input type="hidden" name="id" value="{{.ID}}"><input type="text" name="name" value="{{.Name}}" required><button type="submit">Rename</button></form>
    <form class="tag-delete" method="POST" action="/tags/delete" onsubmit="return confirm('Delete category &quot;{{.Name}}&quot; and its sub-categories? Worksheets keep their other categories.');"><input type="hidden" name="id" value="{{.ID}}"><button type="submit" class="no">Delete</button></form>
    {{if .Children}}<ul>{{range .Children}}
      <li><span class="tag-name sub">{{.Name}}</span>
        <form class="tag-rename" method="POST" action="/tags/rename"><input type="hidden" name="id" value="{{.ID}}"><input type="text" name="name" value="{{.Name}}" required><button type="submit">Rename</button></form>
        <form class="tag-delete" method="POST" action="/tags/delete"><input type="hidden" name="id" value="{{.ID}}"><button type="submit" class="no">Delete</button></form>
      </li>{{end}}</ul>{{end}}
  </li>
{{end}}
</ul>
{{end}}

<h3>Tag worksheets</h3>
{{if .Tags}}
<p class="meta">Tick the categories each worksheet belongs to.</p>
{{range .TagForms}}
<div class="tag-worksheet">
  <div class="title">{{.Title}}</div>
  <form method="POST" action="/worksheets/tags" class="tag-assign">
    <input type="hidden" name="worksheet" value="{{.Path}}">
    {{range .Checks}}<label class="tag-check"><input type="checkbox" name="tag" value="{{.ID}}"{{if .Checked}} checked{{end}}> {{.Name}}</label>{{end}}
    <button type="submit">Save</button>
  </form>
</div>
{{end}}
{{else}}<p class="meta">Create a category first, then tag your worksheets.</p>{{end}}
</section>
{{end}}
`

const requestActions = `
{{define "actions"}}
  <div class="actions">
    {{if eq .Status "failed"}}
    <form method="POST" action="/work/retry"><input type="hidden" name="id" value="{{.ID}}"><button type="submit">Retry</button></form>
    {{end}}
    <form method="POST" action="/work/reject"><input type="hidden" name="id" value="{{.ID}}"><button class="no" type="submit">Reject</button></form>
  </div>
  {{if and .Admin (eq .Status "failed")}}
  <form class="refine" method="POST" action="/work/refine">
    <input type="hidden" name="id" value="{{.ID}}"><input type="text" name="body" required placeholder="Describe the refinement">
    <button type="submit">Refine</button>
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
)

// brandUnderline is a hand-drawn crayon squiggle beneath the site name — a
// single uneven stroke whose width wobbles like a real pencil line.
const brandUnderlineSVG = `<svg class="brand-under" width="330" height="12" viewBox="0 0 330 12" fill="none"><path d="M4 8 C 40 3, 70 10, 105 6 S 170 2, 205 7 S 275 10, 326 5" stroke="#e2574c" stroke-width="4" stroke-linecap="round" opacity="0.55"/><path d="M6 9 C 42 4, 74 11, 108 7 S 172 3, 207 8 S 278 11, 324 6" stroke="#f0932b" stroke-width="2.4" stroke-linecap="round" opacity="0.7"/></svg>`

// brandLogoSVG is the site icon: two hands shaking beneath a floating heart,
// in the hand-coloured pencil palette of the title — the friendly, helping
// each other side of Teacher's Friend.
const brandLogoSVG = `<svg class="brand-logo" viewBox="0 0 120 120" fill="none" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M60 42 C 50 33, 43 27, 46.5 20 C 49.5 14.5, 56.5 14.5, 60 22 C 63.5 14.5, 70.5 14.5, 73.5 20 C 77 27, 70 33, 60 42 Z" fill="#f7b6c8" stroke="#e2574c" stroke-width="5"/><path d="M4 86 L 28 78 C 42 73, 52 68, 62 58 L 70 68 C 62 78, 50 84, 38 88 L 12 95 Z" fill="#a9cdf0" stroke="#6390c0" stroke-width="4"/><path d="M116 86 L 92 78 C 78 73, 68 68, 58 58 L 50 68 C 58 78, 70 84, 82 88 L 108 95 Z" fill="#f6b1a7" stroke="#e2574c" stroke-width="4"/><path d="M54 66 C 57 71, 63 73, 68 71" stroke="#6390c0" stroke-width="4.5" fill="none"/><path d="M66 66 C 63 71, 57 73, 52 71" stroke="#e2574c" stroke-width="4.5" fill="none"/></svg>`

// reqActions bundles a request with whether the viewer may use the
// admin-only actions (refine), so the actions template can decide.
type reqActions struct {
	store.Request
	Admin bool
}

var indexTmpl = template.Must(template.New("index").Funcs(template.FuncMap{
	"statusLabel":    statusLabel,
	"statusHelp":     statusHelp,
	"iconHome":       func() template.HTML { return template.HTML(iconHomeSVG) },
	"iconGlobe":      func() template.HTML { return template.HTML(iconGlobeSVG) },
	"iconCheck":      func() template.HTML { return template.HTML(iconCheckSVG) },
	"iconCog":        func() template.HTML { return template.HTML(iconCogSVG) },
	"colorTitle":     colorTitle,
	"brandUnderline": func() template.HTML { return template.HTML(brandUnderlineSVG) },
	"brandLogo":      func() template.HTML { return template.HTML(brandLogoSVG) },
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
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
{{if .Busy}}<meta http-equiv="refresh" content="10">{{end}}
<title>Learning material</title>
<style>{{.Fonts}}` + indexCSS + `</style>
</head>
<body>
{{if .Static}}
<div class="wrap">
<header><div class="brand-block"><h1 class="brand-title">{{colorTitle "Teacher's"}}<span class="brand-gap"></span>{{brandLogo}}<span class="brand-gap"></span>{{colorTitle "Friend"}}</h1>{{brandUnderline}}<p class="brand-sub"><span class="doodle">✏️</span>Unleash your creativity<span class="doodle">🖍️</span></p></div></header>
{{else}}
<div class="layout">
<nav class="sidebar" aria-label="Worksheet navigation">
  <div class="brand">{{if .User}}{{.User}}{{else}}Worksheets{{end}}</div>
  <a class="nav-item{{if and (not .Manage) (not .FinishedView) (not .PublicView) (eq .ActiveTagID 0)}} active{{end}}" href="/">{{iconHome}} Home</a>
  <a class="nav-item{{if .PublicView}} active{{end}}" href="/?public=1">{{iconGlobe}} Public Worksheets</a>
  <a class="nav-item{{if .FinishedView}} active{{end}}" href="/?finished=1">{{iconCheck}} Finished Worksheets</a>
  <a class="nav-item{{if .Manage}} active{{end}}" href="/?manage=1">{{iconCog}} Manage</a>
  {{if .Tags}}
  <div class="nav-section">My worksheets</div>
  {{range .Tags}}
  <a class="nav-item{{if eq $.ActiveTagID .ID}} active{{end}}" href="/?tag={{.ID}}">{{.Name}}</a>
  {{range .Children}}<a class="nav-item sub{{if eq $.ActiveTagID .ID}} active{{end}}" href="/?tag={{.ID}}">{{.Name}}</a>{{end}}
  {{end}}
  {{end}}
</nav>
<div class="main"><div class="wrap">
{{if .User}}<div class="topbar"><div class="account">{{if .Impersonating}}<span class="impersonation">Viewing as {{.User}}</span>{{end}}{{if .CanImpersonate}}<form method="POST" action="/account/impersonate"><input type="hidden" name="next" value="/"><input class="impersonate-username" type="text" name="username" list="impersonation-users" required placeholder="Username" aria-label="View site as user"><datalist id="impersonation-users">{{range .Users}}<option value="{{.}}">{{end}}</datalist><button type="submit">View as</button></form>{{if .Impersonating}}<form method="POST" action="/account/impersonate"><input type="hidden" name="next" value="/"><input type="hidden" name="username" value="{{.Actor}}"><button type="submit">Stop impersonating</button></form>{{end}}{{end}}<a href="/worksheets/{{.User}}/index">Public page</a><form method="POST" action="/account/sign-out"><button type="submit">Sign out</button></form></div></div>{{end}}
<header><div class="brand-block"><h1 class="brand-title">{{colorTitle "Teacher's"}}<span class="brand-gap"></span>{{brandLogo}}<span class="brand-gap"></span>{{colorTitle "Friend"}}</h1>{{brandUnderline}}<p class="brand-sub"><span class="doodle">✏️</span>Unleash your creativity<span class="doodle">🖍️</span></p></div></header>
{{end}}
{{if .Flash}}<div class="flash">{{.Flash}}</div>{{end}}

{{if and .Manage (not .Static)}}
{{template "manage" .}}
{{else}}

{{if and .FinishedView (not .Static)}}
<section aria-labelledby="worksheets-heading">
<h2 id="worksheets-heading">Finished worksheets <span class="count">({{len .FinishedWorksheets}})</span></h2>
{{if .FinishedWorksheets}}
<table class="worksheets">
<thead><tr><th>Worksheet</th><th>Updated</th><th>Details</th><th>State</th><th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . .FinishedWorksheets}}</tbody>
</table>
{{else}}<p class="meta">No finished worksheets yet. Use the "Mark as finished" button on a worksheet to file it here.</p>{{end}}
</section>
{{else if and .PublicView (not .Static)}}
<section aria-labelledby="worksheets-heading">
<h2 id="worksheets-heading">Public worksheets <span class="count">({{len .PublicWorksheets}})</span></h2>
<p class="meta">These worksheets are visible to anyone on <a href="/worksheets/{{.User}}/index">your public page</a>.</p>
{{if .PublicWorksheets}}
<table class="worksheets">
<thead><tr><th>Worksheet</th><th>Updated</th><th>Details</th><th>State</th><th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . .PublicWorksheets}}</tbody>
</table>
{{else}}<p class="meta">No public worksheets yet. Set a worksheet's visibility to "Public" under "Worksheet settings" to list it here.</p>{{end}}
</section>
{{else}}

{{if and (not .Static) (not .FinishedView)}}
<section class="new-request">
<h2>What can I do for you?</h2>
<form class="ask" method="POST" action="/requests">
  <input type="hidden" name="kind" value="new">
  <textarea name="body" required placeholder="Which subject, topic and level? What should the tasks look like?"></textarea>
  <button type="submit">Send request</button>
</form>
</section>
{{end}}

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
<thead><tr><th>Worksheet</th><th>Updated</th><th>Details</th>{{if not .Static}}<th>State</th>{{end}}<th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . .ShownWorksheets}}</tbody>
</table>
{{$finished := .ShownFinishedWorksheets}}
{{if $finished}}
<details class="finished-heading"{{if .Static}} open{{end}}><summary>Finished worksheets <span class="count">({{len $finished}})</span></summary>
<table class="worksheets">
<thead><tr><th>Worksheet</th><th>Updated</th><th>Details</th>{{if not .Static}}<th>State</th>{{end}}<th></th></tr></thead>
<tbody>{{template "worksheet-rows" rowSet . $finished}}</tbody>
</table>
</details>
{{end}}
</section>

{{if not .Static}}
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
{{end}}{{/* end FinishedView else (Home/category list) */}}
{{end}}{{/* end not Manage */}}
</div></div>
</div></body></html>
`))
