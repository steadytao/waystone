// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

func TestWriteGitHubImport(t *testing.T) {
	root := t.TempDir()
	imported := model.GitHubImport{
		Project: model.Project{
			ID:        "github:repo:1",
			Name:      "example/project",
			URL:       "https://github.com/example/project",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		Source: model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		Issues: []model.Issue{
			{ID: "github:issue:10", SourceID: 10, Number: 1, Title: "issue"},
		},
		Comments: []model.Comment{
			{ID: "github:issue_comment:20", SourceID: 20, IssueNumber: 1},
		},
		PullRequests: []model.PullRequest{
			{ID: "github:pull_request:30", SourceID: 30, Number: 2, Title: "pr"},
		},
		ReviewComments: []model.ReviewComment{
			{ID: "github:review_comment:40", SourceID: 40, PullRequestNumber: 2},
		},
		Labels: []model.Label{
			{ID: "github:label:50", SourceID: 50, Name: "help wanted"},
		},
		Milestones: []model.Milestone{
			{ID: "github:milestone:60", SourceID: 60, Number: 1, Title: "v1"},
		},
		Releases: []model.Release{
			{ID: "github:release:70", SourceID: 70, TagName: "v0.1.0"},
		},
	}

	if err := (Writer{Root: root}).WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}

	for _, path := range []string{
		"ledger.json",
		"projects/github/example/project.json",
		"imports/github/example-project-759b2c42ed72.json",
		"objects/github/example/project/issues/000001.json",
		"objects/github/example/project/comments/github-issue_comment-20-c14c8ad5438e.json",
		"objects/github/example/project/pull_requests/000002.json",
		"objects/github/example/project/reviews/github-review_comment-40-dc5a8ba86019.json",
		"objects/github/example/project/labels/help-wanted-bbeeffdda6f9.json",
		"objects/github/example/project/milestones/000001.json",
		"objects/github/example/project/releases/github-release-70-a063378098e0.json",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected %s to be written: %v", path, err)
		}
	}

	var source model.Source
	if err := readJSONFile(filepath.Join(root, "imports/github/example-project-759b2c42ed72.json"), &source); err != nil {
		t.Fatalf("read source manifest: %v", err)
	}
	if len(source.Objects) != 8 {
		t.Fatalf("source objects = %d, want 8", len(source.Objects))
	}
	for _, object := range source.Objects {
		if object.Path == "" || object.SHA256 == "" {
			t.Fatalf("source object missing path/hash: %#v", object)
		}
	}
}

func TestRecordSourceOperationAddsOperationManifestEntry(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	source := model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"}
	imported := model.GitHubImport{
		Project: model.Project{ID: "github:repo:1", Name: "example/project"},
		Source:  source,
	}
	if err := writer.WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}
	operation := model.Operation{
		ID:         NewOperationID("github import", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "github import",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root},
	}
	if err := writer.WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}
	if err := writer.RecordSourceOperation(source, operation); err != nil {
		t.Fatalf("RecordSourceOperation returned error: %v", err)
	}

	var got model.Source
	if err := readJSONFile(filepath.Join(root, "imports/github/example-project-759b2c42ed72.json"), &got); err != nil {
		t.Fatalf("read source manifest: %v", err)
	}
	if len(got.Operations) != 1 {
		t.Fatalf("source operations = %d, want 1", len(got.Operations))
	}
	if got.Operations[0].ID != operation.ID || got.Operations[0].Path == "" {
		t.Fatalf("bad source operation ref: %#v", got.Operations[0])
	}
}

