// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"archive/tar"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/steadytao/waystone/internal/model"
)

const archiveManifestName = "WAYSTONE-MANIFEST.json"

func buildArchiveManifest(root string) (model.ArchiveManifest, error) {
	verification, err := (Reader{Root: root}).Verify()
	if err != nil {
		return model.ArchiveManifest{}, err
	}
	operationVerification, err := (Reader{Root: root}).VerifyOperations()
	if err != nil {
		return model.ArchiveManifest{}, err
	}
	operations, err := (Reader{Root: root}).Operations()
	if err != nil {
		return model.ArchiveManifest{}, err
	}
	sources, err := (Reader{Root: root}).Sources()
	if err != nil {
		return model.ArchiveManifest{}, err
	}
	files, err := archiveFileRefs(root)
	if err != nil {
		return model.ArchiveManifest{}, err
	}
	manifest := model.ArchiveManifest{
		Version:         "waystone.archive.v1",
		CreatedAt:       time.Now().UTC(),
		Format:          "tar+zstd",
		Files:           files,
		Sources:         sources,
		LedgerChecksum:  verification.Checksum,
		Operations:      operationVerification.Operations,
		OperationSHA256: operationVerification.Checksum,
	}
	if len(operations) > 0 {
		head := operations[len(operations)-1]
		manifest.OperationHead = head.ID
		manifest.OperationHash = head.OperationHash
	}
	return manifest, nil
}

func archiveFileRefs(root string) ([]model.ArchiveFileRef, error) {
	var files []string
	if err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := rejectSymlinkEntry(filePath, entry); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if isPrivateIdentityKey(name) {
			return nil
		}
		files = append(files, name)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)

	refs := make([]model.ArchiveFileRef, 0, len(files))
	for _, name := range files {
		filePath, err := SafeRootedPath(root, name)
		if err != nil {
			return nil, err
		}
		data, err := readFileNoSymlink(filePath)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		refs = append(refs, model.ArchiveFileRef{
			Path:   name,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(data)),
		})
	}
	return refs, nil
}

func signArchiveManifest(root string, manifest *model.ArchiveManifest) error {
	identity, err := DefaultIdentity(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	privateKey, err := defaultPrivateKey(root)
	if err != nil {
		return err
	}
	data, err := archiveManifestSigningBytes(*manifest)
	if err != nil {
		return err
	}
	signature, err := (Writer{Root: root}).signPayload(identity, privateKey, data)
	if err != nil {
		return err
	}
	manifest.Signature = signature
	return nil
}

func ReadArchiveManifest(archivePath string) (model.ArchiveManifest, error) {
	var manifest model.ArchiveManifest
	err := walkArchive(archivePath, func(header *tar.Header, data []byte) error {
		if header.Name != archiveManifestName {
			return nil
		}
		if manifest.Version != "" {
			return fmt.Errorf("duplicate archive manifest")
		}
		return json.Unmarshal(data, &manifest)
	})
	if err != nil {
		return model.ArchiveManifest{}, err
	}
	if manifest.Version == "" {
		return model.ArchiveManifest{}, fmt.Errorf("archive manifest missing")
	}
	return manifest, nil
}

func VerifyArchiveManifest(archivePath string) (model.ArchiveManifest, error) {
	manifest, entries, err := readArchiveEntries(archivePath)
	if err != nil {
		return model.ArchiveManifest{}, err
	}
	expected := make(map[string]model.ArchiveFileRef, len(manifest.Files))
	for _, ref := range manifest.Files {
		if _, ok := expected[ref.Path]; ok {
			return model.ArchiveManifest{}, fmt.Errorf("archive manifest lists duplicate path %s", ref.Path)
		}
		if ref.Path == archiveManifestName {
			return model.ArchiveManifest{}, fmt.Errorf("archive manifest must not list itself")
		}
		if isPrivateIdentityKey(ref.Path) {
			return model.ArchiveManifest{}, fmt.Errorf("archive manifest includes private identity key %s", ref.Path)
		}
		if _, err := cleanArchivePath(ref.Path); err != nil {
			return model.ArchiveManifest{}, err
		}
		expected[ref.Path] = ref
	}
	for name, data := range entries {
		ref, ok := expected[name]
		if !ok {
			return model.ArchiveManifest{}, fmt.Errorf("archive entry %s is not listed in manifest", name)
		}
		if int64(len(data)) != ref.Size {
			return model.ArchiveManifest{}, fmt.Errorf("archive entry %s size mismatch", name)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != ref.SHA256 {
			return model.ArchiveManifest{}, fmt.Errorf("archive entry %s hash mismatch", name)
		}
		delete(expected, name)
	}
	for name := range expected {
		return model.ArchiveManifest{}, fmt.Errorf("archive manifest lists missing entry %s", name)
	}
	if manifest.Signature != nil && manifest.Signature.Value != "" {
		if err := verifyArchiveManifestSignature(manifest); err != nil {
			return model.ArchiveManifest{}, fmt.Errorf("archive manifest signature: %w", err)
		}
	}
	return manifest, nil
}

func verifyArchiveManifestSignature(manifest model.ArchiveManifest) error {
	if manifest.Signature == nil {
		return fmt.Errorf("missing signature")
	}
	signature := *manifest.Signature
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
	data, err := archiveManifestSigningBytes(manifest)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), data, value) {
		return fmt.Errorf("verification failed")
	}
	return nil
}

