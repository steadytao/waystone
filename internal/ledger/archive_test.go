// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestExportArchiveIncludesSignedManifest(t *testing.T) {
	root := writeTestLedger(t)
	if _, err := CreateDefaultIdentity(root, "test"); err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}
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
	if err := (Writer{Root: root}).WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "ledger")
	if err := ExportArchive(root, archive); err != nil {
		t.Fatalf("ExportArchive returned error: %v", err)
	}

	manifest, err := ReadArchiveManifest(archive)
	if err != nil {
		t.Fatalf("ReadArchiveManifest returned error: %v", err)
	}
	if manifest.Version != "waystone.archive.v1" {
		t.Fatalf("manifest version = %q, want waystone.archive.v1", manifest.Version)
	}
	if manifest.Signature == nil || manifest.Signature.Value == "" {
		t.Fatal("manifest is not signed")
	}
	if len(manifest.Files) == 0 {
		t.Fatal("manifest has no file refs")
	}
	for _, file := range manifest.Files {
		if file.Path == archiveManifestName {
			t.Fatal("manifest lists itself")
		}
		if strings.HasSuffix(file.Path, ".key") {
			t.Fatalf("manifest includes private key path %q", file.Path)
		}
	}
	if _, err := VerifyArchiveManifest(archive); err != nil {
		t.Fatalf("VerifyArchiveManifest returned error: %v", err)
	}
}

func TestImportArchiveRejectsManifestHashMismatch(t *testing.T) {
	root := writeTestLedger(t)
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
	if err := (Writer{Root: root}).WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "ledger")
	if err := ExportArchive(root, archive); err != nil {
		t.Fatalf("ExportArchive returned error: %v", err)
	}
	tampered := filepath.Join(t.TempDir(), "tampered")
	if err := rewriteArchiveEntry(archive, tampered, "ledger.json", []byte("{}\n")); err != nil {
		t.Fatalf("rewriteArchiveEntry returned error: %v", err)
	}

	if err := ImportArchive(tampered, filepath.Join(t.TempDir(), ".waystone")); err == nil {
		t.Fatal("ImportArchive returned nil error for archive hash mismatch")
	}
}

func TestVerifyArchiveManifestRejectsDuplicateFileRefs(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "duplicate-ref.tar.zst")
	manifestData, err := canonicalJSON(model.ArchiveManifest{
		Version:   "waystone.archive.v1",
		Format:    "tar+zstd",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Files: []model.ArchiveFileRef{
			{Path: "ledger.json", SHA256: strings.Repeat("0", 64), Size: 3},
			{Path: "ledger.json", SHA256: strings.Repeat("0", 64), Size: 3},
		},
	})
	if err != nil {
		t.Fatalf("canonicalJSON returned error: %v", err)
	}
	if err := writeTestArchiveEntries(archive, map[string][]byte{
		"ledger.json":       []byte("{}\n"),
		archiveManifestName: manifestData,
	}); err != nil {
		t.Fatalf("writeTestArchiveEntries returned error: %v", err)
	}

	_, err = VerifyArchiveManifest(archive)
	if err == nil || !strings.Contains(err.Error(), "duplicate path") {
		t.Fatalf("VerifyArchiveManifest error = %v, want duplicate path error", err)
	}
}

func TestImportArchiveRejectsUnsafePaths(t *testing.T) {
	tests := []string{
		"..",
		"../evil.json",
		"./evil.json",
		"safe//evil.json",
		"safe/../evil.json",
		"/evil.json",
		`..\evil.json`,
		`C:/evil.json`,
		`C:\evil.json`,
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "bad.tar.zst")
			if err := writeTestArchive(archive, name, []byte("{}\n")); err != nil {
				t.Fatalf("writeTestArchive returned error: %v", err)
			}

			err := ImportArchive(archive, filepath.Join(t.TempDir(), ".waystone"))
			if err == nil {
				t.Fatal("ImportArchive returned nil error for unsafe path")
			}
		})
	}
}

