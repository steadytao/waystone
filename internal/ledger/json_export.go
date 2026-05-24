// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type JSONExport struct {
	Version    string           `json:"version"`
	ExportedAt time.Time        `json:"exported_at"`
	Files      []JSONExportFile `json:"files"`
}

type JSONExportFile struct {
	Path   string          `json:"path"`
	SHA256 string          `json:"sha256"`
	JSON   json.RawMessage `json:"json"`
}

func ExportJSON(root, out string, compact bool) error {
	if root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}
	if _, err := (Reader{Root: root}).VerifyOperations(); err != nil {
		return err
	}
	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := rejectSymlinkEntry(path, entry); err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(files)

	bundle := JSONExport{
		Version:    "waystone.export.v1",
		ExportedAt: time.Now().UTC(),
		Files:      make([]JSONExportFile, 0, len(files)),
	}
	for _, path := range files {
		data, err := readFileNoSymlink(path)
		if err != nil {
			return err
		}
		if !json.Valid(data) {
			return fmt.Errorf("invalid JSON: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		bundle.Files = append(bundle.Files, JSONExportFile{
			Path:   filepath.ToSlash(relative),
			SHA256: hex.EncodeToString(sum[:]),
			JSON:   json.RawMessage(data),
		})
	}

	file, err := createUserSelectedFile(out)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(bundle)
}
