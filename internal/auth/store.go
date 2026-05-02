// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "waystone"
	keyringUser    = "github"
)

type GitHubToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	CreatedAt   time.Time `json:"created_at"`
}

type CredentialStore interface {
	SaveGitHubToken(GitHubToken) error
	GitHubToken() (GitHubToken, error)
	DeleteGitHubToken() error
	Description() string
}

type KeyringStore struct{}

func DefaultStore() CredentialStore {
	return KeyringStore{}
}

func (KeyringStore) SaveGitHubToken(token GitHubToken) error {
	token, err := normalizeToken(token)
	if err != nil {
		return err
	}
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, keyringUser, string(data))
}

func (KeyringStore) GitHubToken() (GitHubToken, error) {
	data, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return GitHubToken{}, err
	}
	var token GitHubToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return GitHubToken{}, fmt.Errorf("decoding keyring token: %w", err)
	}
	return token, nil
}

func (KeyringStore) DeleteGitHubToken() error {
	err := keyring.Delete(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func (KeyringStore) Description() string {
	return "the OS credential store"
}

type PlaintextStore struct {
	Root string
}

func DefaultPlaintextStore() (PlaintextStore, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return PlaintextStore{}, err
	}
	return PlaintextStore{Root: filepath.Join(root, "waystone")}, nil
}

func (s PlaintextStore) SaveGitHubToken(token GitHubToken) error {
	token, err := normalizeToken(token)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.path(), data, 0o600)
}

func (s PlaintextStore) GitHubToken() (GitHubToken, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		return GitHubToken{}, err
	}
	var token GitHubToken
	if err := json.Unmarshal(data, &token); err != nil {
		return GitHubToken{}, fmt.Errorf("decoding %s: %w", s.path(), err)
	}
	return token, nil
}

func (s PlaintextStore) DeleteGitHubToken() error {
	err := os.Remove(s.path())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s PlaintextStore) Description() string {
	return "plaintext file " + s.path()
}

func (s PlaintextStore) path() string {
	return filepath.Join(s.Root, "github.json")
}

func normalizeToken(token GitHubToken) (GitHubToken, error) {
	if token.AccessToken == "" {
		return GitHubToken{}, errors.New("github token must not be empty")
	}
	if token.TokenType == "" {
		token.TokenType = "bearer"
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	return token, nil
}
