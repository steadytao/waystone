// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPlaintextStoreRejectsSymlinkedTokenPath(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "github.json")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	store := PlaintextStore{Root: root}

	err := store.SaveGitHubToken(GitHubToken{AccessToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("SaveGitHubToken error = %v, want symlink rejection", err)
	}
	data, readErr := os.ReadFile(outside)
	if readErr != nil {
		t.Fatalf("ReadFile returned error: %v", readErr)
	}
	if string(data) != "outside" {
		t.Fatalf("symlink target content = %q, want unchanged", string(data))
	}
	if _, err := store.GitHubToken(); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("GitHubToken error = %v, want symlink rejection", err)
	}
}

func TestPlaintextStoreRejectsSymlinkedRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	if err := os.Symlink(t.TempDir(), root); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	store := PlaintextStore{Root: root}

	err := store.SaveGitHubToken(GitHubToken{AccessToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("SaveGitHubToken error = %v, want symlink rejection", err)
	}
}
