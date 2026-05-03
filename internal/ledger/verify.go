// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/steadytao/waystone/internal/model"
)

type Verification struct {
	Files    int
	Checksum string
	Changes  []model.ObjectChange
}

type OperationVerification struct {
	Operations int
	Files      int
	Checksum   string
	Changes    []model.ObjectChange
}

type OperationSignatureVerification struct {
	Operations int
	Valid      int
	Unsigned   int
}

func (r Reader) Verify() (Verification, error) {
	var files []string
	if err := filepath.WalkDir(r.Root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		relative, err := filepath.Rel(r.Root, path)
		if err != nil {
			return err
		}
		if isOperationPath(relative) {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return Verification{}, err
	}
	sort.Strings(files)

	var changes []model.ObjectChange
	hash := sha256.New()
	for _, path := range files {
		data, err := os.ReadFile(path) // #nosec G304 -- verification walks files under the configured ledger root.
		if err != nil {
			return Verification{}, err
		}
		if !json.Valid(data) {
			return Verification{}, fmt.Errorf("invalid JSON: %s", path)
		}
		relative, err := filepath.Rel(r.Root, path)
		if err != nil {
			return Verification{}, err
		}
		hash.Write([]byte(filepath.ToSlash(relative)))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
		sum := sha256.Sum256(data)
		changes = append(changes, model.ObjectChange{
			Type:   "verified",
			Object: objectFromLedgerPath(relative),
			Path:   filepath.ToSlash(relative),
			SHA256: hex.EncodeToString(sum[:]),
		})
	}

	return Verification{
		Files:    len(files),
		Checksum: hex.EncodeToString(hash.Sum(nil)),
		Changes:  changes,
	}, nil
}

func isOperationPath(relative string) bool {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	return len(parts) > 0 && parts[0] == "operations"
}

func objectFromLedgerPath(relative string) string {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) == 0 {
		return "file"
	}
	if parts[0] == "ledger.json" {
		return "ledger"
	}
	if parts[0] == "projects" {
		return "project"
	}
	if len(parts) >= 5 && parts[0] == "objects" {
		return strings.TrimSuffix(parts[4], "s")
	}
	if len(parts) >= 2 && parts[0] == "imports" {
		return "source"
	}
	return "file"
}

func (r Reader) VerifyOperations() (OperationVerification, error) {
	operations, err := r.Operations()
	if err != nil {
		return OperationVerification{}, err
	}

	expectedFiles := make(map[string]string)
	hash := sha256.New()
	changes := make([]model.ObjectChange, 0, len(operations))
	var previousID string

	for _, operation := range operations {
		if operation.OperationHash == "" {
			return OperationVerification{}, fmt.Errorf("operation %s has no operation_hash", operation.ID)
		}
		if operation.PreviousOperation != previousID {
			return OperationVerification{}, fmt.Errorf("operation %s previous_operation = %q, want %q", operation.ID, operation.PreviousOperation, previousID)
		}
		gotHash, err := OperationHash(operation)
		if err != nil {
			return OperationVerification{}, err
		}
		if gotHash != operation.OperationHash {
			return OperationVerification{}, fmt.Errorf("operation %s hash mismatch", operation.ID)
		}

		relative := filepath.ToSlash(operationRelativePath(operation.ID))
		data, err := os.ReadFile(filepath.Join(r.Root, relative)) // #nosec G304 -- operation path is derived from recorded operation ID.
		if err != nil {
			return OperationVerification{}, err
		}
		fileHash := sha256.Sum256(data)
		hash.Write([]byte(relative))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
		changes = append(changes, model.ObjectChange{
			Type:   "verified",
			Object: "operation",
			ID:     operation.ID,
			Path:   relative,
			SHA256: hex.EncodeToString(fileHash[:]),
		})

		for _, change := range operation.Changes {
			if change.Path == "" {
				continue
			}
			if change.SHA256 == "" {
				return OperationVerification{}, fmt.Errorf("operation %s change %s has no sha256", operation.ID, change.Path)
			}
			path := filepath.ToSlash(change.Path)
			if change.Type == "deleted" {
				delete(expectedFiles, path)
				continue
			}
			expectedFiles[path] = change.SHA256
		}
		previousID = operation.ID
	}

	for path, expectedHash := range expectedFiles {
		actualHash, err := r.fileHash(path)
		if err != nil {
			return OperationVerification{}, err
		}
		if actualHash != expectedHash {
			return OperationVerification{}, fmt.Errorf("file %s hash mismatch", path)
		}
	}

	return OperationVerification{
		Operations: len(operations),
		Files:      len(expectedFiles),
		Checksum:   hex.EncodeToString(hash.Sum(nil)),
		Changes:    changes,
	}, nil
}

func (r Reader) VerifyOperationSignatures() (OperationSignatureVerification, error) {
	operations, err := r.Operations()
	if err != nil {
		return OperationSignatureVerification{}, err
	}
	verification := OperationSignatureVerification{Operations: len(operations)}
	for _, operation := range operations {
		if operation.Signature == nil || operation.Signature.Value == "" {
			verification.Unsigned++
			continue
		}
		if err := verifyOperationSignature(operation); err != nil {
			return OperationSignatureVerification{}, fmt.Errorf("operation %s signature: %w", operation.ID, err)
		}
		verification.Valid++
	}
	return verification, nil
}

func verifyOperationSignature(operation model.Operation) error {
	if operation.Signature == nil {
		return fmt.Errorf("missing signature")
	}
	signature := *operation.Signature
	if signature.Algorithm != identityAlgorithmEd25519 {
		return fmt.Errorf("unsupported algorithm %q", signature.Algorithm)
	}
	publicKey, err := base64.StdEncoding.DecodeString(signature.PublicKey)
	if err != nil {
		return err
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("public key has invalid length")
	}
	value, err := base64.StdEncoding.DecodeString(signature.Value)
	if err != nil {
		return err
	}
	data, err := operationSigningBytes(operation)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), data, value) {
		return fmt.Errorf("verification failed")
	}
	return nil
}

func (r Reader) fileHash(relative string) (string, error) {
	path, err := SafeRootedPath(r.Root, relative)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is validated by SafeRootedPath before reading.
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
