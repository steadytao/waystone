// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/steadytao/waystone/internal/model"
)

type Diff struct {
	Changes   []model.ObjectChange
	Created   int
	Updated   int
	Deleted   int
	Unchanged int
}

func (w Writer) DiffGitHubImport(imported model.GitHubImport) (Diff, error) {
	var diff Diff
	for _, target := range gitHubImportTargets(imported) {
		changeType, err := w.diffTarget(target)
		if err != nil {
			return Diff{}, err
		}
		switch changeType {
		case "created":
			diff.Created++
		case "updated":
			diff.Updated++
		case "unchanged":
			diff.Unchanged++
		}
		if changeType == "unchanged" {
			changeType = "verified"
		}
		objectHash, err := w.targetHash(target)
		if err != nil {
			return Diff{}, err
		}
		diff.Changes = append(diff.Changes, model.ObjectChange{
			Type:   changeType,
			Object: target.object,
			Number: target.number,
			ID:     target.id,
			Path:   filepath.ToSlash(target.relative),
			SHA256: objectHash,
		})
	}
	return diff, nil
}

func (w Writer) DiffGitHubAudit(audit model.GitHubAudit) (Diff, error) {
	var diff Diff
	for _, target := range gitHubAuditTargets(audit) {
		changeType, err := w.diffTarget(target)
		if err != nil {
			return Diff{}, err
		}
		switch changeType {
		case "created":
			diff.Created++
		case "updated":
			diff.Updated++
		case "unchanged":
			diff.Unchanged++
		}
		if changeType == "unchanged" {
			changeType = "verified"
		}
		objectHash, err := w.targetHash(target)
		if err != nil {
			return Diff{}, err
		}
		diff.Changes = append(diff.Changes, model.ObjectChange{
			Type:   changeType,
			Object: target.object,
			ID:     target.id,
			Path:   filepath.ToSlash(target.relative),
			SHA256: objectHash,
		})
	}
	return diff, nil
}

func (w Writer) targetHash(target writeTarget) (string, error) {
	next, err := canonicalJSON(target.value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(next)
	return hex.EncodeToString(sum[:]), nil
}

func (w Writer) diffTarget(target writeTarget) (string, error) {
	next, err := canonicalJSON(target.value)
	if err != nil {
		return "", err
	}

	current, err := os.ReadFile(filepath.Join(w.Root, target.relative))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "created", nil
		}
		return "", err
	}
	if bytes.Equal(bytes.TrimSpace(current), bytes.TrimSpace(next)) {
		return "unchanged", nil
	}
	return "updated", nil
}

func canonicalJSON(value any) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
