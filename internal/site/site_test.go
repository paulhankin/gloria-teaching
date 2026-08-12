package site

import (
	"strings"
	"testing"

	"learningmaterial/internal/store"
)

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
