// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func runIdentity(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printIdentityUsage(stderr)
		return errors.New("missing identity command")
	}
	switch args[0] {
	case "init":
		return runIdentityInit(args[1:], stdout)
	case "list":
		return runIdentityList(args[1:], stdout)
	case "show":
		return runIdentityShow(args[1:], stdout)
	case "status":
		return runIdentityStatus(args[1:], stdout)
	case "trust":
		return runIdentityTrust(args[1:], stdout)
	case "untrust":
		return runIdentityUntrust(args[1:], stdout)
	default:
		printIdentityUsage(stderr)
		return fmt.Errorf("unknown identity command %q", args[0])
	}
}

func runIdentityInit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone identity init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	name := fs.String("name", "", "identity display name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone identity init [flags]")
	}
	startedAt := time.Now().UTC()
	identity, err := ledger.CreateDefaultIdentity(*root, *name)
	if err != nil {
		return err
	}
	verification, err := (ledger.Reader{Root: *root}).Verify()
	if err != nil {
		return err
	}
	finishedAt := time.Now().UTC()
	operation := model.Operation{
		ID:         ledger.NewOperationID("identity init", startedAt),
		Command:    "identity init",
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), false),
		Output: model.OperationOutput{
			Ledger:    *root,
			Unchanged: verification.Files,
		},
		Changes: verification.Changes,
	}
	if err := (ledger.Writer{Root: *root}).WriteOperation(operation); err != nil {
		return err
	}
	writeField(stdout, "Identity", identity.ID)
	writeField(stdout, "Algorithm", identity.Algorithm)
	writeField(stdout, "Public key", identity.PublicKey)
	writeField(stdout, "Operation", operation.ID)
	return nil
}

func runIdentityShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone identity show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone identity show [flags]")
	}
	identity, err := ledger.DefaultIdentity(*root)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, identity)
	}
	writeField(stdout, "Identity", identity.ID)
	if identity.Name != "" {
		writeField(stdout, "Name", identity.Name)
	}
	writeField(stdout, "Algorithm", identity.Algorithm)
	writeField(stdout, "Public key", identity.PublicKey)
	return nil
}

func runIdentityList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone identity list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone identity list [flags]")
	}
	reader := ledger.Reader{Root: *root}
	identities, err := reader.Identities()
	if err != nil {
		return err
	}
	policy, err := reader.TrustPolicy()
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, map[string]any{
			"identities": identities,
			"trust":      policy,
		})
	}
	fmt.Fprintf(stdout, "%-16s %-10s %-10s %s\n", "IDENTITY", "TRUST", "ALGORITHM", "NAME")
	for _, identity := range identities {
		trust := "untrusted"
		if policy.Trusts(identity.ID) {
			trust = "trusted"
		}
		fmt.Fprintf(stdout, "%-16s %-10s %-10s %s\n", identity.ID, trust, identity.Algorithm, identity.Name)
	}
	return nil
}

func runIdentityTrust(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone identity trust", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone identity trust [flags] <identity-id>")
	}
	startedAt := time.Now().UTC()
	if err := (ledger.Writer{Root: *root}).TrustIdentity(fs.Arg(0)); err != nil {
		return err
	}
	operation, err := writeIdentityTrustOperation(*root, "identity trust", fs.Arg(0), startedAt, args)
	if err != nil {
		return err
	}
	writeField(stdout, "Identity", fs.Arg(0))
	writeField(stdout, "Trust", "trusted")
	writeField(stdout, "Operation", operation.ID)
	return nil
}

func runIdentityUntrust(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone identity untrust", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone identity untrust [flags] <identity-id>")
	}
	startedAt := time.Now().UTC()
	if err := (ledger.Writer{Root: *root}).UntrustIdentity(fs.Arg(0)); err != nil {
		return err
	}
	operation, err := writeIdentityTrustOperation(*root, "identity untrust", fs.Arg(0), startedAt, args)
	if err != nil {
		return err
	}
	writeField(stdout, "Identity", fs.Arg(0))
	writeField(stdout, "Trust", "untrusted")
	writeField(stdout, "Operation", operation.ID)
	return nil
}

func writeIdentityTrustOperation(root, command, identityID string, startedAt time.Time, args []string) (model.Operation, error) {
	verification, err := (ledger.Reader{Root: root}).Verify()
	if err != nil {
		return model.Operation{}, err
	}
	finishedAt := time.Now().UTC()
	operation := model.Operation{
		ID:         ledger.NewOperationID(command, startedAt),
		Command:    command,
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), false),
		Input: map[string]string{
			"identity": identityID,
		},
		Output: model.OperationOutput{
			Ledger:    root,
			Unchanged: verification.Files,
		},
		Changes: verification.Changes,
	}
	if err := (ledger.Writer{Root: root}).WriteOperation(operation); err != nil {
		return model.Operation{}, err
	}
	return operation, nil
}

func runIdentityStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone identity status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone identity status [flags]")
	}
	reader := ledger.Reader{Root: *root}
	identities, err := reader.Identities()
	if err != nil {
		return err
	}
	policy, err := reader.TrustPolicy()
	if err != nil {
		return err
	}
	trusted := 0
	for _, identity := range identities {
		if policy.Trusts(identity.ID) {
			trusted++
		}
	}
	status := map[string]any{
		"identities": len(identities),
		"trusted":    trusted,
		"untrusted":  len(identities) - trusted,
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, status)
	}
	writeField(stdout, "Identities", len(identities))
	writeField(stdout, "Trusted", trusted)
	writeField(stdout, "Untrusted", len(identities)-trusted)
	return nil
}
