// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type ArchiveManifest struct {
	Version         string              `json:"version"`
	CreatedAt       time.Time           `json:"created_at"`
	Format          string              `json:"format"`
	Files           []ArchiveFileRef    `json:"files"`
	Sources         []Source            `json:"sources,omitempty"`
	LedgerChecksum  string              `json:"ledger_checksum,omitempty"`
	Operations      int                 `json:"operations"`
	OperationHead   string              `json:"operation_head,omitempty"`
	OperationHash   string              `json:"operation_hash,omitempty"`
	OperationSHA256 string              `json:"operation_sha256,omitempty"`
	Signature       *OperationSignature `json:"signature,omitempty"`
}

type ArchiveFileRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
