package site

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ManifestName)
	want := []Worksheet{{Subject: "math", Name: "fractions", Title: "Brüche", Date: "12 Aug 2026", Meta: "Level 4"}}
	if err := WriteManifest(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadManifest() = %#v, want %#v", got, want)
	}

	replacement := []Worksheet{{Subject: "language", Name: "verbs", Title: "Verben"}}
	if err := WriteManifest(path, replacement); err != nil {
		t.Fatal(err)
	}
	got, err = ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, replacement) {
		t.Fatalf("ReadManifest() after replacement = %#v, want %#v", got, replacement)
	}
}
