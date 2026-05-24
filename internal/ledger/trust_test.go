// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

func TestCreateDefaultIdentityTrustsIdentity(t *testing.T) {
	root := writeTestLedger(t)
	identity, err := CreateDefaultIdentity(root, "test")
	if err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}

	policy, err := (Reader{Root: root}).TrustPolicy()
	if err != nil {
		t.Fatalf("TrustPolicy returned error: %v", err)
	}
	if !policy.Trusts(identity.ID) {
		t.Fatalf("trust policy = %#v, want trusted identity %s", policy, identity.ID)
	}
}

func TestTrustAndUntrustIdentity(t *testing.T) {
	root := writeTestLedger(t)
	identity, err := CreateDefaultIdentity(root, "test")
	if err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}
	if err := (Writer{Root: root}).UntrustIdentity(identity.ID); err != nil {
		t.Fatalf("UntrustIdentity returned error: %v", err)
	}
	policy, err := (Reader{Root: root}).TrustPolicy()
	if err != nil {
		t.Fatalf("TrustPolicy returned error: %v", err)
	}
	if policy.Trusts(identity.ID) {
		t.Fatalf("identity %s is still trusted", identity.ID)
	}
	if err := (Writer{Root: root}).TrustIdentity(identity.ID); err != nil {
		t.Fatalf("TrustIdentity returned error: %v", err)
	}
	policy, err = (Reader{Root: root}).TrustPolicy()
	if err != nil {
		t.Fatalf("TrustPolicy returned error: %v", err)
	}
	if !policy.Trusts(identity.ID) {
		t.Fatalf("identity %s is not trusted", identity.ID)
	}
}

func TestVerifyOperationSignaturesReportsTrust(t *testing.T) {
	root := writeTestLedger(t)
	identity, err := CreateDefaultIdentity(root, "test")
	if err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}
	operation := model.Operation{
		ID:         NewOperationID("github import", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "github import",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root},
	}
	verification, err := (Reader{Root: root}).Verify()
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	operation.Changes = verification.Changes
	if err := (Writer{Root: root}).WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}

	signatureVerification, err := (Reader{Root: root}).VerifyOperationSignatures()
	if err != nil {
		t.Fatalf("VerifyOperationSignatures returned error: %v", err)
	}
	if signatureVerification.Trusted == 0 || signatureVerification.Untrusted != 0 {
		t.Fatalf("signature verification = %#v, want trusted signatures", signatureVerification)
	}

	if err := (Writer{Root: root}).UntrustIdentity(identity.ID); err != nil {
		t.Fatalf("UntrustIdentity returned error: %v", err)
	}
	signatureVerification, err = (Reader{Root: root}).VerifyOperationSignatures()
	if err != nil {
		t.Fatalf("VerifyOperationSignatures returned error: %v", err)
	}
	if signatureVerification.Untrusted == 0 {
		t.Fatalf("signature verification = %#v, want untrusted signatures", signatureVerification)
	}
}

func TestVerifyOperationSignaturesRejectsTrustedIDWithDifferentKey(t *testing.T) {
	root := writeTestLedger(t)
	trusted, err := CreateDefaultIdentity(root, "trusted")
	if err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}
	operation := model.Operation{
		ID:         NewOperationID("github import", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Command:    "github import",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Output:     model.OperationOutput{Ledger: root},
	}
	if err := (Writer{Root: root}).WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}
	operations, err := (Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	operation = operations[0]

	attackerPublic, attackerPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	data, err := operationSigningBytes(operation)
	if err != nil {
		t.Fatalf("operationSigningBytes returned error: %v", err)
	}
	signature, err := (Writer{Root: root}).signPayload(model.Identity{
		ID:        trusted.ID,
		Algorithm: identityAlgorithmEd25519,
		PublicKey: base64.StdEncoding.EncodeToString(attackerPublic),
	}, attackerPrivate, data)
	if err != nil {
		t.Fatalf("signPayload returned error: %v", err)
	}
	operation.Signature = signature
	if err := writeJSON(filepath.Join(root, operationRelativePath(operation.ID)), operation); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}

	_, err = (Reader{Root: root}).VerifyOperationSignatures()
	if err == nil || !strings.Contains(err.Error(), "public key does not match trusted identity") {
		t.Fatalf("VerifyOperationSignatures error = %v, want trusted identity key mismatch", err)
	}
}

func TestVerifySourceSignaturesRejectsTrustedIDWithDifferentKey(t *testing.T) {
	root := t.TempDir()
	trusted, err := CreateDefaultIdentity(root, "trusted")
	if err != nil {
		t.Fatalf("CreateDefaultIdentity returned error: %v", err)
	}
	imported := model.GitHubImport{
		Project: model.Project{ID: "github:repo:1", Name: "example/project"},
		Source:  model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		Issues:  []model.Issue{{ID: "github:issue:1", Number: 1, Title: "issue"}},
	}
	if err := (Writer{Root: root}).WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}
	source, err := (Reader{Root: root}).Source(imported.Source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}

	attackerPublic, attackerPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	data, err := sourceSigningBytes(source)
	if err != nil {
		t.Fatalf("sourceSigningBytes returned error: %v", err)
	}
	signature, err := (Writer{Root: root}).signPayload(model.Identity{
		ID:        trusted.ID,
		Algorithm: identityAlgorithmEd25519,
		PublicKey: base64.StdEncoding.EncodeToString(attackerPublic),
	}, attackerPrivate, data)
	if err != nil {
		t.Fatalf("signPayload returned error: %v", err)
	}
	source.Signature = signature
	if err := writeJSON(filepath.Join(root, sourceManifestPath(source)), source); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}

	_, err = (Reader{Root: root}).VerifySourceSignatures()
	if err == nil || !strings.Contains(err.Error(), "public key does not match trusted identity") {
		t.Fatalf("VerifySourceSignatures error = %v, want trusted identity key mismatch", err)
	}
}
