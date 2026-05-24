// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

func TestDiffGitHubImportReportsCreatedAndUnchanged(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	imported := model.GitHubImport{
		Project: model.Project{ID: "github:repo:1", Name: "example/project", URL: "https://github.com/example/project"},
		Source:  model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		Issues:  []model.Issue{{ID: "github:issue:1", Number: 1, Title: "issue"}},
	}

	diff, err := writer.DiffGitHubImport(imported)
	if err != nil {
		t.Fatalf("DiffGitHubImport returned error: %v", err)
	}
	if diff.Created == 0 {
		t.Fatalf("created = %d, want > 0", diff.Created)
	}

	if err := writer.WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}
	diff, err = writer.DiffGitHubImport(imported)
	if err != nil {
		t.Fatalf("DiffGitHubImport returned error after write: %v", err)
	}
	if diff.Unchanged == 0 {
		t.Fatalf("unchanged = %d, want > 0", diff.Unchanged)
	}
	if len(diff.Changes) != diff.Unchanged {
		t.Fatalf("changes = %d, want one verified action per unchanged object %d", len(diff.Changes), diff.Unchanged)
	}
	for _, change := range diff.Changes {
		if change.Type != "verified" {
			t.Fatalf("change type = %q, want verified", change.Type)
		}
	}
	if diff.Created != 0 || diff.Updated != 0 {
		t.Fatalf("created=%d updated=%d, want 0/0", diff.Created, diff.Updated)
	}
}

func TestWriteAndReadOperation(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	operation := model.Operation{
		ID:         NewOperationID("github import", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "github import",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root, Created: 1},
	}

	if err := writer.WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}
	got, err := (Reader{Root: root}).Operation(operation.ID)
	if err != nil {
		t.Fatalf("Operation returned error: %v", err)
	}
	if got.Command != operation.Command {
		t.Fatalf("command = %q, want %q", got.Command, operation.Command)
	}

	filenameID := namedFile(operation.ID)
	filenameID = filenameID[:len(filenameID)-len(".json")]
	got, err = (Reader{Root: root}).Operation(filenameID)
	if err != nil {
		t.Fatalf("Operation by filename ID returned error: %v", err)
	}
	if got.ID != operation.ID {
		t.Fatalf("operation ID = %q, want %q", got.ID, operation.ID)
	}
}

func TestOperationsSortDeterministicallyForEqualTimestamps(t *testing.T) {
	root := t.TempDir()
	startedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	operations := []model.Operation{
		{
			ID:         "operation-b",
			Command:    "second",
			StartedAt:  startedAt,
			FinishedAt: startedAt,
			Output:     model.OperationOutput{Ledger: root},
		},
		{
			ID:         "operation-a",
			Command:    "first",
			StartedAt:  startedAt,
			FinishedAt: startedAt,
			Output:     model.OperationOutput{Ledger: root},
		},
	}
	for _, operation := range operations {
		if err := (Writer{Root: root}).WriteOperation(operation); err != nil {
			t.Fatalf("WriteOperation(%s) returned error: %v", operation.ID, err)
		}
	}

	got, err := (Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("operations = %d, want 2", len(got))
	}
	if got[0].ID != "operation-a" || got[1].ID != "operation-b" {
		t.Fatalf("operation order = %q, %q; want operation-a, operation-b", got[0].ID, got[1].ID)
	}
}

func TestLocalActorOmitsMachineIdentityByDefault(t *testing.T) {
	actor := LocalActor("Tao", "tao@example.com", false)
	if actor.Source != "local" {
		t.Fatalf("source = %q, want local", actor.Source)
	}
	if actor.User != "" || actor.Hostname != "" {
		t.Fatalf("actor leaked local machine identity: %#v", actor)
	}
	if actor.GitUserName != "Tao" || actor.GitUserEmail != "tao@example.com" {
		t.Fatalf("git identity not preserved: %#v", actor)
	}
}

func TestVerifyLedger(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	imported := model.GitHubImport{
		Project: model.Project{ID: "github:repo:1", Name: "example/project", URL: "https://github.com/example/project"},
		Source:  model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		Issues:  []model.Issue{{ID: "github:issue:1", Number: 1, Title: "issue"}},
	}
	if err := writer.WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}

	verification, err := (Reader{Root: root}).Verify()
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if verification.Files == 0 {
		t.Fatal("verification file count was zero")
	}
	if verification.Checksum == "" {
		t.Fatal("verification checksum was empty")
	}
	if len(verification.Changes) != verification.Files {
		t.Fatalf("verified changes = %d, want %d", len(verification.Changes), verification.Files)
	}
	for _, change := range verification.Changes {
		if change.Type != "verified" {
			t.Fatalf("change type = %q, want verified", change.Type)
		}
	}
}

