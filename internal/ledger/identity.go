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
	publicPath, err := safeRootedFilePath(root, filepath.Join("identities", "default.json"))
	if err != nil {
		return model.Identity{}, err
	}
	if _, err := os.Stat(publicPath); err == nil {
		return model.Identity{}, fmt.Errorf("default identity already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return model.Identity{}, err
	}
	privatePath, err := safeRootedWritePath(root, filepath.Join("identities", "default.key"))
	if err != nil {
		return model.Identity{}, err
	}
	if err := rejectExistingPrivateKeyPath(privatePath); err != nil {
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
	if err := writeJSONUnderRoot(root, filepath.Join("identities", "default.json"), identity); err != nil {
		return model.Identity{}, err
	}
	if err := writePrivateKey(root, filepath.Join("identities", "default.key"), privateKey); err != nil {
		return model.Identity{}, err
	}
	if err := (Writer{Root: root}).TrustIdentity(identity.ID); err != nil {
		return model.Identity{}, err
	}
	return identity, nil
}

func DefaultIdentity(root string) (model.Identity, error) {
	var identity model.Identity
	path, err := safeRootedFilePath(root, filepath.Join("identities", "default.json"))
	if err != nil {
		return model.Identity{}, err
	}
	if err := readJSONFile(path, &identity); err != nil {
		return model.Identity{}, err
	}
	return identity, nil
}

func defaultPrivateKey(root string) (ed25519.PrivateKey, error) {
	path, err := safeRootedFilePath(root, filepath.Join("identities", "default.key"))
	if err != nil {
		return nil, err
	}
	data, err := readFileNoSymlink(path)
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

func writePrivateKey(root, relative string, privateKey ed25519.PrivateKey) error {
	path, err := safeRootedWritePath(root, relative)
	if err != nil {
		return err
	}
	if err := rejectExistingPrivateKeyPath(path); err != nil {
		return err
	}
	file, err := createNewFile(path, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write([]byte(base64.StdEncoding.EncodeToString(privateKey))); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func rejectExistingPrivateKeyPath(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("default identity private key path %s is a symlink", path)
	}
	return fmt.Errorf("default identity private key already exists")
}
