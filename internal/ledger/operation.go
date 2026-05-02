// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

func (w Writer) WriteOperation(operation model.Operation) error {
	if w.Root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}
	if operation.ID == "" {
		return fmt.Errorf("operation ID must not be empty")
	}
	previous, err := (Reader(w)).LastOperation()
	if err != nil {
		return err
	}
	if previous.ID != "" {
		operation.PreviousOperation = previous.ID
	}
	hash, err := OperationHash(operation)
	if err != nil {
		return err
	}
	operation.OperationHash = hash
	dir := filepath.Join(w.Root, "operations")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return writeJSON(filepath.Join(w.Root, operationRelativePath(operation.ID)), operation)
}

func OperationHash(operation model.Operation) (string, error) {
	operation.OperationHash = ""
	data, err := canonicalOperationJSON(operation)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func canonicalOperationJSON(operation model.Operation) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(operation); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (r Reader) Operations() ([]model.Operation, error) {
	operations, err := readDirJSON[model.Operation](filepath.Join(r.Root, "operations"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].StartedAt.Before(operations[j].StartedAt)
	})
	return operations, nil
}

func (r Reader) Operation(id string) (model.Operation, error) {
	operations, err := r.Operations()
	if err != nil {
		return model.Operation{}, err
	}
	for _, operation := range operations {
		if operation.ID == id || strings.TrimSuffix(namedFile(operation.ID), ".json") == id {
			return operation, nil
		}
	}
	return model.Operation{}, fmt.Errorf("operation %q not found", id)
}

func operationRelativePath(id string) string {
	return filepath.Join("operations", namedFile(id))
}

func OperationPath(id string) string {
	return filepath.ToSlash(operationRelativePath(id))
}

func (r Reader) LastOperation() (model.Operation, error) {
	operations, err := r.Operations()
	if err != nil {
		return model.Operation{}, err
	}
	if len(operations) == 0 {
		return model.Operation{}, nil
	}
	return operations[len(operations)-1], nil
}

func NewOperationID(command string, startedAt time.Time) string {
	normalized := safeName(command)
	timestamp := startedAt.UTC().Format("20060102T150405.000000000Z")
	return strings.Trim(normalized+"-"+timestamp, "-")
}

func LocalActor(gitName, gitEmail string, includeMachine bool) model.OperationActor {
	actor := model.OperationActor{
		Source:       "local",
		GitUserName:  gitName,
		GitUserEmail: gitEmail,
	}
	if includeMachine {
		if user := os.Getenv("USER"); user != "" {
			actor.User = user
		} else if user := os.Getenv("USERNAME"); user != "" {
			actor.User = user
		}
		if hostname, err := os.Hostname(); err == nil {
			actor.Hostname = hostname
		}
	}
	return actor
}