func readArchiveEntries(archivePath string) (model.ArchiveManifest, map[string][]byte, error) {
	var manifest model.ArchiveManifest
	entries := make(map[string][]byte)
	err := walkArchive(archivePath, func(header *tar.Header, data []byte) error {
		switch header.Name {
		case archiveManifestName:
			if manifest.Version != "" {
				return fmt.Errorf("duplicate archive manifest")
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return err
			}
		default:
			if _, err := cleanArchivePath(header.Name); err != nil {
				return err
			}
			if _, ok := entries[header.Name]; ok {
				return fmt.Errorf("duplicate archive entry %s", header.Name)
			}
			entries[header.Name] = data
		}
		return nil
	})
	if err != nil {
		return model.ArchiveManifest{}, nil, err
	}
	if manifest.Version == "" {
		return model.ArchiveManifest{}, nil, fmt.Errorf("archive manifest missing")
	}
	if manifest.Version != "waystone.archive.v1" {
		return model.ArchiveManifest{}, nil, fmt.Errorf("unsupported archive manifest version %q", manifest.Version)
	}
	return manifest, entries, nil
}

func walkArchive(archivePath string, visit func(*tar.Header, []byte) error) error {
	return walkArchiveWithLimits(archivePath, defaultArchiveLimits(), visit)
}

type archiveLimits struct {
	MaxEntryBytes int64
	MaxEntries    int
	MaxTotalBytes int64
}

func defaultArchiveLimits() archiveLimits {
	return archiveLimits{
		MaxEntryBytes: maxArchiveEntryBytes,
		MaxEntries:    maxArchiveEntries,
		MaxTotalBytes: maxArchiveTotalBytes,
	}
}

func walkArchiveWithLimits(archivePath string, limits archiveLimits, visit func(*tar.Header, []byte) error) error {
	file, err := openUserSelectedFile(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder, err := zstd.NewReader(file)
	if err != nil {
		return err
	}
	defer decoder.Close()
	tr := tar.NewReader(decoder)
	var entries int
	var totalBytes int64
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("unsupported archive entry type for %s", header.Name)
		}
		if err := checkArchiveLimits(header, limits, &entries, &totalBytes); err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(tr, header.Size))
		if err != nil {
			return err
		}
		if int64(len(data)) != header.Size {
			return fmt.Errorf("archive entry %s size mismatch", header.Name)
		}
		if err := visit(header, data); err != nil {
			return err
		}
	}
}

func checkArchiveLimits(header *tar.Header, limits archiveLimits, entries *int, totalBytes *int64) error {
	*entries = *entries + 1
	if limits.MaxEntries > 0 && *entries > limits.MaxEntries {
		return fmt.Errorf("archive has too many entries")
	}
	if header.Size < 0 || limits.MaxEntryBytes > 0 && header.Size > limits.MaxEntryBytes {
		return fmt.Errorf("archive entry %s is too large", header.Name)
	}
	*totalBytes += header.Size
	if limits.MaxTotalBytes > 0 && *totalBytes > limits.MaxTotalBytes {
		return fmt.Errorf("archive total size exceeds limit")
	}
	return nil
}

func archiveManifestSigningBytes(manifest model.ArchiveManifest) ([]byte, error) {
	manifest.Signature = nil
	return canonicalJSON(manifest)
}
