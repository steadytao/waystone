// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	signature, err := w.operationSignature(operation)
	if err != nil {
		return err
	}
	operation.Signature = signature
	dir := filepath.Join(w.Root, "operations")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return writeJSON(filepath.Join(w.Root, operationRelativePath(operation.ID)), operation)
}

func OperationHash(operation model.Operation) (string, error) {
	operation.OperationHash = ""
	operation.Signature = nil
	data, err := canonicalOperationJSON(operation)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (w Writer) operationSignature(operation model.Operation) (*model.OperationSignature, error) {
	identity, err := DefaultIdentity(w.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	privateKey, err := defaultPrivateKey(w.Root)
	if err != nil {
		return nil, err
	}
	data, err := operationSigningBytes(operation)
	if err != nil {
		return nil, err
	}
	signature, err := w.signPayload(identity, privateKey, data)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (w Writer) signPayload(identity model.Identity, privateKey ed25519.PrivateKey, data []byte) (*model.OperationSignature, error) {
	return &model.OperationSignature{
		Algorithm:  identity.Algorithm,
		IdentityID: identity.ID,
		PublicKey:  identity.PublicKey,
		Value:      base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, data)),
	}, nil
}

func operationSigningBytes(operation model.Operation) ([]byte, error) {
	operation.OperationHash = ""
	operation.Signature = nil
	return canonicalOperationJSON(operation)
}

func canonicalOperationJSON(operation model.Operation) ([]byte, error) {
	value := struct {
		ID                string                `json:"id"`
		PreviousOperation string                `json:"previous_operation,omitempty"`
		OperationHash     string                `json:"operation_hash,omitempty"`
		Command           string                `json:"command"`
		Args              []string              `json:"args,omitempty"`
		StartedAt         time.Time             `json:"started_at"`
		FinishedAt        time.Time             `json:"finished_at"`
		Actor             model.OperationActor  `json:"actor"`
		Auth              model.OperationAuth   `json:"auth,omitempty"`
		Input             map[string]string     `json:"input,omitempty"`
		Output            model.OperationOutput `json:"output"`
		Changes           []model.ObjectChange  `json:"changes,omitempty"`
	}{
		ID:                operation.ID,
		PreviousOperation: operation.PreviousOperation,
		OperationHash:     operation.OperationHash,
		Command:           operation.Command,
		Args:              operation.Args,
		StartedAt:         operation.StartedAt,
		FinishedAt:        operation.FinishedAt,
		Actor:             operation.Actor,
		Auth:              operation.Auth,
		Input:             operation.Input,
		Output:            operation.Output,
		Changes:           operation.Changes,
	}
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
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
