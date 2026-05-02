// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/steadytao/waystone/internal/model"
)

func TestExportImportArchiveRoundTrip(t *testing.T) {
	root := writeTestLedger(t)
	writer := Writer{Root: root}
	operation := model.Operation{
		ID:         NewOperationID("github import", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "github import",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root},
	}
	verification, err := (Reader{Root: root}).Verify()
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	operation.Changes = verification.Changes
	if err := writer.WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "ledger.tar.zst")
	if err := ExportArchive(root, archive); err != nil {
		t.Fatalf("ExportArchive returned error: %v", err)
	}
	imported := filepath.Join(t.TempDir(), ".waystone")
	if err := ImportArchive(archive, imported); err != nil {
		t.Fatalf("ImportArchive returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(imported, "projects", "github", "example", "project.json")); err != nil {
		t.Fatalf("imported project missing: %v", err)
	}
	if _, err := (Reader{Root: imported}).VerifyOperations(); err != nil {
		t.Fatalf("VerifyOperations after import returned error: %v", err)
	}
}

func TestImportArchiveRejectsUnsafePath(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bad.tar.zst")
	if err := writeTestArchive(archive, "../evil.json", []byte("{}\n")); err != nil {
		t.Fatalf("writeTestArchive returned error: %v", err)
	}

	err := ImportArchive(archive, filepath.Join(t.TempDir(), ".waystone"))
	if err == nil {
		t.Fatal("ImportArchive returned nil error for unsafe path")
	}
}

func writeTestArchive(path, name string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder, err := zstd.NewWriter(file)
	if err != nil {
		return err
	}
	defer encoder.Close()
	tw := tar.NewWriter(encoder)
	defer tw.Close()
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}); err != nil {
		return err
	}
	_, err = tw.Write(data)
	return err
}
