// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuditRepositoryFindsGitHubDependencySurfaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/example/project":
			writeJSON(t, w, ghRepository{
				ID:            1,
				FullName:      "example/project",
				Description:   "test project",
				HTMLURL:       "https://github.com/example/project",
				DefaultBranch: "main",
				CreatedAt:     mustTime("2026-01-01T00:00:00Z"),
				UpdatedAt:     mustTime("2026-01-02T00:00:00Z"),
			})
		case "/repos/example/project/contents/.github/workflows":
			writeJSON(t, w, []ghContent{
				{Name: "ci.yml", Path: ".github/workflows/ci.yml", Type: "file"},
				{Name: "release.yml", Path: ".github/workflows/release.yml", Type: "file"},
			})
		case "/repos/example/project/contents/.github/workflows/ci.yml":
			writeContent(t, w, `.github/workflows/ci.yml`, "name: CI\njobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n      - uses: ./.github/actions/setup\n      - uses: owner/reusable/.github/workflows/release.yml@v1\n")
		case "/repos/example/project/contents/.github/workflows/release.yml":
			writeContent(t, w, `.github/workflows/release.yml`, "name: Release\njobs:\n  release:\n    steps:\n      - uses: docker/login-action@v3\n")
		case "/repos/example/project/contents/.github/dependabot.yml":
			writeContent(t, w, `.github/dependabot.yml`, "version: 2\n")
		case "/repos/example/project/contents/.github/codeql.yml":
			writeContent(t, w, `.github/codeql.yml`, "name: codeql\n")
		case "/repos/example/project/contents/.github/ISSUE_TEMPLATE":
			writeJSON(t, w, []ghContent{{Name: "bug.yml", Path: ".github/ISSUE_TEMPLATE/bug.yml", Type: "file"}})
		case "/repos/example/project/contents/.github/PULL_REQUEST_TEMPLATE.md":
			writeContent(t, w, `.github/PULL_REQUEST_TEMPLATE.md`, "## Summary\n")
		case "/repos/example/project/contents/CODEOWNERS":
			writeContent(t, w, `CODEOWNERS`, "* @example/maintainers\n")
		case "/repos/example/project/branches/main/protection":
			_, _ = w.Write([]byte(`{
				"required_status_checks": {"contexts": ["test"], "checks": [{"context": "lint"}]},
				"required_pull_request_reviews": {"required_approving_review_count": 2, "require_code_owner_reviews": true},
				"enforce_admins": {"enabled": true}
			}`))
		case "/repos/example/project/actions/secrets":
			_, _ = w.Write([]byte(`{"total_count":3,"secrets":[{"name":"TOKEN"}]}`))
		case "/repos/example/project/actions/variables":
			_, _ = w.Write([]byte(`{"total_count":2,"variables":[{"name":"REGION"}]}`))
		case "/repos/example/project/environments":
			_, _ = w.Write([]byte(`{"total_count":1,"environments":[{"name":"production"}]}`))
		case "/repos/example/project/pages":
			_, _ = w.Write([]byte(`{"html_url":"https://example.github.io/project/"}`))
		case "/repos/example/project/releases":
			_, _ = w.Write([]byte(`[{"id":1,"tag_name":"v1.0.0","assets":[{"id":10,"name":"waystone.zip"}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	audit, err := NewClient(server.URL, "", time.Second).AuditRepository(context.Background(), "example", "project")
	if err != nil {
		t.Fatalf("AuditRepository returned error: %v", err)
	}

	if audit.Repository.FullName != "example/project" {
		t.Fatalf("repository = %q, want example/project", audit.Repository.FullName)
	}
	if len(audit.Workflows) != 2 {
		t.Fatalf("workflows = %d, want 2", len(audit.Workflows))
	}
	if len(audit.Actions) != 4 {
		t.Fatalf("actions = %d, want 4", len(audit.Actions))
	}
	if !audit.Dependabot.Present {
		t.Fatal("dependabot config was not detected")
	}
	if !audit.CodeQL.Present {
		t.Fatal("codeql config was not detected")
	}
	if !audit.IssueTemplates.Present {
		t.Fatal("issue templates were not detected")
	}
	if !audit.PullRequestTemplate.Present {
		t.Fatal("pull request template was not detected")
	}
	if !audit.Codeowners.Present {
		t.Fatal("CODEOWNERS was not detected")
	}
	if !audit.BranchProtection.Present || audit.BranchProtection.RequiredStatusChecks != 2 {
		t.Fatalf("branch protection = %#v, want required checks", audit.BranchProtection)
	}
	if !audit.Secrets.Accessible || audit.Secrets.Count != 3 {
		t.Fatalf("secrets = %#v, want accessible count", audit.Secrets)
	}
	if !audit.Variables.Accessible || audit.Variables.Count != 2 {
		t.Fatalf("variables = %#v, want accessible count", audit.Variables)
	}
	if !audit.Environments.Accessible || audit.Environments.Count != 1 {
		t.Fatalf("environments = %#v, want accessible count", audit.Environments)
	}
	if !audit.Pages.Present {
		t.Fatal("GitHub Pages was not detected")
	}
	if audit.ReleaseAssets.Releases != 1 || audit.ReleaseAssets.Assets != 1 {
		t.Fatalf("release assets = %#v, want one release asset", audit.ReleaseAssets)
	}
	if !containsAuditFinding(audit.NeedsMigrationPlan, "GitHub Actions workflows") {
		t.Fatalf("needs migration plan = %#v, want workflows", audit.NeedsMigrationPlan)
	}
	if !containsAuditFinding(audit.Portable, "issues") {
		t.Fatalf("portable = %#v, want issues", audit.Portable)
	}
}

func TestAuditRepositoryReturnsOptionalContentErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/example/project":
			writeJSON(t, w, ghRepository{
				ID:            1,
				FullName:      "example/project",
				HTMLURL:       "https://github.com/example/project",
				DefaultBranch: "main",
				CreatedAt:     mustTime("2026-01-01T00:00:00Z"),
				UpdatedAt:     mustTime("2026-01-02T00:00:00Z"),
			})
		case "/repos/example/project/contents/.github/workflows":
			http.NotFound(w, r)
		case "/repos/example/project/contents/.github/dependabot.yml":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"temporary failure"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := NewClient(server.URL, "", time.Second).AuditRepository(context.Background(), "example", "project")
	if err == nil || !strings.Contains(err.Error(), "temporary failure") {
		t.Fatalf("AuditRepository error = %v, want optional content failure", err)
	}
}

func containsAuditFinding(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeContent(t *testing.T, w http.ResponseWriter, path, content string) {
	t.Helper()
	writeJSON(t, w, ghContent{
		Name:     path,
		Path:     path,
		Type:     "file",
		Encoding: "base64",
		Content:  base64.StdEncoding.EncodeToString([]byte(content)),
	})
}
