package site

import (
	"strings"
	"testing"

	"learningmaterial/internal/store"
)

func TestWorksheetIndexLinksWorksheetAndSolutionsPDFs(t *testing.T) {
	html := Index(Data{Static: true, Worksheets: []Worksheet{{
		Subject: "math", Name: "fractions", Title: "Brüche", Date: "12 Aug 2026", Meta: "4. Klasse · Lösungen", Version: "abc123",
	}}})
	for _, want := range []string{
		"<table", "Brüche", "12 Aug 2026", "Worksheet PDF", "Solutions PDF",
		"math/fractions/index.pdf?v=abc123", "math/fractions/solutions.pdf?v=abc123",
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

func TestActiveWorkStateIsProminentWithoutAdmin(t *testing.T) {
	html := Index(Data{
		Worksheets: []Worksheet{{Subject: "math", Name: "fractions", Title: "Brüche"}},
		Requests: []store.Request{{
			ID: 7, Kind: store.KindChange, Worksheet: "math/fractions",
			Status: store.StatusWorking, Body: "Add another task",
		}},
	})
	for _, want := range []string{"Updates in progress", "Work in progress", "The worksheet is being updated now", "Add another task"} {
		if !strings.Contains(html, want) {
			t.Fatalf("active work display is missing %q", want)
		}
	}
	if strings.Contains(html, "/work/approve") || strings.Contains(html, "/work/reject") {
		t.Fatal("admin actions are visible while admin controls are off")
	}
}

func TestSignedInUserHasSignOutControl(t *testing.T) {
	html := Index(Data{User: "paul.hankin@pobox.com"})
	for _, want := range []string{"paul.hankin@pobox.com", `/account/sign-out`, "Sign out"} {
		if !strings.Contains(html, want) {
			t.Fatalf("signed-in header is missing %q", want)
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
