// Package sheet is the mini framework for worksheets.
//
// A worksheet's code lives in the local per-user Git repository
// /users/<username>/<name>/ and is exposed to the Go build below generate/.
// It registers itself via Register(). The generator (cmd/generate) turns it into
// output/<username>/<name>/index.{html,pdf} and solutions.{html,pdf}.
package sheet

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed assets/fonts.css assets/rough.js assets/logo.png
var assets embed.FS

// Asset returns an embedded file from internal/sheet/assets.
func Asset(name string) string {
	b, err := assets.ReadFile("assets/" + name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// AssetBytes returns an embedded binary asset (images) from internal/sheet/assets.
func AssetBytes(name string) []byte {
	b, err := assets.ReadFile("assets/" + name)
	if err != nil {
		panic(err)
	}
	return b
}

// Worksheet describes a worksheet for the registry and the index page.
type Worksheet struct {
	Username string // source owner, e.g. "gloriahankin" -> generate/gloriahankin/<name>
	Subject  string // output category, e.g. "math" -> output/math/<name>
	Name     string // e.g. "venn_diagrams" -> directory name
	Title    string // display name, e.g. "Venn-Diagramme"
	Date     string // last content update, e.g. "12 Aug 2026"
	Meta     string // short factual description for the index page
	Build    func() *Doc
	Portrait bool // print the PDF in portrait (default: landscape)
}

// Path is the output path relative to the output directory.
func (w Worksheet) Path() string { return w.Subject + "/" + w.Name }

// OutputPath is the generated-file location below the output root.
func (w Worksheet) OutputPath() string { return w.Username + "/" + w.Name }

var registry []Worksheet

// Register adds a worksheet to the registry (call from init()).
func Register(w Worksheet) {
	if w.Username == "" || w.Subject == "" || w.Name == "" || w.Build == nil {
		panic("sheet.Register: Username, Subject, Name and Build are required")
	}
	registry = append(registry, w)
}

// All returns every registered worksheet, sorted by path.
func All() []Worksheet {
	out := append([]Worksheet(nil), registry...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path() < out[j].Path() })
	return out
}

// Doc is the built content of a worksheet.
type Doc struct {
	Title     string         // <title> of the worksheet document
	CSS       string         // extra CSS (BaseCSS is always included)
	Body      string         // worksheet HTML body (usually built with Page(...))
	Solutions string         // solution HTML body (usually built with SolutionPage(...))
	Data      map[string]any // emitted as `const NAME = <json>;` in the script
	Scripts   []string       // JS sources (run after the data)
	Rough     bool           // embed rough.js
}

// AddScript appends JS source code.
func (d *Doc) AddScript(src string) { d.Scripts = append(d.Scripts, src) }

// Set defines a JS constant with a JSON value.
func (d *Doc) Set(name string, v any) {
	if d.Data == nil {
		d.Data = map[string]any{}
	}
	d.Data[name] = v
}

// HTML renders the finished, self-contained worksheet document.
func (d *Doc) HTML() string { return d.html(d.Title, d.Body) }

// SolutionsHTML renders the solutions as a separate, self-contained document.
func (d *Doc) SolutionsHTML() string {
	return d.html(d.Title+" – Lösungen", d.Solutions)
}

func (d *Doc) html(title, body string) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"de\">\n<head>\n<meta charset=\"utf-8\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n<style>\n%s\n%s\n</style>\n</head>\n<body>\n",
		title, Asset("fonts.css"), BaseCSS+d.CSS)
	b.WriteString(body)
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
