// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"testing"
	"time"
)

func TestStoreGitHubToken(t *testing.T) {
	store := PlaintextStore{Root: t.TempDir()}
	token := GitHubToken{
		AccessToken: "token",
		TokenType:   "bearer",
		Scope:       "repo",
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := store.SaveGitHubToken(token); err != nil {
		t.Fatalf("SaveGitHubToken returned error: %v", err)
	}

	got, err := store.GitHubToken()
	if err != nil {
		t.Fatalf("GitHubToken returned error: %v", err)
	}
	if got.AccessToken != token.AccessToken {
		t.Fatalf("access token = %q, want %q", got.AccessToken, token.AccessToken)
	}
	if got.Scope != token.Scope {
		t.Fatalf("scope = %q, want %q", got.Scope, token.Scope)
	}
}

func TestPlaintextStoreDeleteGitHubToken(t *testing.T) {
	store := PlaintextStore{Root: t.TempDir()}

	if err := store.SaveGitHubToken(GitHubToken{AccessToken: "token"}); err != nil {
		t.Fatalf("SaveGitHubToken returned error: %v", err)
	}
	if err := store.DeleteGitHubToken(); err != nil {
		t.Fatalf("DeleteGitHubToken returned error: %v", err)
	}
	if _, err := store.GitHubToken(); err == nil {
		t.Fatal("GitHubToken returned nil error after delete")
	}
}
