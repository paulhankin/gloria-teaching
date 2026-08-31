// Command serve exposes the output/ directory behind username/email/password accounts
// and renders the interactive front page (worksheet index + request forms).
//
//	go run ./cmd/serve -addr :8000 -dir output -db data/requests.db
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"learningmaterial/internal/account"
	"learningmaterial/internal/pipeline"
	"learningmaterial/internal/sheet"
	"learningmaterial/internal/site"
	"learningmaterial/internal/store"
)

var db *store.DB
var pipe *pipeline.Pipeline
var manifestPath string
var outputRoot string

const existingWorksheetOwner = "g.n.hankin@gmail.com"

func siteBaseURL() string {
	if value := strings.TrimRight(os.Getenv("SITE_BASE_URL"), "/"); value != "" {
		return value
	}
	return "https://gloria-teaching.exe.xyz"
}

func adminEmails() []string {
	return []string{"paul.hankin@pobox.com", "g.n.hankin@gmail.com"}
}

func allowedEmails() []string {
	value := os.Getenv("SITE_ALLOWED_EMAILS")
	if value == "" {
		value = "paul.hankin@pobox.com,g.n.hankin@gmail.com"
	}
	seen := make(map[string]bool)
	var emails []string
	for _, email := range append(strings.Split(value, ","), adminEmails()...) {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" && !seen[email] {
			seen[email] = true
			emails = append(emails, email)
		}
	}
	return emails
}