func TestVerifyRejectsSymlinkedJSON(t *testing.T) {
	root := writeTestLedger(t)
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	link := filepath.Join(root, "objects", "github", "example", "project", "issues", "symlink.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	_, err := (Reader{Root: root}).Verify()
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Verify error = %v, want symlink rejection", err)
	}
}

func TestVerifyOperationsRejectsMissingOperationHash(t *testing.T) {
	root := t.TempDir()
	operation := model.Operation{
		ID:         NewOperationID("legacy command", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "legacy command",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root},
	}
	if err := os.MkdirAll(filepath.Join(root, "operations"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := writeJSON(filepath.Join(root, operationRelativePath(operation.ID)), operation); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}

	_, err := (Reader{Root: root}).VerifyOperations()
	if err == nil || !strings.Contains(err.Error(), "has no operation_hash") {
		t.Fatalf("VerifyOperations error = %v, want missing hash error", err)
	}
}

func TestVerifyOperationsDetectsRecordedFileTampering(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	imported := model.GitHubImport{
		Project: model.Project{ID: "github:repo:1", Name: "example/project", URL: "https://github.com/example/project"},
		Source:  model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		Issues:  []model.Issue{{ID: "github:issue:1", Number: 1, Title: "issue"}},
	}
	diff, err := writer.DiffGitHubImport(imported)
	if err != nil {
		t.Fatalf("DiffGitHubImport returned error: %v", err)
	}
	if err := writer.WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}
	operation := model.Operation{
		ID:         NewOperationID("github import", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "github import",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root, Created: diff.Created, Unchanged: diff.Unchanged},
		Changes:    diff.Changes,
	}
	if err := writer.WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}
	if _, err := (Reader{Root: root}).VerifyOperations(); err != nil {
		t.Fatalf("VerifyOperations returned error before tamper: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "objects", "github", "example", "project", "issues", "000001.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err = (Reader{Root: root}).VerifyOperations()
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("VerifyOperations error = %v, want hash mismatch", err)
	}
}

func TestVerifyOperationsChecksSourceManifestObjectRefsWithoutOperations(t *testing.T) {
	root := writeTestLedger(t)
	if err := os.WriteFile(filepath.Join(root, "objects", "github", "example", "project", "issues", "000001.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := (Reader{Root: root}).VerifyOperations()
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("VerifyOperations error = %v, want source object hash mismatch", err)
	}
}

func TestVerifyOperationsRejectsMissingSourceManifestObject(t *testing.T) {
	root := writeTestLedger(t)
	if err := os.Remove(filepath.Join(root, "objects", "github", "example", "project", "issues", "000001.json")); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	_, err := (Reader{Root: root}).VerifyOperations()
	if err == nil {
		t.Fatal("VerifyOperations returned nil error for missing source object")
	}
}

func TestVerifyOperationsRejectsUnsafeSourceManifestObjectPath(t *testing.T) {
	tests := []string{
		"../outside.json",
		"objects/github/example/project/issues/../issues/000001.json",
	}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			root := writeTestLedger(t)
			source, err := (Reader{Root: root}).Source(model.Source{System: "github", Owner: "example", Repo: "project"})
			if err != nil {
				t.Fatalf("Source returned error: %v", err)
			}
			found := false
			for i := range source.Objects {
				if source.Objects[i].Object == "issue" {
					source.Objects[i].Path = path
					found = true
					break
				}
			}
			if !found {
				t.Fatal("fixture source has no issue object ref")
			}
			if err := writeJSON(filepath.Join(root, sourceManifestPath(source)), source); err != nil {
				t.Fatalf("writeJSON returned error: %v", err)
			}

			_, err = (Reader{Root: root}).VerifyOperations()
			if err == nil || !strings.Contains(err.Error(), "unsafe path") {
				t.Fatalf("VerifyOperations error = %v, want unsafe path error", err)
			}
		})
	}
}

func TestVerifyOperationsRejectsUnsafeChangePath(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	operation := model.Operation{
		ID:         NewOperationID("unsafe operation", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "unsafe operation",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root},
		Changes: []model.ObjectChange{{
			Type:   "created",
			Object: "issue",
			Path:   "../evil.json",
			SHA256: strings.Repeat("0", 64),
		}},
	}
	if err := writer.WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}

	_, err := (Reader{Root: root}).VerifyOperations()
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("VerifyOperations error = %v, want unsafe path error", err)
	}
}
