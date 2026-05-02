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
