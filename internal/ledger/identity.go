// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

const identityAlgorithmEd25519 = "ed25519"

func CreateDefaultIdentity(root, name string) (model.Identity, error) {
	publicPath := defaultIdentityPath(root)
	if _, err := os.Stat(publicPath); err == nil {
		return model.Identity{}, fmt.Errorf("default identity already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return model.Identity{}, err
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return model.Identity{}, err
	}
	idHash := sha256.Sum256(publicKey)
	identity := model.Identity{
		ID:        "key_" + hex.EncodeToString(idHash[:6]),
		Name:      name,
		Algorithm: identityAlgorithmEd25519,
		PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		CreatedAt: time.Now().UTC(),
	}
	if err := writeJSON(publicPath, identity); err != nil {
		return model.Identity{}, err
	}
	if err := writePrivateKey(defaultIdentityKeyPath(root), privateKey); err != nil {
		return model.Identity{}, err
	}
	return identity, nil
}

func DefaultIdentity(root string) (model.Identity, error) {
	var identity model.Identity
	if err := readJSONFile(defaultIdentityPath(root), &identity); err != nil {
		return model.Identity{}, err
	}
	return identity, nil
}

func defaultPrivateKey(root string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(defaultIdentityKeyPath(root)) // #nosec G304 -- key path is derived from the configured ledger root.
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	if len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("default identity private key has invalid length")
	}
	return ed25519.PrivateKey(key), nil
}

func writePrivateKey(path string, privateKey ed25519.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(privateKey)), 0o600) // #nosec G306 -- private identity key intentionally uses owner-only permissions.
}

func defaultIdentityPath(root string) string {
	return filepath.Join(root, "identities", "default.json")
}

func defaultIdentityKeyPath(root string) string {
	return filepath.Join(root, "identities", "default.key")
}