// index renders the front page with the request UI.
func index(w http.ResponseWriter, r *http.Request) {
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	worksheets, err := ownedWorksheets(manifest, account.Email(r))
	if err != nil {
		http.Error(w, "worksheet access: "+err.Error(), http.StatusInternalServerError)
		return
	}
	revisions := make(map[string][]site.Revision, len(worksheets))
	for _, ws := range worksheets {
		history, err := pipe.Revisions(ws.Path())
		if err != nil {
			http.Error(w, "worksheet revisions: "+err.Error(), http.StatusInternalServerError)
			return
		}
		revisions[ws.Path()] = history
	}
	d := site.Data{
		Worksheets:     worksheets,
		Revisions:      revisions,
		Admin:          db.AdminMode(),
		Flash:          flash(w, r),
		User:           account.Username(r),
		Actor:          account.ActorUsername(r),
		CanImpersonate: account.IsAdmin(r),
		WorksheetTags:  make(map[string]map[int64]bool),
		Lang:           langOf(r),
		RequestPath:    r.URL.RequestURI(),
	}
	if me, err := db.UserByEmail(account.Email(r)); err == nil {
		d.HasAvatar = len(me.Avatar) > 0
		if d.HasAvatar {
			sum := sha256.Sum256(me.Avatar)
			d.AvatarVersion = fmt.Sprintf("%x", sum[:6])
		}
	}
	owner := account.Email(r)
	tags, err := db.Tags(owner)
	if err != nil {
		http.Error(w, "tags: "+err.Error(), http.StatusInternalServerError)
		return
	}
	d.Tags = tags
	for _, ws := range worksheets {
		ids, err := db.WorksheetTagIDs(ws.Path())
		if err != nil {
			http.Error(w, "worksheet tags: "+err.Error(), http.StatusInternalServerError)
			return
		}
		d.WorksheetTags[ws.Path()] = ids
	}
	// Navigation state: ?manage=1 for the manage view, ?finished=1 for the
	// finished list, ?tag=<id> for a category.
	q := r.URL.Query()
	switch {
	case q.Get("manage") == "1":
		d.Manage = true
	case q.Get("finished") == "1":
		d.FinishedView = true
	case q.Get("public") == "1":
		d.PublicView = true
	default:
		if tagID, err := strconv.ParseInt(q.Get("tag"), 10, 64); err == nil && tagID != 0 {
			if name, ok := findTag(tags, tagID); ok {
				d.ActiveTagID = tagID
				d.ActiveTagName = name
			}
		}
	}
	if d.CanImpersonate {
		users, err := db.Users()
		if err != nil {
			http.Error(w, "users: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, user := range users {
			d.Users = append(d.Users, user.Username)
		}
	}
	rs, err := db.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.Requests = rs
	if d.Admin {
		d.Log = pipe.Log()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	io.WriteString(w, site.Index(d))
}

// flash reads and clears the one-shot confirmation message.
// langOf returns the UI language cookie ("de" or anything else = English).
func langOf(r *http.Request) string {
	if c, err := r.Cookie("lm_lang"); err == nil && c.Value == "de" {
		return "de"
	}
	return "en"
}

// setLanguage stores the chosen UI language and returns to the page the user
// was on.
func setLanguage(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	if lang != "de" {
		lang = "en"
	}
	http.SetCookie(w, &http.Cookie{Name: "lm_lang", Value: lang, Path: "/", MaxAge: 365 * 24 * 3600})
	next := r.FormValue("next")
	if next == "" || !strings.HasPrefix(next, "/") {
		next = "/"
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func flash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("lm_flash")
	if err != nil || c.Value == "" {
		return ""
	}
	http.SetCookie(w, &http.Cookie{Name: "lm_flash", Value: "", Path: "/", MaxAge: -1})
	switch c.Value {
	case "new":
		return "Thanks! Your worksheet request has been saved."
	case "change":
		return "Thanks! Your change request has been saved."
	case "deleted":
		return "Request deleted."
	case "approve":
		return "Approved: merging, pushing and rebuilding."
	case "reject":
		return "Rejected: the change was thrown away."
	case "refine":
		return "Refinement sent to the agent."
	case "retry":
		return "Restarted."
	case "rebuild":
		return "Rebuilding the site."
	case "sharing":
		return "Sharing settings saved."
	case "reverted":
		return "Earlier revision restored and published."
	case "finished":
		return "Worksheet moved to the finished list."
	case "unfinished":
		return "Worksheet moved back to the active list."
	case "tag":
		return "Categories saved."
	}
	return ""
}

func setFlash(w http.ResponseWriter, v string) {
	http.SetCookie(w, &http.Cookie{Name: "lm_flash", Value: v, Path: "/", MaxAge: 300})
}

// postRequest stores a new or change request.
func postRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	kind := store.Kind(r.FormValue("kind"))
	worksheet := r.FormValue("worksheet")
	if kind == store.KindNew {
		worksheet = ""
	} else if kind != store.KindChange {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	} else if known, err := knownOwnedWorksheet(worksheet, account.Email(r)); err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	} else if !known {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "empty request", http.StatusBadRequest)
		return
	}
	req := store.Request{
		Kind:      kind,
		Worksheet: worksheet,
		Author:    account.Username(r),
		Requester: account.Email(r),
		Body:      body,
	}
	if _, err := db.Add(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pipe.Kick()
	setFlash(w, string(kind))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// deleteRequest removes a request (admin mode only).
func deleteRequest(w http.ResponseWriter, r *http.Request) {
	id, ok := adminPost(w, r)
	if !ok {
		return
	}
	if err := db.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setFlash(w, "deleted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// adminPost validates a POST with an id field in admin mode.
func adminPost(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return 0, false
	}
	if !db.AdminMode() {
		http.Error(w, "admin mode is off", http.StatusForbidden)
		return 0, false
	}
	r.ParseForm()
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// work handles the decisions on a work item: approve, reject, refine, retry.
func work(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimPrefix(r.URL.Path, "/work/")
	if action == "rebuild" {
		if r.Method != http.MethodPost || !db.AdminMode() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		pipe.Rebuild()
		setFlash(w, "rebuild")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	// Reject and retry are safe for any signed-in user: they only discard or
	// re-queue a request. Approve and refine publish content or drive the
	// agent, so those stay behind admin mode.
	switch action {
	case "approve", "refine":
		if !db.AdminMode() {
			http.Error(w, "admin mode is off", http.StatusForbidden)
			return
		}
	case "reject", "retry":
		// any signed-in user
	default:
		http.NotFound(w, r)
		return
	}
	switch action {
	case "approve":
		err = pipe.Approve(id)
	case "reject":
		err = pipe.Reject(id)
	case "retry":
		err = pipe.Retry(id)
	case "refine":
		body := strings.TrimSpace(r.FormValue("body"))
		if body == "" {
			http.Error(w, "empty refinement", http.StatusBadRequest)
			return
		}
		err = pipe.Refine(id, body)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	setFlash(w, action)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// serveAvatar serves a user's profile picture.
func serveAvatar(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Path, "/avatar/")
	user, err := db.UserByUsername(username)
	if err != nil || len(user.Avatar) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(user.Avatar))
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write(user.Avatar)
}

// setAvatar stores the signed-in user's uploaded profile picture.
func setAvatar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(4 << 20); err != nil { // 4 MiB cap
		http.Error(w, "upload too large", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "choose an image", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 4<<20))
	if err != nil || len(data) == 0 {
		http.Error(w, "could not read image", http.StatusBadRequest)
		return
	}
	if ct := http.DetectContentType(data); !strings.HasPrefix(ct, "image/") {
		http.Error(w, "that is not an image", http.StatusBadRequest)
		return
	}
	user, err := db.UserByEmail(account.Email(r))
	if err != nil {
		http.Error(w, "account not found", http.StatusBadRequest)
		return
	}
	if err := db.SetUserAvatar(user.ID, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	next := r.FormValue("next")
	if next == "" || !strings.HasPrefix(next, "/") {
		next = "/"
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// serveLogo serves the embedded Teacher's Friend logo (the handshake image).
func serveLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(sheet.AssetBytes("logo.png"))
}

// postAdmin toggles admin mode.
func postAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	if err := db.SetAdminMode(r.FormValue("admin") == "on"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func knownOwnedWorksheet(path, owner string) (bool, error) {
	worksheets, err := site.ReadManifest(manifestPath)
	if err != nil {
		return false, err
	}
	known := false
	for _, ws := range worksheets {
		if ws.Path() == path {
			known = true
			break
		}
	}
	if !known {
		return false, nil
	}
	access, err := db.WorksheetByPath(path)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(access.OwnerEmail, owner), nil
}

func ownedWorksheets(manifest []site.Worksheet, owner string) ([]site.Worksheet, error) {
	paths := make([]string, 0, len(manifest))
	byPath := make(map[string]site.Worksheet, len(manifest))
	for _, ws := range manifest {
		paths = append(paths, ws.Path())
		byPath[ws.Path()] = ws
	}
	if err := db.EnsureWorksheets(paths, existingWorksheetOwner); err != nil {
		return nil, err
	}
	access, err := db.WorksheetsOwnedBy(owner)
	if err != nil {
		return nil, err
	}
	out := make([]site.Worksheet, 0, len(access))
	for _, a := range access {
		ws, ok := byPath[a.Worksheet]
		if !ok {
			continue
		}
		ws.Owner = a.OwnerEmail
		ws.Visibility = a.Visibility
		ws.Finished = a.Finished
		ws.Shares = a.Shares
		out = append(out, ws)
	}
	return out, nil
}

func worksheetRevert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	worksheet := r.FormValue("worksheet")
	known, err := knownOwnedWorksheet(worksheet, account.Email(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !known {
		http.Error(w, "worksheet not found", http.StatusNotFound)
		return
	}
	if err := pipe.Revert(worksheet, r.FormValue("commit")); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	setFlash(w, "reverted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func worksheetVisibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	visibility := store.Visibility(r.FormValue("visibility"))
	if err := db.SetWorksheetVisibility(r.FormValue("worksheet"), account.Email(r), visibility); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "sharing")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// worksheetFinished moves a worksheet between the active and finished lists.
func worksheetFinished(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	finished := r.FormValue("finished") == "on"
	if err := db.SetWorksheetFinished(r.FormValue("worksheet"), account.Email(r), finished); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if finished {
		setFlash(w, "finished")
	} else {
		setFlash(w, "unfinished")
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// findTag returns the name of the tag with id within the owner's tree.
func findTag(tags []store.Tag, id int64) (string, bool) {
	for _, t := range tags {
		if t.ID == id {
			return t.Name, true
		}
		if name, ok := findTag(t.Children, id); ok {
			return name, true
		}
	}
	return "", false
}

// tagCreate adds a navigation category (optionally a sub-category).
func tagCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	parent, _ := strconv.ParseInt(r.FormValue("parent"), 10, 64)
	if _, err := db.CreateTag(account.Email(r), r.FormValue("name"), parent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "tag")
	http.Redirect(w, r, "/?manage=1", http.StatusSeeOther)
}

// tagRename renames a category.
func tagRename(w http.ResponseWriter, r *http.Request) {
	tagMutate(w, r, func(id int64, owner string) error {
		return db.RenameTag(id, owner, r.FormValue("name"))
	})
}

// tagDelete removes a category and its sub-categories.
func tagDelete(w http.ResponseWriter, r *http.Request) {
	tagMutate(w, r, func(id int64, owner string) error {
		return db.DeleteTag(id, owner)
	})
}

// tagMutate runs an owner-scoped tag operation identified by an id field.
func tagMutate(w http.ResponseWriter, r *http.Request, fn func(id int64, owner string) error) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := fn(id, account.Email(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "tag")
	http.Redirect(w, r, "/?manage=1", http.StatusSeeOther)
}

// worksheetTags saves the category checkboxes of one worksheet.
func worksheetTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	var ids []int64
	for _, v := range r.Form["tag"] {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if err := db.SetWorksheetTags(r.FormValue("worksheet"), account.Email(r), ids); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "tag")
	http.Redirect(w, r, "/?manage=1", http.StatusSeeOther)
}

func worksheetShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	user, err := db.UserByEmail(email)
	if err != nil || !user.Verified {
		http.Error(w, "share recipient must be an existing verified user", http.StatusBadRequest)
		return
	}
	if err := db.SetWorksheetShare(
		r.FormValue("worksheet"), account.Email(r), user.Email, store.Permission(r.FormValue("permission")),
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "sharing")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func worksheetShareDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	if err := db.DeleteWorksheetShare(
		r.FormValue("worksheet"), account.Email(r), strings.TrimSpace(r.FormValue("email")),
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "sharing")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func publicWorksheets(manifest []site.Worksheet, username, ownerEmail string) ([]site.Worksheet, error) {
	owned, err := ownedWorksheets(manifest, ownerEmail)
	if err != nil {
		return nil, err
	}
	out := make([]site.Worksheet, 0, len(owned))
	for _, ws := range owned {
		if ws.Username == username && ws.Visibility == store.VisibilityPublic {
			out = append(out, ws)
		}
	}
	return out, nil
}

func worksheetForUser(manifest []site.Worksheet, username, name string) (site.Worksheet, bool) {
	for _, ws := range manifest {
		if ws.Username == username && ws.Name == name {
			return ws, true
		}
	}
	return site.Worksheet{}, false
}

func worksheetRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[0] == "worksheets" && parts[2] == "index" {
		publicWorksheetIndex(w, r, parts[1])
		return
	}
	if len(parts) == 4 && parts[0] == "worksheets" && parts[2] == "sheet" {
		serveWorksheet(w, r, parts[1], parts[3])
		return
	}
	http.NotFound(w, r)
}

func publicWorksheetIndex(w http.ResponseWriter, r *http.Request, username string) {
	owner, err := db.UserByUsername(username)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	worksheets, err := publicWorksheets(manifest, owner.Username, owner.Email)
	if err != nil {
		http.Error(w, "worksheet access: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	io.WriteString(w, site.PublicIndex(site.PublicData{
		OwnerUsername: owner.Username, ViewerUsername: account.Username(r), Worksheets: worksheets,
	}))
}

func serveWorksheet(w http.ResponseWriter, r *http.Request, username, name string) {
	owner, err := db.UserByUsername(username)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	ws, ok := worksheetForUser(manifest, owner.Username, name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	access, err := db.WorksheetByPath(ws.Path())
	if err != nil || !strings.EqualFold(access.OwnerEmail, owner.Email) {
		http.NotFound(w, r)
		return
	}
	allowed, err := db.CanViewWorksheet(ws.Path(), account.Email(r))
	if err != nil {
		http.Error(w, "worksheet access: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(outputRoot, ws.OutputPath(), "index.html"))
}

func canServeWorksheet(path, viewerEmail string) bool {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return true
	}
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		return false
	}
	outputPath := parts[0] + "/" + parts[1]
	for _, ws := range manifest {
		if ws.OutputPath() != outputPath {
			continue
		}
		allowed, err := db.CanViewWorksheet(ws.Path(), viewerEmail)
		return err == nil && allowed
	}
	return false
}

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	dir := flag.String("dir", "output", "directory to serve")
	dbPath := flag.String("db", "data/requests.db", "SQLite database with the requests")
	repo := flag.String("repo", ".", "git checkout the pipeline works on")
	workRoot := flag.String("work", "data/worktrees", "directory for the per-item git worktrees")
	usersRoot := flag.String("users", "/users", "directory containing local per-user worksheet repositories")
	preview := flag.String("preview", "data/preview", "directory for the per-item preview builds")
	sandbox := flag.Bool("sandbox", false, "run agent work in per-request sandboxes with standalone repository clones")
	sandboxRoot := flag.String("sandboxes", "data/sandboxes", "parent directory for the per-request sandboxes")
	push := flag.Bool("push", true, "push to origin after an approved change")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	var err error
	db, err = store.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	secret := []byte(os.Getenv("SITE_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Fatal(err)
		}
	}
	abs := func(p string) string {
		a, err := filepath.Abs(p)
		if err != nil {
			log.Fatal(err)
		}
		return a
	}
	outputDir := abs(*dir)
	outputRoot = outputDir
	manifestPath = filepath.Join(outputDir, site.ManifestName)
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		log.Fatalf("worksheet catalog: %v", err)
	}
	paths := make([]string, 0, len(manifest))
	for _, ws := range manifest {
		paths = append(paths, ws.Path())
	}
	if err := db.EnsureWorksheets(paths, existingWorksheetOwner); err != nil {
		log.Fatalf("worksheet access: %v", err)
	}
	pipe = pipeline.New(db, pipeline.Config{
		Repo:          abs(*repo),
		WorksheetRoot: abs(*usersRoot),
		WorkRoot:      abs(*workRoot),
		PreviewRoot:   abs(*preview),
		OutputDir:     outputDir,
		Push:          *push,
		Sandbox:       *sandbox,
		SandboxRoot:   abs(*sandboxRoot),
	})
	pipe.Start()

	// SIGTERM (systemd stop) and SIGINT (Ctrl-C) gracefully stop every
	// sandboxed Shelley server before the process exits, so no sandboxed
	// agent outlives the service.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigs
		log.Printf("received %s; stopping sandboxed shelley servers", sig)
		pipe.Shutdown()
		os.Exit(0)
	}()

	files := http.FileServer(http.Dir(*dir))
	previews := http.StripPrefix("/preview/", http.FileServer(http.Dir(abs(*preview))))
	mux := http.NewServeMux()
	mux.HandleFunc("/requests", postRequest)
	mux.HandleFunc("/requests/delete", deleteRequest)
	mux.HandleFunc("/worksheets/visibility", worksheetVisibility)
	mux.HandleFunc("/worksheets/finished", worksheetFinished)
	mux.HandleFunc("/worksheets/tags", worksheetTags)
	mux.HandleFunc("/worksheets/revert", worksheetRevert)
	mux.HandleFunc("/tags", tagCreate)
	mux.HandleFunc("/tags/rename", tagRename)
	mux.HandleFunc("/tags/delete", tagDelete)
	mux.HandleFunc("/worksheets/shares", worksheetShare)
	mux.HandleFunc("/worksheets/shares/delete", worksheetShareDelete)
	mux.HandleFunc("/worksheets/", worksheetRoutes)
	mux.HandleFunc("/work/", work)
	mux.HandleFunc("/admin", postAdmin)
	mux.HandleFunc("/language", setLanguage)
	mux.HandleFunc("/account/avatar", setAvatar)
	mux.HandleFunc("/avatar/", serveAvatar)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || r.URL.Path == "/index.html":
			index(w, r)
		case strings.HasPrefix(r.URL.Path, "/preview/"):
			if !db.AdminMode() {
				http.Error(w, "admin mode is off", http.StatusForbidden)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			previews.ServeHTTP(w, r)
		default:
			if !canServeWorksheet(r.URL.Path, account.Email(r)) {
				http.NotFound(w, r)
				return
			}
			files.ServeHTTP(w, r)
		}
	})

	accounts := account.New(db, secret, allowedEmails(), adminEmails(), account.GatewayMailer{}, siteBaseURL())
	public := http.NewServeMux()
	accounts.Register(public)
	// The logo is public: it decorates the sign-in page too.
	public.HandleFunc("/logo.png", serveLogo)
	public.Handle("/", accounts.RequireAccess(mux))

	log.Printf("serving %s on %s (db %s)", *dir, *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, public))
}
