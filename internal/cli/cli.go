// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
)

const defaultGitHubOAuthClientID = "Ov23liWNheWsFXT3BnPf"

const Version = "0.0.0-dev"

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("missing command")
	}

	switch args[0] {
	case "audit":
		return runAudit(args[1:], stdout, stderr)
	case "github":
		return runGitHub(ctx, args[1:], stdout, stderr)
	case "identity":
		return runIdentity(args[1:], stdout, stderr)
	case "issue":
		return runIssue(args[1:], stdout, stderr)
	case "label":
		return runLabel(args[1:], stdout, stderr)
	case "ledger":
		return runLedger(ctx, args[1:], stdout, stderr)
	case "migrate":
		return runMigrate(args[1:], stdout, stderr)
	case "milestone":
		return runMilestone(args[1:], stdout, stderr)
	case "pr":
		return runPullRequest(args[1:], stdout, stderr)
	case "source":
		return runSource(ctx, args[1:], stdout, stderr)
	case "version":
		return runVersion(args[1:], stdout)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}
