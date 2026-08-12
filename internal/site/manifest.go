package site

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ManifestName = "worksheets.json"

// ReadManifest loads the generated worksheet catalog used by the live server.
func ReadManifest(path string) ([]Worksheet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var worksheets []Worksheet
	if err := json.Unmarshal(data, &worksheets); err != nil {
		return nil, fmt.Errorf("decode worksheet manifest: %w", err)
	}
	return worksheets, nil
}

// WriteManifest atomically publishes a worksheet catalog. Readers therefore
// see either the old complete catalog or the new complete catalog.
func WriteManifest(path string, worksheets []Worksheet) error {
	data, err := json.MarshalIndent(worksheets, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".worksheets-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
