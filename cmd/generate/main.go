// Command generate builds every worksheet into output/<username>/<name>/.
//
//	go run ./cmd/generate            # HTML + PDF for all worksheets
//	go run ./cmd/generate -pdf=false # HTML only (fast)
//	go run ./cmd/generate venn       # only matching worksheets (substring)
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"learningmaterial/internal/pdf"
	"learningmaterial/internal/sheet"
	"learningmaterial/internal/site"
	"learningmaterial/internal/worksheetrepo"
)

func main() {
	out := flag.String("out", "output", "output directory")
	usersRoot := flag.String("users", "", "directory containing per-user worksheet repositories")
	makePDF := flag.Bool("pdf", true, "also produce PDFs")
	wait := flag.Duration("wait", 4*time.Second, "time to wait for JS rendering before the PDF export")
	flag.Parse()

	changed, err := worksheetrepo.Prepare(*usersRoot)
	if err != nil {
		log.Fatal(err)
	}
	if changed {
		cmd := exec.Command("go", append([]string{"run", "./cmd/generate"}, os.Args[1:]...)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	filters := flag.Args()
	target, err := filepath.Abs(*out)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		log.Fatal(err)
	}
	buildDir, err := os.MkdirTemp(filepath.Dir(target), ".output-build-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(buildDir)

	var built []sheet.Worksheet
	versions := make(map[string]string)

	for _, w := range sheet.All() {
		if !matches(w, filters) {
			continue
		}
		dir := filepath.Join(buildDir, w.OutputPath())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatal(err)
		}
		doc := w.Build()
		documents := []struct {
			name string
			html string
		}{
			{name: "index", html: doc.HTML()},
			{name: "solutions", html: doc.SolutionsHTML()},
		}
		for _, document := range documents {
			htmlPath := filepath.Join(dir, document.name+".html")
			if err := os.WriteFile(htmlPath, []byte(document.html), 0o644); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%-34s %4d KB\n", htmlPath, len(document.html)/1024)

			if *makePDF {
				pdfPath := filepath.Join(dir, document.name+".pdf")
				err := pdf.Render(htmlPath, pdfPath, pdf.Options{
					Landscape: !w.Portrait, Wait: *wait,
				})
				if err != nil {
					log.Fatalf("%s: %v", pdfPath, err)
				}
				st, _ := os.Stat(pdfPath)
				fmt.Printf("%-34s %4d KB\n", pdfPath, st.Size()/1024)
			}
		}
		built = append(built, w)
		version, err := worksheetVersion(buildDir, w.OutputPath())
		if err != nil {
			log.Fatal(err)
		}
		versions[w.Path()] = version
	}

	if len(built) == 0 {
		log.Fatal("no worksheet matches the filter")
	}

	// The index page and manifest always cover every known worksheet. The
	// server reads the manifest at request time, so worksheet additions and
	// metadata changes do not require rebuilding the server binary.
	all := sheet.All()
	for _, w := range all {
		if versions[w.Path()] != "" {
			continue
		}
		version, err := worksheetVersion(target, w.OutputPath())
		if err != nil && !os.IsNotExist(err) {
			log.Fatal(err)
		}
		versions[w.Path()] = version
	}
	worksheets := siteWorksheets(all, versions)
	idx := filepath.Join(buildDir, "index.html")
	page := site.Index(site.Data{Worksheets: worksheets, Static: true})
	if err := os.WriteFile(idx, []byte(page), 0o644); err != nil {
		log.Fatal(err)
	}
	if err := site.WriteManifest(filepath.Join(buildDir, site.ManifestName), worksheets); err != nil {
		log.Fatal(err)
	}
	if err := publishOutput(buildDir, target); err != nil {
		log.Fatal(err)
	}
	if err := pruneOutput(target, worksheets); err != nil {
		log.Fatal(err)
	}
	fmt.Println(filepath.Join(target, "index.html"))
}

func pruneOutput(target string, worksheets []site.Worksheet) error {
	valid := make(map[string]bool, len(worksheets))
	for _, ws := range worksheets {
		valid[filepath.Clean(ws.OutputPath())] = true
	}
	users, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, user := range users {
		if !user.IsDir() {
			continue
		}
		userDir := filepath.Join(target, user.Name())
		sheets, err := os.ReadDir(userDir)
		if err != nil {
			return err
		}
		for _, worksheet := range sheets {
			if !worksheet.IsDir() {
				continue
			}
			rel := filepath.Join(user.Name(), worksheet.Name())
			if valid[rel] {
				continue
			}
			dir := filepath.Join(userDir, worksheet.Name())
			if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
				if err := os.RemoveAll(dir); err != nil {
					return err
				}
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		remaining, err := os.ReadDir(userDir)
		if err != nil {
			return err
		}
		if len(remaining) == 0 {
			if err := os.Remove(userDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func publishOutput(buildDir, target string) error {
	var files []string
	if err := filepath.WalkDir(buildDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			rel, err := filepath.Rel(buildDir, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	}); err != nil {
		return err
	}
	// Publish the two catalogs last. New worksheet links cannot become visible
	// before all of their generated files are in place; the live-server manifest
	// is the final commit point.
	rank := func(path string) int {
		switch path {
		case "index.html":
			return 1
		case site.ManifestName:
			return 2
		default:
			return 0
		}
	}
	sort.Slice(files, func(i, j int) bool {
		ri, rj := rank(files[i]), rank(files[j])
		if ri != rj {
			return ri < rj
		}
		return files[i] < files[j]
	})
	for _, rel := range files {
		src := filepath.Join(buildDir, rel)
		dst := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("publish %s: %w", rel, err)
		}
	}
	return nil
}

func siteWorksheets(in []sheet.Worksheet, versions map[string]string) []site.Worksheet {
	out := make([]site.Worksheet, 0, len(in))
	for _, w := range in {
		out = append(out, site.Worksheet{
			Username: w.Username,
			Subject:  w.Subject,
			Name:     w.Name,
			Title:    w.Title,
			Date:     w.Date,
			Meta:     w.Meta,
			Version:  versions[w.Path()],
		})
	}
	return out
}

// worksheetVersion hashes the generated files that browsers download. PDFs
// are preferred; HTML is the fallback for fast -pdf=false builds.
func worksheetVersion(root, worksheetPath string) (string, error) {
	for _, ext := range []string{"pdf", "html"} {
		h := sha256.New()
		complete := true
		for _, name := range []string{"index", "solutions"} {
			path := filepath.Join(root, worksheetPath, name+"."+ext)
			f, err := os.Open(path)
			if err != nil {
				if os.IsNotExist(err) {
					complete = false
					break
				}
				return "", err
			}
			_, copyErr := io.Copy(h, f)
			closeErr := f.Close()
			if copyErr != nil {
				return "", copyErr
			}
			if closeErr != nil {
				return "", closeErr
			}
			h.Write([]byte{0})
		}
		if complete {
			return hex.EncodeToString(h.Sum(nil))[:12], nil
		}
	}
	return "", os.ErrNotExist
}

func matches(w sheet.Worksheet, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if strings.Contains(w.Path(), f) {
			return true
		}
	}
	return false
}