func TestExportArchiveRejectsSymlinkedLedgerFile(t *testing.T) {
	root := writeTestLedger(t)
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"outside":true}`+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	link := filepath.Join(root, "objects", "github", "example", "project", "issues", "symlink.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	err := ExportArchive(root, filepath.Join(t.TempDir(), "ledger"))
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("ExportArchive error = %v, want symlink rejection", err)
	}
}

func TestExportSourceArchiveRejectsSymlinkedObjectParent(t *testing.T) {
	root := writeTestLedger(t)
	link := filepath.Join(root, "objects", "github", "example", "project", "issues")
	if err := os.RemoveAll(link); err != nil {
		t.Fatalf("RemoveAll returned error: %v", err)
	}
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	err := ExportSourceArchive(root, "github:example/project", filepath.Join(t.TempDir(), "source-ledger"))
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("ExportSourceArchive error = %v, want symlink rejection", err)
	}
}

func TestCopyFileRejectsSymlinkedDestination(t *testing.T) {
	from := filepath.Join(t.TempDir(), "from.json")
	if err := os.WriteFile(from, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	to := filepath.Join(t.TempDir(), "to.json")
	if err := os.Symlink(outside, to); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	err := copyFile(from, to)
	if err == nil {
		t.Fatal("copyFile returned nil error for symlinked destination")
	}
	data, readErr := os.ReadFile(outside)
	if readErr != nil {
		t.Fatalf("ReadFile returned error: %v", readErr)
	}
	if string(data) != "outside" {
		t.Fatalf("symlink target content = %q, want unchanged", string(data))
	}
}

func TestOpenFileNoSymlinkRejectsDirectory(t *testing.T) {
	_, err := openFileNoSymlink(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("openFileNoSymlink error = %v, want regular file rejection", err)
	}
}

func TestWalkArchiveRejectsTooManyEntries(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "too-many.tar.zst")
	if err := writeTestArchiveEntries(archive, map[string][]byte{
		"one.json": []byte("{}\n"),
		"two.json": []byte("{}\n"),
	}); err != nil {
		t.Fatalf("writeTestArchiveEntries returned error: %v", err)
	}

	err := walkArchiveWithLimits(archive, archiveLimits{MaxEntryBytes: 100, MaxEntries: 1, MaxTotalBytes: 100}, func(*tar.Header, []byte) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "too many entries") {
		t.Fatalf("walkArchiveWithLimits error = %v, want too many entries", err)
	}
}

func TestWalkArchiveRejectsExcessiveTotalSize(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "too-large-total.tar.zst")
	if err := writeTestArchiveEntries(archive, map[string][]byte{
		"one.json": []byte("1234"),
		"two.json": []byte("5678"),
	}); err != nil {
		t.Fatalf("writeTestArchiveEntries returned error: %v", err)
	}

	err := walkArchiveWithLimits(archive, archiveLimits{MaxEntryBytes: 100, MaxEntries: 10, MaxTotalBytes: 5}, func(*tar.Header, []byte) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "total size") {
		t.Fatalf("walkArchiveWithLimits error = %v, want total size limit", err)
	}
}

func TestExportArchiveExcludesPrivateIdentityKey(t *testing.T) {
	root := writeTestLedger(t)
	if _, err := CreateDefaultIdentity(root, "test"); err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "ledger")
	if err := ExportArchive(root, archive); err != nil {
		t.Fatalf("ExportArchive returned error: %v", err)
	}
	names, err := archiveEntryNames(archive)
	if err != nil {
		t.Fatalf("archiveEntryNames returned error: %v", err)
	}
	for _, name := range names {
		if name == "identities/default.key" {
			t.Fatal("archive includes private identity key")
		}
	}
}

func TestExportSourceArchiveIncludesAuditObjects(t *testing.T) {
	root := writeTestLedger(t)
	audit := model.GitHubAudit{
		ID:          "github-audit-20260101t000000.000000000z",
		Source:      model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Repository:  model.GitHubAuditRepository{FullName: "example/project", URL: "https://github.com/example/project", DefaultBranch: "main"},
		Portable:    []string{"issues"},
	}
	if source, err := (Reader{Root: root}).Source(audit.Source); err == nil {
		audit.Source.Objects = source.Objects
		audit.Source.Operations = source.Operations
	}
	if err := (Writer{Root: root}).WriteGitHubAudit(audit); err != nil {
		t.Fatalf("WriteGitHubAudit returned error: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "source-ledger")
	if err := ExportSourceArchive(root, "github:example/project", archive); err != nil {
		t.Fatalf("ExportSourceArchive returned error: %v", err)
	}
	imported := filepath.Join(t.TempDir(), ".waystone")
	if err := ImportArchive(archive, imported); err != nil {
		t.Fatalf("ImportArchive returned error: %v", err)
	}
	audits, err := (Reader{Root: imported}).Audits()
	if err != nil {
		t.Fatalf("Audits returned error: %v", err)
	}
	if len(audits) != 1 || audits[0].ID != audit.ID {
		t.Fatalf("audits = %#v, want source audit object", audits)
	}
}

func TestSafeRootedPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	tests := []string{
		"..",
		"../evil.json",
		"./evil.json",
		"safe//evil.json",
		"safe/../evil.json",
		"/evil.json",
		`..\evil.json`,
		`C:/evil.json`,
		`C:\evil.json`,
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := SafeRootedPath(root, name); err == nil {
				t.Fatal("SafeRootedPath returned nil error")
			}
		})
	}
}

func archiveEntryNames(path string) ([]string, error) {
	file, err := openUserSelectedFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder, err := zstd.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	tr := tar.NewReader(decoder)
	var names []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return names, nil
		}
		if err != nil {
			return nil, err
		}
		names = append(names, header.Name)
	}
}

func writeTestArchive(path, name string, data []byte) error {
	return writeTestArchiveEntries(path, map[string][]byte{name: data})
}

func writeTestArchiveEntries(path string, entries map[string][]byte) error {
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
	for name, data := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func rewriteArchiveEntry(from, to, name string, replacement []byte) error {
	input, err := openUserSelectedFile(from)
	if err != nil {
		return err
	}
	defer input.Close()
	decoder, err := zstd.NewReader(input)
	if err != nil {
		return err
	}
	defer decoder.Close()
	output, err := os.Create(to)
	if err != nil {
		return err
	}
	defer output.Close()
	encoder, err := zstd.NewWriter(output)
	if err != nil {
		return err
	}
	defer encoder.Close()
	tr := tar.NewReader(decoder)
	tw := tar.NewWriter(encoder)
	defer tw.Close()
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		if header.Name == name {
			data = replacement
			header.Size = int64(len(data))
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}
}
