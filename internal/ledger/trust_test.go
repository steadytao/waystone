// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
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