func TestWriteGitHubAuditRecordsAuditObjectAndSourceRefs(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	audit := model.GitHubAudit{
		ID:          "github-audit-20260101t000000.000000000z",
		Source:      model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Repository: model.GitHubAuditRepository{
			ID:            1,
			FullName:      "example/project",
			URL:           "https://github.com/example/project",
			DefaultBranch: "main",
		},
		Portable:           []string{"issues"},
		NeedsMigrationPlan: []string{"GitHub Actions workflows"},
	}
	operation := model.Operation{
		ID:         audit.ID,
		Command:    "github audit",
		StartedAt:  audit.GeneratedAt,
		FinishedAt: audit.GeneratedAt,
	}
	audit.Source.Operations = []model.SourceOperationRef{
		{ID: operation.ID, Command: operation.Command, Path: OperationPath(operation.ID), StartedAt: operation.StartedAt},
	}

	if err := writer.WriteGitHubAudit(audit); err != nil {
		t.Fatalf("WriteGitHubAudit returned error: %v", err)
	}

	audits, err := Reader{Root: root}.SourceAudits(audit.Source)
	if err != nil {
		t.Fatalf("SourceAudits returned error: %v", err)
	}
	if len(audits) != 1 || audits[0].ID != audit.ID {
		t.Fatalf("audits = %#v, want written audit", audits)
	}
	if len(audits[0].Source.Objects) != 0 || len(audits[0].Source.Operations) != 0 {
		t.Fatalf("audit source embedded manifest refs: %#v", audits[0].Source)
	}
	source, err := Reader{Root: root}.Source(audit.Source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(source.Objects) != 1 || source.Objects[0].Object != "audit" {
		t.Fatalf("source objects = %#v, want audit object ref", source.Objects)
	}
	if len(source.Operations) != 1 || source.Operations[0].ID != operation.ID {
		t.Fatalf("source operations = %#v, want audit operation ref", source.Operations)
	}
}

func TestWriteOperationSignsWhenDefaultIdentityExists(t *testing.T) {
	root := t.TempDir()
	identity, err := CreateDefaultIdentity(root, "test")
	if err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}
	operation := model.Operation{
		ID:         NewOperationID("ledger verify", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "ledger verify",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root},
	}

	if err := (Writer{Root: root}).WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}
	operations, err := (Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	if len(operations) != 1 || operations[0].Signature == nil || operations[0].Signature.IdentityID != identity.ID {
		t.Fatalf("operations = %#v, want signed operation", operations)
	}
	verification, err := (Reader{Root: root}).VerifyOperationSignatures()
	if err != nil {
		t.Fatalf("VerifyOperationSignatures returned error: %v", err)
	}
	if verification.Valid != 1 || verification.Unsigned != 0 {
		t.Fatalf("verification = %#v, want one valid signature", verification)
	}
	operations[0].Output.Ledger = filepath.Join(root, "tampered")
	if err := writeJSON(filepath.Join(root, operationRelativePath(operations[0].ID)), operations[0]); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}
	if _, err := (Reader{Root: root}).VerifyOperationSignatures(); err == nil {
		t.Fatal("VerifyOperationSignatures returned nil error for tampered operation")
	}
}

func TestWriteGitHubImportSignsSourceManifestWhenDefaultIdentityExists(t *testing.T) {
	root := t.TempDir()
	identity, err := CreateDefaultIdentity(root, "test")
	if err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}
	imported := model.GitHubImport{
		Project: model.Project{ID: "github:repo:1", Name: "example/project"},
		Source:  model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		Issues:  []model.Issue{{ID: "github:issue:1", Number: 1, Title: "issue"}},
	}

	if err := (Writer{Root: root}).WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}
	source, err := (Reader{Root: root}).Source(imported.Source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if source.Signature == nil || source.Signature.IdentityID != identity.ID {
		t.Fatalf("source signature = %#v, want signed source manifest", source.Signature)
	}
	verification, err := (Reader{Root: root}).VerifySourceSignatures()
	if err != nil {
		t.Fatalf("VerifySourceSignatures returned error: %v", err)
	}
	if verification.Valid != 1 || verification.Unsigned != 0 {
		t.Fatalf("verification = %#v, want one valid source signature", verification)
	}
	source.Objects = nil
	if err := writeJSON(filepath.Join(root, sourceManifestPath(imported.Source)), source); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}
	if _, err := (Reader{Root: root}).VerifySourceSignatures(); err == nil {
		t.Fatal("VerifySourceSignatures returned nil error for tampered source manifest")
	}
}

