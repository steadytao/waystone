// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

func TestReaderSummary(t *testing.T) {
	root := writeTestLedger(t)

	summary, err := (Reader{Root: root}).Summary()
	if err != nil {
		t.Fatalf("Summary returned error: %v", err)
	}

	if len(summary.Projects) != 1 || summary.Projects[0].Name != "example/project" {
		t.Fatalf("projects = %#v, want example/project", summary.Projects)
	}
	if got := summary.Issues; got != 2 {
		t.Fatalf("issues = %d, want 2", got)
	}
	if got := summary.PullRequests; got != 1 {
		t.Fatalf("pull requests = %d, want 1", got)
	}
	if got := len(summary.Sources); got != 1 {
		t.Fatalf("sources = %d, want 1", got)
	}
}

func TestReaderSource(t *testing.T) {
	root := writeTestLedger(t)

	source, err := (Reader{Root: root}).Source(model.Source{System: "github", Owner: "example", Repo: "project"})
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if source.Owner != "example" || source.Repo != "project" {
		t.Fatalf("source = %#v, want example/project", source)
	}
	if len(source.Objects) == 0 {
		t.Fatal("source object manifest was empty")
	}
}

func TestSourceIssuesRejectsSymlinkedSourceParent(t *testing.T) {
	root := writeTestLedger(t)
	link := filepath.Join(root, "objects", "github", "example", "project")
	if err := os.RemoveAll(link); err != nil {
		t.Fatalf("RemoveAll returned error: %v", err)
	}
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	_, err := (Reader{Root: root}).SourceIssues(model.Source{System: "github", Owner: "example", Repo: "project"})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("SourceIssues error = %v, want symlink rejection", err)
	}
}

func TestReaderSourcesReadsAllSystems(t *testing.T) {
	root := writeTestLedger(t)
	source := model.Source{System: "waystone", Owner: "example", Repo: "project", URL: "waystone:example/project"}
	if err := writeJSON(filepath.Join(root, sourceManifestPath(source)), source); err != nil {
		t.Fatalf("write waystone source manifest: %v", err)
	}

	sources, err := (Reader{Root: root}).Sources()
	if err != nil {
		t.Fatalf("Sources returned error: %v", err)
	}
	if got := sourceSpecs(sources); len(got) != 2 || got[0] != "github:example/project" || got[1] != "waystone:example/project" {
		t.Fatalf("sources = %v, want [github:example/project waystone:example/project]", got)
	}
}

func TestReaderIssue(t *testing.T) {
	root := writeTestLedger(t)

	issue, err := (Reader{Root: root}).Issue(2)
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	if issue.Title != "second issue" {
		t.Fatalf("issue title = %q, want second issue", issue.Title)
	}
}

func TestReaderPullRequest(t *testing.T) {
	root := writeTestLedger(t)

	pr, err := (Reader{Root: root}).PullRequest(3)
	if err != nil {
		t.Fatalf("PullRequest returned error: %v", err)
	}
	if pr.Title != "pr" {
		t.Fatalf("pr title = %q, want pr", pr.Title)
	}
}

func TestReaderReviewComments(t *testing.T) {
	root := writeTestLedger(t)

	comments, err := (Reader{Root: root}).ReviewComments(3)
	if err != nil {
		t.Fatalf("ReviewComments returned error: %v", err)
	}
	if got := len(comments); got != 1 {
		t.Fatalf("review comments = %d, want 1", got)
	}
}

func TestReaderLabelsAndMilestones(t *testing.T) {
	root := writeTestLedger(t)

	labels, err := (Reader{Root: root}).Labels()
	if err != nil {
		t.Fatalf("Labels returned error: %v", err)
	}
	if got := len(labels); got != 1 {
		t.Fatalf("labels = %d, want 1", got)
	}
	milestones, err := (Reader{Root: root}).Milestones()
	if err != nil {
		t.Fatalf("Milestones returned error: %v", err)
	}
	if got := len(milestones); got != 1 {
		t.Fatalf("milestones = %d, want 1", got)
	}
}

func TestReaderIssuesSortsBySourceThenNumber(t *testing.T) {
	root := writeTestLedger(t)
	source := model.Source{System: "waystone", Owner: "example", Repo: "project", URL: "waystone:example/project"}
	provenance := model.Provenance{ImportID: "waystone:example/project", Source: source}
	imported := model.GitHubImport{
		Project: model.Project{ID: "waystone:repo:1", Name: "example/project-local"},
		Source:  source,
		Issues: []model.Issue{
			{Provenance: provenance, ID: "waystone:issue:1", Number: 1, Title: "local issue"},
		},
	}
	if err := (Writer{Root: root}).WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}

	issues, err := (Reader{Root: root}).Issues()
	if err != nil {
		t.Fatalf("Issues returned error: %v", err)
	}
	if got := issueOrder(issues); len(got) != 3 || got[0] != "github:example/project#1" || got[1] != "github:example/project#2" || got[2] != "waystone:example/project#1" {
		t.Fatalf("issue order = %v, want [github:example/project#1 github:example/project#2 waystone:example/project#1]", got)
	}
}

func sourceSpecs(sources []model.Source) []string {
	specs := make([]string, 0, len(sources))
	for _, source := range sources {
		specs = append(specs, SourceSpec(source))
	}
	return specs
}

func issueOrder(issues []model.Issue) []string {
	order := make([]string, 0, len(issues))
	for _, issue := range issues {
		order = append(order, SourceSpec(issue.Source)+"#"+strconv.Itoa(issue.Number))
	}
	return order
}

func writeTestLedger(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"}
	provenance := model.Provenance{ImportID: "github:example/project", Source: source}
	imported := model.GitHubImport{
		Project: model.Project{
			ID:        "github:repo:1",
			Name:      "example/project",
			URL:       "https://github.com/example/project",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		Source: source,
		Issues: []model.Issue{
			{Provenance: provenance, ID: "github:issue:10", SourceID: 10, Number: 1, Title: "first issue", State: "open"},
			{Provenance: provenance, ID: "github:issue:11", SourceID: 11, Number: 2, Title: "second issue", State: "closed"},
		},
		Comments: []model.Comment{
			{Provenance: provenance, ID: "github:issue_comment:20", SourceID: 20, IssueNumber: 1},
		},
		PullRequests: []model.PullRequest{
			{Provenance: provenance, ID: "github:pull_request:30", SourceID: 30, Number: 3, Title: "pr", State: "open"},
		},
		ReviewComments: []model.ReviewComment{
			{Provenance: provenance, ID: "github:review_comment:40", SourceID: 40, PullRequestNumber: 3},
		},
		Labels: []model.Label{
			{Provenance: provenance, ID: "github:label:50", SourceID: 50, Name: "bug"},
		},
		Milestones: []model.Milestone{
			{Provenance: provenance, ID: "github:milestone:60", SourceID: 60, Number: 1, Title: "v1"},
		},
		Releases: []model.Release{
			{Provenance: provenance, ID: "github:release:70", SourceID: 70, TagName: "v0.1.0"},
		},
	}
	if err := (Writer{Root: root}).WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}
	return root
}
