package sheet

import (
	"strings"
	"testing"
)

func TestDocRendersWorksheetAndSolutionsSeparately(t *testing.T) {
	d := &Doc{
		Title:     "Testblatt",
		Body:      `<div class="page">Aufgaben</div>`,
		Solutions: `<div class="page solution">Lösungen</div>`,
	}

	worksheet := d.HTML()
	solutions := d.SolutionsHTML()
	if !strings.Contains(worksheet, "Aufgaben") || strings.Contains(worksheet, "Lösungen") {
		t.Fatalf("worksheet HTML contains the wrong pages: %q", worksheet)
	}
	if !strings.Contains(solutions, "Lösungen") || strings.Contains(solutions, "Aufgaben") {
		t.Fatalf("solutions HTML contains the wrong pages: %q", solutions)
	}
	if !strings.Contains(solutions, "<title>Testblatt – Lösungen</title>") {
		t.Fatal("solutions HTML has no distinct title")
	}
}