func TestSourcePathIsSystemScoped(t *testing.T) {
	source := model.Source{System: "waystone", Owner: "example", Repo: "project"}
	if got := SourcePath(source); got != "imports/waystone/example-project-759b2c42ed72.json" {
		t.Fatalf("SourcePath = %q, want imports/waystone/example-project-759b2c42ed72.json", got)
	}
}

func TestWriteGitHubImportPreservesLabelsWithCollidingSlugs(t *testing.T) {
	root := t.TempDir()
	imported := model.GitHubImport{
		Project: model.Project{ID: "github:repo:1", Name: "example/project"},
		Source:  model.Source{System: "github", Owner: "example", Repo: "project"},
		Labels: []model.Label{
			{ID: "github:label:1", SourceID: 1, Name: "a/b"},
			{ID: "github:label:2", SourceID: 2, Name: "a-b"},
		},
	}

	if err := (Writer{Root: root}).WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}

	paths, err := filepath.Glob(filepath.Join(root, "objects", "github", "example", "project", "labels", "*.json"))
	if err != nil {
		t.Fatalf("glob labels: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 label files, got %d: %v", len(paths), paths)
	}

	names := make(map[string]bool)
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read label %s: %v", path, err)
		}
		var label model.Label
		if err := json.Unmarshal(content, &label); err != nil {
			t.Fatalf("decode label %s: %v", path, err)
		}
		names[label.Name] = true
	}

	for _, name := range []string{"a/b", "a-b"} {
		if !names[name] {
			t.Fatalf("expected label %q to be preserved, got %v", name, names)
		}
	}
}

func TestWriterWritesLocalLabelByID(t *testing.T) {
	root := t.TempDir()
	writer := Writer{Root: root}
	source := model.Source{System: "waystone", Owner: "example", Repo: "project"}
	label := model.Label{
		Provenance:  model.Provenance{ImportID: "waystone:example/project", Source: source},
		ID:          "lbl_test",
		Slug:        "bug",
		Name:        "Software Issue",
		Color:       "d73a4a",
		Description: "Something broken",
		CreatedAt:   time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
	}

	if err := writer.WriteLocalLabel(label); err != nil {
		t.Fatalf("WriteLocalLabel returned error: %v", err)
	}

	path := filepath.Join(root, "objects", "waystone", "example", "project", "labels", "lbl_test.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected label to be written by ID: %v", err)
	}
	sourceManifest, err := (Reader{Root: root}).Source(source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(sourceManifest.Objects) != 1 || sourceManifest.Objects[0].Object != "label" || sourceManifest.Objects[0].ID != "lbl_test" {
		t.Fatalf("source objects = %#v, want label ref", sourceManifest.Objects)
	}
}

func TestReaderFindsLocalLabelBySlug(t *testing.T) {
	root := t.TempDir()
	source := model.Source{System: "waystone", Owner: "example", Repo: "project"}
	label := model.Label{
		Provenance: model.Provenance{ImportID: "waystone:example/project", Source: source},
		ID:         "lbl_test",
		Slug:       "bug",
		Name:       "Software Issue",
		CreatedAt:  time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
	}
	if err := (Writer{Root: root}).WriteLocalLabel(label); err != nil {
		t.Fatalf("WriteLocalLabel returned error: %v", err)
	}

	byID, err := (Reader{Root: root}).SourceLabelByID(source, "lbl_test")
	if err != nil {
		t.Fatalf("SourceLabelByID returned error: %v", err)
	}
	if byID.Name != "Software Issue" {
		t.Fatalf("label by ID = %#v", byID)
	}
	bySlug, err := (Reader{Root: root}).SourceLabelBySlug(source, "BUG")
	if err != nil {
		t.Fatalf("SourceLabelBySlug returned error: %v", err)
	}
	if bySlug.ID != "lbl_test" {
		t.Fatalf("label by slug = %#v", bySlug)
	}
}
