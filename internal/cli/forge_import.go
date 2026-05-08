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

	"github.com/steadytao/waystone/internal/forgeapi"
	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func runForgejo(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printForgejoUsage(stderr)
		return errors.New("missing forgejo command")
	}
	switch args[0] {
	case "import":
		return runForgeImport(ctx, args[1:], stdout, forgeImportOptions{
			Command:     "forgejo import",
			System:      "forgejo",
			Provider:    "forgejo",
			DefaultBase: "https://codeberg.org/api/v1",
			TokenNames:  []string{"FORGEJO_TOKEN"},
		})
	default:
		printForgejoUsage(stderr)
		return fmt.Errorf("unknown forgejo command %q", args[0])
	}
}

func runGitea(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printGiteaUsage(stderr)
		return errors.New("missing gitea command")
	}
	switch args[0] {
	case "import":
		return runForgeImport(ctx, args[1:], stdout, forgeImportOptions{
			Command:     "gitea import",
			System:      "gitea",
			Provider:    "gitea",
			DefaultBase: "https://gitea.com/api/v1",
			TokenNames:  []string{"GITEA_TOKEN"},
		})
	default:
		printGiteaUsage(stderr)
		return fmt.Errorf("unknown gitea command %q", args[0])
	}
}

type forgeImportOptions struct {
	Command     string
	System      string
	Provider    string
	DefaultBase string
	TokenNames  []string
}

func runForgeImport(ctx context.Context, args []string, stdout io.Writer, options forgeImportOptions) error {
	fs := flag.NewFlagSet("waystone "+options.Command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outDir := fs.String("out", ".waystone", "directory to write Waystone ledger data")
	tokenEnv := fs.String("token-env", "", "environment variable containing a token")
	apiBase := fs.String("api-base", options.DefaultBase, "API base URL")
	concurrency := fs.Int("concurrency", 8, "maximum concurrent detail requests")
	timeout := fs.Duration("timeout", 2*time.Minute, "request timeout")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")
	normalizedArgs, err := normalizeSingleValueCommandArgs(args, "repository", map[string]bool{"--local": true, "-local": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: waystone %s [flags] owner/repo", options.Command)
	}
	owner, repo, err := parseRepo(fs.Arg(0))
	if err != nil {
		return err
	}
	token, authMode := forgeToken(*tokenEnv, options.TokenNames)
	startedAt := time.Now().UTC()
	operationID := ledger.NewOperationID(options.Command, startedAt)
	writeField(stdout, "Project", owner+"/"+repo)
	writeField(stdout, "Ledger", *outDir)
	if token == "" {
		writeField(stdout, "Auth", "unauthenticated")
	} else {
		writeField(stdout, "Auth", "authenticated")
	}
	fmt.Fprintln(stdout)

	client := forgeapi.NewClient(*apiBase, options.System, token, *timeout, *concurrency)
	imported, err := client.ImportRepository(ctx, owner, repo)
	if err != nil {
		return err
	}
	if existingSource, err := (ledger.Reader{Root: *outDir}).Source(imported.Source); err == nil {
		imported.Source.Operations = existingSource.Operations
	}
	imported.Source.Operations = append(imported.Source.Operations, model.SourceOperationRef{
		ID:        operationID,
		Command:   options.Command,
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
		Command:    options.Command,
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), *includeLocal),
		Auth:       model.OperationAuth{Provider: options.Provider, Mode: authMode},
		Input:      map[string]string{"source": options.System + ":" + owner + "/" + repo},
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

func forgeToken(tokenEnv string, tokenNames []string) (string, string) {
	if tokenEnv != "" {
		if token := os.Getenv(tokenEnv); token != "" {
			return token, "environment"
		}
		return "", "none"
	}
	for _, name := range tokenNames {
		if token := os.Getenv(name); token != "" {
			return token, "environment"
		}
	}
	return "", "none"
}
