// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/steadytao/waystone/internal/model"
)

const (
	// Archive limits are defensive local resource bounds, not format-level compatibility limits.
	maxArchiveEntryBytes = 100 << 20
	maxArchiveEntries    = 10000
	maxArchiveTotalBytes = 512 << 20
)

func ExportArchive(root, archivePath string) error {
	if root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}
	manifest, err := buildArchiveManifest(root)
	if err != nil {
		return err
	}
	if err := signArchiveManifest(root, &manifest); err != nil {
		return err
	}
	manifestData, err := canonicalJSON(manifest)
	if err != nil {
		return err
	}
	file, err := os.Create(archivePath) // #nosec G304 -- archive output path is an explicit user-selected destination.
	if err != nil {
		return err
	}
	defer file.Close()

	encoder, err := zstd.NewWriter(file)
	if err != nil {
		return err
	}
	defer encoder.Close()

	tw := tar.NewWriter(encoder)
	defer tw.Close()

	for _, ref := range manifest.Files {
		filePath, err := safeRootedFilePath(root, ref.Path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(filePath)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("ledger path %s is a symlink", filePath)
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = ref.Path
		header.Mode = 0o644
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		in, err := openFileNoSymlink(filePath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, in)
		closeErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}

	header := &tar.Header{
		Name: archiveManifestName,
		Mode: 0o644,
		Size: int64(len(manifestData)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = tw.Write(manifestData)
	return err
}

func isPrivateIdentityKey(name string) bool {
	return strings.HasPrefix(filepath.ToSlash(name), "identities/") && strings.HasSuffix(name, ".key")
}

type ArchiveInspection struct {
	Format     string
	Files      int
	Sources    int
	Operations int
	Strict     bool
	Checksum   string
	Manifest   bool
	Signed     bool
}

func InspectArchive(archivePath string) (ArchiveInspection, error) {
	manifest, err := VerifyArchiveManifest(archivePath)
	if err != nil {
		return ArchiveInspection{}, err
	}
	staging, err := os.MkdirTemp("", "waystone-inspect-*")
	if err != nil {
		return ArchiveInspection{}, err
	}
	defer os.RemoveAll(staging)
	if err := extractArchive(archivePath, staging); err != nil {
		return ArchiveInspection{}, err
	}
	verification, err := (Reader{Root: staging}).Verify()
	if err != nil {
		return ArchiveInspection{}, err
	}
	operationVerification, err := (Reader{Root: staging}).VerifyOperations()
	strict := err == nil
	sources, sourceErr := (Reader{Root: staging}).Sources()
	if sourceErr != nil {
		return ArchiveInspection{}, sourceErr
	}
	return ArchiveInspection{
		Format:     "archive",
		Files:      verification.Files,
		Sources:    len(sources),
		Operations: operationVerification.Operations,
		Strict:     strict,
		Checksum:   verification.Checksum,
		Manifest:   true,
		Signed:     manifest.Signature != nil && manifest.Signature.Value != "",
	}, nil
}

func ExportSourceArchive(root, sourceSpec, archivePath string) error {
	source, err := ParseSourceSpec(sourceSpec)
	if err != nil {
		return err
	}
	if _, err := (Reader{Root: root}).VerifyOperations(); err != nil {
		return err
	}
	reader := Reader{Root: root}
	currentSource, err := reader.Source(source)
	if err != nil {
		return err
	}
	staging, err := os.MkdirTemp("", "waystone-source-export-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	project, err := reader.SourceProject(currentSource)
	if err != nil {
		return err
	}
	if err := (Writer{Root: staging}).writeLedgerMetadata(); err != nil {
		return err
	}
	projectPath := filepath.Join("projects", currentSource.System, currentSource.Owner, currentSource.Repo+".json")
	if err := writeJSONUnderRoot(staging, projectPath, project); err != nil {
		return err
	}
	now := time.Now().UTC()
	currentSource.Operations = []model.SourceOperationRef{{
		ID:        NewOperationID("ledger export source "+SourceSpec(currentSource), now),
		Command:   "ledger export --source",
		StartedAt: now,
	}}
	currentSource.Operations[0].Path = OperationPath(currentSource.Operations[0].ID)
	if err := writeJSONUnderRoot(staging, sourceManifestPath(currentSource), currentSource); err != nil {
		return err
	}
	for _, object := range currentSource.Objects {
		from, err := safeRootedFilePath(root, object.Path)
		if err != nil {
			return err
		}
		to, err := safeRootedWritePath(staging, object.Path)
		if err != nil {
			return err
		}
		if err := copyFile(from, to); err != nil {
			return err
		}
	}
	verification, err := (Reader{Root: staging}).Verify()
	if err != nil {
		return err
	}
	operation := model.Operation{
		ID:         currentSource.Operations[0].ID,
		Command:    "ledger export --source",
		Args:       []string{SourceSpec(currentSource)},
		StartedAt:  now,
		FinishedAt: now,
		Actor:      LocalActor("", "", false),
		Output:     model.OperationOutput{Ledger: staging, Unchanged: verification.Files},
		Changes:    verification.Changes,
	}
	if err := (Writer{Root: staging}).WriteOperation(operation); err != nil {
		return err
	}
	return ExportArchive(staging, archivePath)
}

func ImportArchive(archivePath, root string) error {
	if root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}
	if err := ensureEmptyOrMissing(root); err != nil {
		return err
	}
	if _, err := VerifyArchiveManifest(archivePath); err != nil {
		return err
	}

	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".waystone-import-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	if err := extractArchive(archivePath, staging); err != nil {
		return err
	}
	if _, err := (Reader{Root: staging}).Verify(); err != nil {
		return err
	}
	if _, err := (Reader{Root: staging}).VerifyOperations(); err != nil {
		return err
	}
	return os.Rename(staging, root)
}

func extractArchive(archivePath, root string) error {
	file, err := os.Open(archivePath) // #nosec G304 -- archive input path is an explicit user-selected file.
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
	limits := defaultArchiveLimits()
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
		if err := checkArchiveLimits(header, limits, &entries, &totalBytes); err != nil {
			return err
		}
		if header.Name == archiveManifestName {
			if _, err := io.CopyN(io.Discard, tr, header.Size); err != nil {
				return err
			}
			continue
		}
		target, err := safeRootedWritePath(root, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- target is derived from cleanArchivePath and rooted in staging.
			if err != nil {
				return err
			}
			if _, err := io.CopyN(out, tr, header.Size); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry type for %s", header.Name)
		}
	}
}

