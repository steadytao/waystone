// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestImportRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			writeJSON(t, w, ghUser{ID: 99, Login: "steadytao", HTMLURL: "https://github.com/steadytao"})
		case "/repos/example/project":
			writeJSON(t, w, ghRepository{
				ID:          1,
				FullName:    "example/project",
				Description: "test project",
				HTMLURL:     "https://github.com/example/project",
				CreatedAt:   mustTime("2026-01-01T00:00:00Z"),
				UpdatedAt:   mustTime("2026-01-02T00:00:00Z"),
			})
		case "/repos/example/project/issues":
			writeJSON(t, w, []ghIssue{
				{
					ID:        10,
					Number:    1,
					Title:     "real issue",
					State:     "open",
					User:      ghUser{ID: 100, Login: "alice", HTMLURL: "https://github.com/alice"},
					HTMLURL:   "https://github.com/example/project/issues/1",
					CreatedAt: mustTime("2026-01-03T00:00:00Z"),
					UpdatedAt: mustTime("2026-01-04T00:00:00Z"),
				},
				{
					ID:     11,
					Number: 2,
					Title:  "pull request",
					User:   ghUser{ID: 101, Login: "bob", HTMLURL: "https://github.com/bob"},
					PullRequest: &struct {
						URL string `json:"url"`
					}{URL: "https://api.github.com/repos/example/project/pulls/2"},
				},
			})
		case "/repos/example/project/issues/1/comments":
			writeJSON(t, w, []ghComment{
				{
					ID:        20,
					User:      ghUser{ID: 102, Login: "carol", HTMLURL: "https://github.com/carol"},
					Body:      "comment",
					HTMLURL:   "https://github.com/example/project/issues/1#issuecomment-20",
					CreatedAt: mustTime("2026-01-05T00:00:00Z"),
					UpdatedAt: mustTime("2026-01-06T00:00:00Z"),
				},
			})
		case "/repos/example/project/issues/2/comments":
			writeJSON(t, w, []ghComment{
				{
					ID:        21,
					User:      ghUser{ID: 104, Login: "erin", HTMLURL: "https://github.com/erin"},
					Body:      "pr conversation",
					HTMLURL:   "https://github.com/example/project/pull/2#issuecomment-21",
					CreatedAt: mustTime("2026-01-06T00:00:00Z"),
					UpdatedAt: mustTime("2026-01-07T00:00:00Z"),
				},
			})
		case "/repos/example/project/pulls/2":
			writeJSON(t, w, ghPullRequest{
				ID:        30,
				Number:    2,
				Title:     "pull request",
				State:     "open",
				User:      ghUser{ID: 101, Login: "bob", HTMLURL: "https://github.com/bob"},
				HTMLURL:   "https://github.com/example/project/pull/2",
				CreatedAt: mustTime("2026-01-07T00:00:00Z"),
				UpdatedAt: mustTime("2026-01-08T00:00:00Z"),
				Base: struct {
					Ref string `json:"ref"`
				}{Ref: "main"},
				Head: struct {
					Ref string `json:"ref"`
				}{Ref: "feature"},
			})
		case "/repos/example/project/pulls/2/comments":
			writeJSON(t, w, []ghReviewComment{
				{
					ID:        40,
					User:      ghUser{ID: 103, Login: "dave", HTMLURL: "https://github.com/dave"},
					Body:      "review",
					HTMLURL:   "https://github.com/example/project/pull/2#discussion_r40",
					Path:      "main.go",
					Line:      12,
					CreatedAt: mustTime("2026-01-09T00:00:00Z"),
					UpdatedAt: mustTime("2026-01-10T00:00:00Z"),
				},
			})
		case "/repos/example/project/labels":
			writeJSON(t, w, []ghLabel{{ID: 50, Name: "bug", Color: "d73a4a"}})
		case "/repos/example/project/milestones":
			writeJSON(t, w, []ghMilestone{{ID: 60, Number: 1, Title: "v1", State: "open"}})
		case "/repos/example/project/releases":
			writeJSON(t, w, []ghRelease{{ID: 70, TagName: "v0.1.0", Name: "v0.1.0"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "", time.Second)
	var progress []Progress
	client.WithProgress(func(event Progress) {
		progress = append(progress, event)
	})
	imported, err := client.ImportRepository(context.Background(), "example", "project")
	if err != nil {
		t.Fatalf("ImportRepository returned error: %v", err)
	}

	if got := len(imported.Issues); got != 1 {
		t.Fatalf("issues = %d, want 1", got)
	}
	if got := len(imported.PullRequests); got != 1 {
		t.Fatalf("pull requests = %d, want 1", got)
	}
	if got := len(imported.Comments); got != 2 {
		t.Fatalf("comments = %d, want 2", got)
	}
	if got := len(imported.ReviewComments); got != 1 {
		t.Fatalf("review comments = %d, want 1", got)
	}
	if got := len(imported.Labels); got != 1 {
		t.Fatalf("labels = %d, want 1", got)
	}
	if got := len(imported.Milestones); got != 1 {
		t.Fatalf("milestones = %d, want 1", got)
	}
	if got := len(imported.Releases); got != 1 {
		t.Fatalf("releases = %d, want 1", got)
	}
	assertProvenance(t, imported.Issues[0].ImportID, imported.Issues[0].Source.System)
	assertProvenance(t, imported.Comments[0].ImportID, imported.Comments[0].Source.System)
	assertProvenance(t, imported.PullRequests[0].ImportID, imported.PullRequests[0].Source.System)
	assertProvenance(t, imported.ReviewComments[0].ImportID, imported.ReviewComments[0].Source.System)
	assertProvenance(t, imported.Labels[0].ImportID, imported.Labels[0].Source.System)
	assertProvenance(t, imported.Milestones[0].ImportID, imported.Milestones[0].Source.System)
	assertProvenance(t, imported.Releases[0].ImportID, imported.Releases[0].Source.System)
	if len(progress) == 0 {
		t.Fatal("progress callback was not called")
	}
	if !hasProgressDetail(progress) {
		t.Fatal("progress callback did not include detail events")
	}
	if !hasProgressMessage(progress, "Fetched pull request #2") {
		t.Fatalf("progress did not include completion event: %#v", progress)
	}
}

func TestAuthenticatedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/user" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		writeJSON(t, w, ghUser{ID: 1, Login: "steadytao"})
	}))
	defer server.Close()

	login, err := NewClient(server.URL, "token", time.Second).AuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("AuthenticatedUser returned error: %v", err)
	}
	if login != "steadytao" {
		t.Fatalf("login = %q, want steadytao", login)
	}
}

func hasProgressDetail(progress []Progress) bool {
	for _, event := range progress {
		if event.Detail {
			return true
		}
	}
	return false
}

func hasProgressMessage(progress []Progress, message string) bool {
	for _, event := range progress {
		if event.Message == message {
			return true
		}
	}
	return false
}

func TestConcurrentMapPreservesInputOrder(t *testing.T) {
	got, err := concurrentMap(context.Background(), 3, []int{1, 2, 3, 4}, func(_ context.Context, value int) (int, error) {
		return value * 10, nil
	})
	if err != nil {
		t.Fatalf("concurrentMap returned error: %v", err)
	}
	want := []int{10, 20, 30, 40}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("results = %v, want %v", got, want)
	}
}

func assertProvenance(t *testing.T, importID, system string) {
	t.Helper()
	if importID != "github:example/project" {
		t.Fatalf("import ID = %q, want github:example/project", importID)
	}
	if system != "github" {
		t.Fatalf("source system = %q, want github", system)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func mustTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t
}
