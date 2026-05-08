// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/steadytao/waystone/internal/gitlab"
	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func runGitLab(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printGitLabUsage(stderr)
		return errors.New("missing gitlab command")
	}
	switch args[0] {
	case "import":
		return runGitLabImport(ctx, args[1:], stdout)
	default:
		printGitLabUsage(stderr)
		return fmt.Errorf("unknown gitlab command %q", args[0])
	}
}

func runGitLabImport(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone gitlab import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outDir := fs.String("out", ".waystone", "directory to write Waystone ledger data")
	tokenEnv := fs.String("token-env", "GITLAB_TOKEN", "environment variable containing a GitLab token")
	apiBase := fs.String("api-base", "https://gitlab.com/api/v4", "GitLab API base URL")
	concurrency := fs.Int("concurrency", 8, "maximum concurrent GitLab detail requests")
	timeout := fs.Duration("timeout", 2*time.Minute, "request timeout")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")
	normalizedArgs, err := normalizeSingleValueCommandArgs(args, "project", map[string]bool{"--local": true, "-local": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone gitlab import [--out <dir>] [--api-base <url>] <group/project>")
	}
	owner, repo, err := parseRepo(fs.Arg(0))
	if err != nil {
		return err
	}
	token := tokenFromEnvironment(*tokenEnv)
	if token == "" {
		return fmt.Errorf("%s must be set for GitLab import", *tokenEnv)
	}
	authMode := "environment"
	startedAt := time.Now().UTC()
	operationID := ledger.NewOperationID("gitlab import", startedAt)
	writeField(stdout, "Project", owner+"/"+repo)
	writeField(stdout, "Ledger", *outDir)
	writeField(stdout, "Auth", "authenticated")
	fmt.Fprintln(stdout)

	client := gitlab.NewClient(*apiBase, token, *timeout, *concurrency)
	imported, err := client.ImportProject(ctx, owner, repo)
	if err != nil {
		return err
	}
	if existingSource, err := (ledger.Reader{Root: *outDir}).Source(imported.Source); err == nil {
		imported.Source.Operations = existingSource.Operations
	}
	imported.Source.Operations = append(imported.Source.Operations, model.SourceOperationRef{
		ID:        operationID,
		Command:   "gitlab import",
		Path:      ledger.OperationPath(operationID),
		StartedAt: startedAt,
	})
	writer := ledger.Writer{Root: *outDir}
	diff, err := writer.DiffForgeImport(imported)
	if err != nil {
		return err
	}
	ledgerExisted := fileExists(filepath.Join(*outDir, "ledger.json"))
	if err := writer.WriteForgeImport(imported); err != nil {
		return err
	}
	if err := addLedgerMetadataChange(&diff, *outDir, ledgerExisted); err != nil {
		return err
	}
	finishedAt := time.Now().UTC()
	operation := model.Operation{
		ID:         operationID,
		Command:    "gitlab import",
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), *includeLocal),
		Auth:       model.OperationAuth{Provider: "gitlab", Mode: authMode},
		Input:      map[string]string{"source": "gitlab:" + owner + "/" + repo},
		Output: model.OperationOutput{
			Ledger:    *outDir,
			Created:   diff.Created,
			Updated:   diff.Updated,
			Deleted:   diff.Deleted,
			Unchanged: diff.Unchanged,
			Summary: model.RecordSummary{
				Issues:       len(imported.Issues),
				Comments:     len(imported.Comments),
				PullRequests: len(imported.PullRequests),
				Labels:       len(imported.Labels),
				Milestones:   len(imported.Milestones),
				Releases:     len(imported.Releases),
			},
		},
		Changes: diff.Changes,
	}
	if err := writer.WriteOperation(operation); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Import complete")
	writeIndentedField(stdout, "Operation", operation.ID)
	writeIndentedField(stdout, "Created", diff.Created)
	writeIndentedField(stdout, "Updated", diff.Updated)
	writeIndentedField(stdout, "Deleted", diff.Deleted)
	writeIndentedField(stdout, "Unchanged", diff.Unchanged)
	fmt.Fprintln(stdout)
	writeIndentedField(stdout, "Issues", len(imported.Issues))
	writeIndentedField(stdout, "Comments", len(imported.Comments))
	writeIndentedField(stdout, "Pull requests", len(imported.PullRequests))
	writeIndentedField(stdout, "Labels", len(imported.Labels))
	writeIndentedField(stdout, "Milestones", len(imported.Milestones))
	writeIndentedField(stdout, "Releases", len(imported.Releases))
	return nil
}

func tokenFromEnvironment(tokenEnv string) string {
	if tokenEnv == "" {
		return ""
	}
	return os.Getenv(tokenEnv)
}