func copyFile(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return err
	}
	in, err := openFileNoSymlink(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to) // #nosec G304 -- destination path is scoped to temporary export staging.
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func rejectSymlinkEntry(path string, entry os.DirEntry) error {
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("ledger path %s is a symlink", path)
	}
	return nil
}

func readFileNoSymlink(path string) ([]byte, error) {
	file, err := openFileNoSymlink(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func openFileNoSymlink(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("ledger path %s is a symlink", path)
	}
	return os.Open(path) // #nosec G304 -- callers validate or discover paths under the configured ledger root.
}

func cleanArchivePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	for _, component := range strings.Split(name, "/") {
		if component == "" || component == "." || component == ".." {
			return "", fmt.Errorf("unsafe path %q", name)
		}
	}
	cleaned := path.Clean(name)
	if cleaned == "." ||
		cleaned == ".." ||
		strings.HasPrefix(cleaned, "../") ||
		strings.HasPrefix(cleaned, "/") ||
		strings.Contains(cleaned, "/../") ||
		strings.Contains(cleaned, ":") ||
		cleaned != name ||
		!filepath.IsLocal(filepath.FromSlash(cleaned)) {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	return cleaned, nil
}

func SafeRootedPath(root, name string) (string, error) {
	cleaned, err := cleanArchivePath(name)
	if err != nil {
		return "", err
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(absoluteRoot, filepath.FromSlash(cleaned))
	relative, err := filepath.Rel(absoluteRoot, target)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	return target, nil
}

func safeRootedFilePath(root, name string) (string, error) {
	target, err := SafeRootedPath(root, filepath.ToSlash(name))
	if err != nil {
		return "", err
	}
	if err := rejectRootedSymlinkDirs(root, filepath.Dir(target)); err != nil {
		return "", err
	}
	return target, nil
}

func safeRootedDirPath(root, name string) (string, error) {
	target, err := SafeRootedPath(root, filepath.ToSlash(name))
	if err != nil {
		return "", err
	}
	if err := rejectRootedSymlinkDirs(root, target); err != nil {
		return "", err
	}
	return target, nil
}

func safeRootedWritePath(root, name string) (string, error) {
	target, err := SafeRootedPath(root, filepath.ToSlash(name))
	if err != nil {
		return "", err
	}
	if err := ensureRootedDirNoSymlink(root, filepath.Dir(target)); err != nil {
		return "", err
	}
	return target, nil
}

func rejectRootedSymlinkDirs(root, dir string) error {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := checkRootDirectoryNoSymlink(absoluteRoot); err != nil {
		return err
	}
	relative, err := filepath.Rel(absoluteRoot, absoluteDir)
	if err != nil {
		return err
	}
	if relative == "." {
		return nil
	}
	current := absoluteRoot
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("ledger path %s is a symlink", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("ledger path %s is not a directory", current)
		}
	}
	return nil
}

func ensureRootedDirNoSymlink(root, dir string) error {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := ensureRootDirectoryNoSymlink(absoluteRoot); err != nil {
		return err
	}
	relative, err := filepath.Rel(absoluteRoot, absoluteDir)
	if err != nil {
		return err
	}
	if relative == "." {
		return nil
	}
	current := absoluteRoot
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("ledger path %s is a symlink", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("ledger path %s is not a directory", current)
		}
	}
	return nil
}

func checkRootDirectoryNoSymlink(root string) error {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("ledger path %s is a symlink", root)
	}
	if !info.IsDir() {
		return fmt.Errorf("ledger path %s is not a directory", root)
	}
	return nil
}

func ensureRootDirectoryNoSymlink(root string) error {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return err
		}
		info, err = os.Lstat(root)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("ledger path %s is a symlink", root)
	}
	if !info.IsDir() {
		return fmt.Errorf("ledger path %s is not a directory", root)
	}
	return nil
}

func ensureEmptyOrMissing(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("ledger %s already exists and is not empty", root)
	}
	return nil
}
