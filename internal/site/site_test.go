package site

import (
	"strings"
	"testing"

	"learningmaterial/internal/store"
)

func TestWorksheetIndexLinksWorksheetAndSolutionsPDFs(t *testing.T) {
	html := Index(Data{Static: true, Worksheets: []Worksheet{{
		Username: "gloriahankin", Subject: "math", Name: "fractions", Title: "Brüche", Date: "12 Aug 2026", Meta: "4. Klasse · Lösungen", Version: "abc123",
	}}})
	for _, want := range []string{
		"<table", "Brüche", "12 Aug 2026", "Worksheet PDF", "Solutions PDF",
		"gloriahankin/fractions/index.pdf?v=abc123", "gloriahankin/fractions/solutions.pdf?v=abc123",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("index is missing %q", want)
		}
	}
	if strings.Contains(html, "index.html") || strings.Contains(html, "View in browser") {
		t.Fatal("worksheet index exposes HTML output")
	}
}

func TestUpdateRequestFormSpansWorksheetTable(t *testing.T) {
	html := Index(Data{Worksheets: []Worksheet{{Subject: "math", Name: "fractions", Title: "Brüche"}}})
	for _, want := range []string{
		`<tr class="worksheet-request"><td colspan="5">`,
		`.worksheet-request form.ask { max-width:none; }`,
		`<summary>Request an update</summary>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("full-width update request form is missing %q", want)
		}
	}
}

func TestRequestFormsUseSignedInIdentity(t *testing.T) {
	html := Index(Data{
		User:       "teacher@example.com",
		Worksheets: []Worksheet{{Subject: "math", Name: "fractions", Title: "Brüche"}},
		Requests: []store.Request{{
			ID: 7, Kind: store.KindChange, Worksheet: "math/fractions", Author: "teacher", Requester: "teacher@example.com",
			Status: store.StatusQueued, Body: "Add another task",
		}},
	})
	if strings.Contains(html, `name="author"`) || strings.Contains(html, "Your name (optional)") {
		t.Fatal("request forms still ask the signed-in user for a name")
	}
	if !strings.Contains(html, "Request #7 · teacher") {
		t.Fatal("request metadata does not use the signed-in identity")
	}
}

func TestActiveWorkStateIsProminentWithoutAdmin(t *testing.T) {
	html := Index(Data{
		Worksheets: []Worksheet{{Subject: "math", Name: "fractions", Title: "Brüche"}},
		Requests: []store.Request{{
			ID: 7, Kind: store.KindChange, Worksheet: "math/fractions",
			Status: store.StatusWorking, Body: "Add another task",
		}},
	})
	for _, want := range []string{"Updates in progress", "Work in progress", "The worksheet is being updated now, in a disposable isolated workspace", "Add another task"} {
		if !strings.Contains(html, want) {
			t.Fatalf("active work display is missing %q", want)
		}
	}
	// The workspace wording must stay truthful about Stage 1: filesystem and
	// process isolation only, never a network boundary.
	for _, banned := range []string{"fully isolated", "secure network sandbox"} {
		if strings.Contains(html, banned) {
			t.Fatalf("status wording overpromises isolation: %q", banned)
		}
	}
	// Reject and retry are available to any signed-in user (they only discard
	// or re-queue a request). Publishing (approve) and driving the agent
	// (refine) stay behind admin mode.
	if strings.Contains(html, "/work/approve") || strings.Contains(html, "/work/refine") {
		t.Fatal("admin-only actions are visible while admin controls are off")
	}
}

func TestSignedInUserHasSignOutControl(t *testing.T) {
	html := Index(Data{User: "paulhankin"})
	for _, want := range []string{"paulhankin", `/account/sign-out`, "Sign out"} {
		if !strings.Contains(html, want) {
			t.Fatalf("signed-in header is missing %q", want)
		}
	}
}

func TestAdminCanSwitchUsersAndStopImpersonating(t *testing.T) {
	html := Index(Data{
		User:           "teacher",
		Actor:          "admin",
		CanImpersonate: true,
		Users:          []string{"admin", "teacher"},
	})
	for _, want := range []string{
		"Viewing as teacher", `/account/impersonate`, "admin",
		"teacher", "View as", "Stop impersonating",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("impersonation controls are missing %q", want)
		}
	}
}

func TestCompletedRequests(t *testing.T) {
	d := Data{Requests: []store.Request{
		{ID: 4, Status: store.StatusDone},
		{ID: 3, Status: store.StatusReview},
		{ID: 2, Status: store.StatusRejected},
	}}
	got := d.CompletedRequests()
	if len(got) != 2 || got[0].ID != 4 || got[1].ID != 2 {
		t.Fatalf("CompletedRequests() = %#v", got)
	}
}

func TestCompletedItemHasNoActions(t *testing.T) {
	html := Index(Data{
		Admin:    true,
		Requests: []store.Request{{ID: 4, Status: store.StatusDone, Body: "published work"}},
	})
	if !strings.Contains(html, "Recent completed work") || !strings.Contains(html, "published work") {
		t.Fatal("completed work is missing")
	}
	if strings.Contains(html, `/work/reject`) || strings.Contains(html, `/work/approve`) {
		t.Fatal("completed work has decision actions")
	}
}

func TestRevisionHistoryOffersRevertWithoutApproval(t *testing.T) {
	html := Index(Data{
		Admin: true,
		Worksheets: []Worksheet{{
			Subject: "math", Name: "fractions", Title: "Brüche",
		}},
		Revisions: map[string][]Revision{"math/fractions": {
			{Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Short: "aaaaaaaaaaaa", Subject: "Latest", Date: "16 Aug 2026, 10:00", Current: true},
			{Commit: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Short: "bbbbbbbbbbbb", Subject: "Earlier", Date: "15 Aug 2026, 10:00"},
		}},
		Requests: []store.Request{{ID: 8, Status: store.StatusReview, Body: "finished work"}},
	})
	for _, want := range []string{"Revision history", "Current", "Revert to this version", `/worksheets/revert`, "will be published automatically"} {
		if !strings.Contains(html, want) {
			t.Fatalf("revision UI is missing %q", want)
		}
	}
	if strings.Contains(html, "Approve &amp; publish") || strings.Contains(html, `/work/approve`) {
		t.Fatal("finished work still asks for approval")
	}
}

func TestPrivateWorksheetShowsSharingControls(t *testing.T) {
	html := Index(Data{User: "owner@example.com", Worksheets: []Worksheet{{
		Subject: "math", Name: "fractions", Title: "Brüche", Owner: "owner@example.com",
		Visibility: store.VisibilityPrivate,
		Shares:     []store.WorksheetShare{{Email: "friend@example.com", Permission: store.PermissionEdit}},
	}}})
	for _, want := range []string{
		"Private · owner: owner@example.com", "Worksheet settings", `/worksheets/visibility`,
		`/worksheets/shares`, `/worksheets/finished`, "Mark as finished", "friend@example.com", "Can edit", "Remove",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("sharing controls are missing %q", want)
		}
	}
}

func TestPublicWorksheetDoesNotShowShareForm(t *testing.T) {
	html := Index(Data{Worksheets: []Worksheet{{
		Subject: "math", Name: "fractions", Title: "Brüche", Owner: "owner@example.com",
		Visibility: store.VisibilityPublic,
	}}})
	if !strings.Contains(html, "Public · owner: owner@example.com") || !strings.Contains(html, "your public page") {
		t.Fatal("public worksheet state is missing")
	}
	if strings.Contains(html, `class="share-form"`) {
		t.Fatal("public worksheet exposes private sharing form")
	}
}

func TestFinishedWorksheetsMoveToTheirOwnSection(t *testing.T) {
	html := Index(Data{User: "owner@example.com", Worksheets: []Worksheet{
		{Subject: "math", Name: "fractions", Title: "Brüche", Owner: "owner@example.com"},
		{Subject: "math", Name: "venn", Title: "Venn-Diagramme", Owner: "owner@example.com", Finished: true},
	}})
	if !strings.Contains(html, "Available worksheets <span class=\"count\">(1)</span>") ||
		!strings.Contains(html, "Finished worksheets <span class=\"count\">(1)</span>") {
		t.Fatal("worksheet counts are not split into active and finished")
	}
	if !strings.Contains(html, "Move back to active") {
		t.Fatal("finished worksheet cannot be moved back")
	}
	if !strings.Contains(html, "Mark as finished") {
		t.Fatal("active worksheet cannot be marked finished")
	}
	finished := html[strings.Index(html, "Finished worksheets"):]
	if strings.Contains(finished, "Mark as finished") {
		t.Fatal("finished worksheet can be marked finished again")
	}
	if strings.Contains(finished, "Request an update") || strings.Contains(finished, "Revision history") {
		t.Fatal("finished worksheets still offer updates or revisions")
	}
}

func TestPublicIndexLinksPublishedWorksheets(t *testing.T) {
	html := PublicIndex(PublicData{
		OwnerUsername:  "teacher",
		ViewerUsername: "friend",
		Worksheets: []Worksheet{{
			Username: "teacher", Subject: "math", Name: "fractions", Title: "Brüche", Date: "16 Aug 2026",
			Meta: "4. Klasse", Version: "abc123",
		}},
	})
	for _, want := range []string{
		"teacher's worksheets", "friend", "/worksheets/teacher/sheet/fractions",
		"/teacher/fractions/index.pdf?v=abc123", "/teacher/fractions/solutions.pdf?v=abc123",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("public index is missing %q", want)
		}
	}
}

func TestPublicIndexEmptyState(t *testing.T) {
	html := PublicIndex(PublicData{OwnerUsername: "teacher", ViewerUsername: "friend"})
	if !strings.Contains(html, "no published worksheets") {
		t.Fatal("public index has no empty state")
	}
}
