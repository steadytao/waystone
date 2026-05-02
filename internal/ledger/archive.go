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

const maxArchiveEntryBytes = 100 << 20

func ExportArchive(root, archivePath string) error {
	if root == "" {
		return fmt.Errorf("ledger root must not be empty")
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

	return filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = name
		header.Mode = 0o644
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		in, err := os.Open(filePath) // #nosec G304 -- filepath.WalkDir restricts exported files to the ledger root.
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tw, in)
		return err
	})
}

type ArchiveInspection struct {
	Format     string
	Files      int
	Sources    int
	Operations int
	Strict     bool
	Checksum   string
}

func InspectArchive(archivePath string) (ArchiveInspection, error) {
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
	}, nil
}

func ExportSourceArchive(root, sourceSpec, archivePath string) error {
	source, err := ParseSourceSpec(sourceSpec)
	if err != nil {
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
	if err := writeJSON(filepath.Join(staging, "projects", currentSource.System, currentSource.Owner, currentSource.Repo+".json"), project); err != nil {
		return err
	}
	now := time.Now().UTC()
	currentSource.Operations = []model.SourceOperationRef{{
		ID:        NewOperationID("ledger export source "+SourceSpec(currentSource), now),
		Command:   "ledger export --source",
		StartedAt: now,
	}}
	currentSource.Operations[0].Path = OperationPath(currentSource.Operations[0].ID)
	sourcePath := filepath.Join(staging, sourceManifestPath(currentSource))
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		return err
	}
	if err := writeJSON(sourcePath, currentSource); err != nil {
		return err
	}
	for _, object := range currentSource.Objects {
		from, err := SafeRootedPath(root, object.Path)
		if err != nil {
			return err
		}
		to, err := SafeRootedPath(staging, object.Path)
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
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := SafeRootedPath(root, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeReg:
			if header.Size < 0 || header.Size > maxArchiveEntryBytes {
				return fmt.Errorf("archive entry %s is too large", header.Name)
			}
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
	in, err := os.Open(from) // #nosec G304 -- source path comes from a verified source manifest object reference.
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

func cleanArchivePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	cleaned := path.Clean(name)
	if cleaned == "." ||
		cleaned == ".." ||
		strings.HasPrefix(cleaned, "../") ||
		strings.HasPrefix(cleaned, "/") ||
		strings.Contains(cleaned, "/../") ||
		strings.Contains(cleaned, ":") ||
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
