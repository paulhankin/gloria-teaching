// Package sheet ist das Mini-Framework fuer Arbeitsblaetter.
//
// Ein Arbeitsblatt liegt unter generate/<fach>/<name>/ und meldet sich per
// Register() an. Der Generator (cmd/generate) baut daraus
// output/<fach>/<name>/index.html (+ index.pdf).
package sheet

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed assets/fonts.css assets/rough.js
var assets embed.FS

// Asset liefert eine eingebettete Datei aus internal/sheet/assets.
func Asset(name string) string {
	b, err := assets.ReadFile("assets/" + name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Worksheet beschreibt ein Arbeitsblatt fuer Registry und Startseite.
type Worksheet struct {
	Fach     string // z.B. "mathe"  -> Verzeichnisname
	Name     string // z.B. "venn_diagramme" -> Verzeichnisname
	Titel    string // Anzeigename, z.B. "Venn-Diagramme"
	Meta     string // Kurzbeschreibung fuer die Startseite
	Build    func() *Doc
	Portrait bool // PDF im Hochformat drucken (Standard: quer)
}

// Path ist der Ausgabepfad relativ zum output-Verzeichnis.
func (w Worksheet) Path() string { return w.Fach + "/" + w.Name }

var registry []Worksheet

// Register meldet ein Arbeitsblatt an (aus init() aufrufen).
func Register(w Worksheet) {
	if w.Fach == "" || w.Name == "" || w.Build == nil {
		panic("sheet.Register: Fach, Name und Build sind Pflicht")
	}
	registry = append(registry, w)
}

// All liefert alle angemeldeten Arbeitsblaetter, sortiert nach Pfad.
func All() []Worksheet {
	out := append([]Worksheet(nil), registry...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path() < out[j].Path() })
	return out
}

// Doc ist der gebaute Inhalt eines Arbeitsblattes.
type Doc struct {
	Titel   string         // <title>
	CSS     string         // zusaetzliches CSS (BaseCSS ist immer dabei)
	Body    string         // HTML-Rumpf (meist mit Page(...) gebaut)
	Data    map[string]any // wird als `const NAME = <json>;` ins JS gelegt
	Scripts []string       // JS-Quelltexte (laufen nach den Daten)
	Rough   bool           // rough.js einbetten
}

// AddScript haengt JS-Quelltext an.
func (d *Doc) AddScript(src string) { d.Scripts = append(d.Scripts, src) }

// Set legt einen JS-Konstantennamen mit JSON-Wert an.
func (d *Doc) Set(name string, v any) {
	if d.Data == nil {
		d.Data = map[string]any{}
	}
	d.Data[name] = v
}

// HTML rendert das fertige, selbstenthaltende Dokument.
func (d *Doc) HTML() string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"de\">\n<head>\n<meta charset=\"utf-8\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n<style>\n%s\n%s\n</style>\n</head>\n<body>\n",
		d.Titel, Asset("fonts.css"), BaseCSS+d.CSS)
	b.WriteString(d.Body)
	if d.Rough {
		fmt.Fprintf(&b, "<script>\n%s\n</script>\n", Asset("rough.js"))
	}
	if len(d.Data) > 0 || len(d.Scripts) > 0 {
		b.WriteString("<script>\n")
		names := make([]string, 0, len(d.Data))
		for k := range d.Data {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&b, "const %s = %s;\n", n, JSON(d.Data[n]))
		}
		for _, s := range d.Scripts {
			b.WriteString(s)
			b.WriteString("\n")
		}
		b.WriteString("</script>\n")
	}
	b.WriteString("</body>\n</html>\n")
	return b.String()
}
