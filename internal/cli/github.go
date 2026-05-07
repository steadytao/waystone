// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/steadytao/waystone/internal/auth"
	"github.com/steadytao/waystone/internal/github"
	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func runGitHubAuth(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printGitHubAuthUsage(stderr)
		return errors.New("missing github auth command")
	}

	switch args[0] {
	case "login":
		return runGitHubAuthLogin(ctx, args[1:], stdout)
	case "logout":
		return runGitHubAuthLogout(args[1:], stdout)
	default:
		printGitHubAuthUsage(stderr)
		return fmt.Errorf("unknown github auth command %q", args[0])
	}
}

func runGitHubAuthLogin(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone github auth login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	clientID := fs.String("client-id", defaultGitHubClientID(), "GitHub OAuth app client ID")
	scope := fs.String("scope", "", "GitHub OAuth scope")
	timeout := fs.Duration("timeout", 15*time.Minute, "login timeout")
	plainFileStore := fs.Bool("plain-file-store", false, "store token in a plaintext local file instead of the OS credential store")

	if err := fs.Parse(args); err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	code, err := github.RequestDeviceCode(ctx, httpClient, *clientID, *scope)
	if err != nil {
		return err
	}

	writeField(stdout, "URL", code.VerificationURI)
	writeField(stdout, "Code", code.UserCode)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Waiting for GitHub authorization...")

	loginCtx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	token, err := github.PollDeviceToken(loginCtx, httpClient, *clientID, code)
	if err != nil {
		return err
	}

	store, err := credentialStore(*plainFileStore)
	if err != nil {
		return err
	}
	if err := store.SaveGitHubToken(token); err != nil {
		return err
	}

	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Stored GitHub token in %s\n", store.Description())
	return nil
}

func runGitHubAuthLogout(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone github auth logout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	plainFileStore := fs.Bool("plain-file-store", false, "delete token from the plaintext local file instead of the OS credential store")

	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := credentialStore(*plainFileStore)
	if err != nil {
		return err
	}
	if err := store.DeleteGitHubToken(); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Deleted GitHub token from %s\n", store.Description())
	return nil
}

func runGitHub(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printGitHubUsage(stderr)
		return errors.New("missing github command")
	}

	switch args[0] {
	case "audit":
		return runGitHubAudit(ctx, args[1:], stdout)
	case "auth":
		return runGitHubAuth(ctx, args[1:], stdout, stderr)
	case "import":
		return runGitHubImport(ctx, args[1:], stdout, "github import")
	case "refresh":
		return runGitHubImport(ctx, args[1:], stdout, "github refresh")
	default:
		printGitHubUsage(stderr)
		return fmt.Errorf("unknown github command %q", args[0])
	}
}

func runGitHubAudit(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone github audit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable containing a GitHub token")
	apiBase := fs.String("api-base", "https://api.github.com", "GitHub API base URL")
	timeout := fs.Duration("timeout", 2*time.Minute, "request timeout")
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	plainFileStore := fs.Bool("plain-file-store", false, "read stored token from a plaintext local file instead of the OS credential store")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	verbose := fs.Bool("verbose", false, "show detailed audit evidence")
	verboseShort := fs.Bool("v", false, "show detailed audit evidence")
	noWrite := fs.Bool("no-write", false, "print the audit without writing it to the ledger")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")

	normalizedArgs, err := normalizeSingleValueCommandArgs(args, "repository", map[string]bool{
		"--json":             true,
		"-json":              true,
		"--verbose":          true,
		"-verbose":           true,
		"--v":                true,
		"-v":                 true,
		"--plain-file-store": true,
		"-plain-file-store":  true,
		"--no-write":         true,
		"-no-write":          true,
		"--local":            true,
		"-local":             true,
	})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone github audit [flags] owner/repo")
	}
	owner, repo, err := parseRepo(fs.Arg(0))
	if err != nil {
		return err
	}
	token := githubTokenFromEnvironment(*tokenEnv)
	authMode := "none"
	if token == "" {
		store, err := credentialStore(*plainFileStore)
		if err == nil {
			stored, err := store.GitHubToken()
			if err == nil {
				token = stored.AccessToken
				authMode = "stored"
			}
		}
	} else {
		authMode = "environment"
	}
	startedAt := time.Now().UTC()
	operationID := ledger.NewOperationID("github audit", startedAt)
	client := github.NewClient(*apiBase, token, *timeout)
	authLogin := ""
	if token != "" {
		login, err := client.AuthenticatedUser(ctx)
		if err != nil {
			return err
		}
		authLogin = login
	}
	audit, err := client.AuditRepository(ctx, owner, repo)
	if err != nil {
		return err
	}
	audit.ID = operationID
	if !*noWrite {
		writer := ledger.Writer{Root: *root}
		recordedAudit := audit
		if existingSource, err := (ledger.Reader{Root: *root}).Source(audit.Source); err == nil {
			recordedAudit.Source.Objects = existingSource.Objects
			recordedAudit.Source.Operations = existingSource.Operations
		}
		recordedAudit.Source.Operations = append(recordedAudit.Source.Operations, model.SourceOperationRef{
			ID:        operationID,
			Command:   "github audit",
			Path:      ledger.OperationPath(operationID),
			StartedAt: startedAt,
		})
		diff, err := writer.DiffGitHubAudit(recordedAudit)
		if err != nil {
			return err
		}
		ledgerExisted := fileExists(filepath.Join(*root, "ledger.json"))
		if err := writer.WriteGitHubAudit(recordedAudit); err != nil {
			return err
		}
		if err := addLedgerMetadataChange(&diff, *root, ledgerExisted); err != nil {
			return err
		}
		finishedAt := time.Now().UTC()
		operation := model.Operation{
			ID:         operationID,
			Command:    "github audit",
			Args:       append([]string(nil), args...),
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
			Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), *includeLocal),
			Auth:       model.OperationAuth{Provider: "github", Mode: authMode, Login: authLogin},
			Input:      map[string]string{"source": owner + "/" + repo},
			Output:     model.OperationOutput{Ledger: *root, Created: diff.Created, Updated: diff.Updated, Unchanged: diff.Unchanged},
			Changes:    diff.Changes,
		}
		if err := writer.WriteOperation(operation); err != nil {
			return err
		}
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, audit)
	}
	writeGitHubAudit(stdout, audit, *verbose || *verboseShort)
	if !*noWrite {
		fmt.Fprintln(stdout)
		writeField(stdout, "Operation", operationID)
		writeField(stdout, "Ledger", *root)
	}
	return nil
}

func runGitHubImport(ctx context.Context, args []string, stdout io.Writer, command string) error {
	fs := flag.NewFlagSet("waystone "+command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	outDir := fs.String("out", ".waystone", "directory to write Waystone ledger data")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable containing a GitHub token")
	apiBase := fs.String("api-base", "https://api.github.com", "GitHub API base URL")
	timeout := fs.Duration("timeout", 2*time.Minute, "request timeout")
	concurrency := fs.Int("concurrency", 6, "maximum concurrent GitHub detail requests")
	plainFileStore := fs.Bool("plain-file-store", false, "read stored token from a plaintext local file instead of the OS credential store")
	verbose := fs.Bool("verbose", false, "show detailed import progress")
	verboseShort := fs.Bool("v", false, "show detailed import progress")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")

	normalizedArgs, err := normalizeImportArgs(args)
	if err != nil {
		return err
	}

	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: waystone %s [flags] owner/repo", command)
	}

	owner, repo, err := parseRepo(fs.Arg(0))
	if err != nil {
		return err
	}
	options := githubImportOptions{
		OutDir:         *outDir,
		TokenEnv:       *tokenEnv,
		APIBase:        *apiBase,
		Timeout:        *timeout,
		Concurrency:    *concurrency,
		PlainFileStore: *plainFileStore,
		Verbose:        *verbose,
		VerboseShort:   *verboseShort,
		IncludeLocal:   *includeLocal,
	}
	return runGitHubImportWithOptions(ctx, stdout, command, owner, repo, args, options)
}

type githubImportOptions struct {
	OutDir         string
	TokenEnv       string
	APIBase        string
	Timeout        time.Duration
	Concurrency    int
	PlainFileStore bool
	Verbose        bool
	VerboseShort   bool
	IncludeLocal   bool
}

func runGitHubImportWithOptions(ctx context.Context, stdout io.Writer, command, owner, repo string, args []string, options githubImportOptions) error {
	token := githubTokenFromEnvironment(options.TokenEnv)
	authMode := "none"
	if token == "" {
		store, err := credentialStore(options.PlainFileStore)
		if err == nil {
			stored, err := store.GitHubToken()
			if err == nil {
				token = stored.AccessToken
				authMode = "stored"
			}
		}
	} else {
		authMode = "environment"
	}
	startedAt := time.Now().UTC()
	operationID := ledger.NewOperationID(command, startedAt)
	writeField(stdout, "Repository", owner+"/"+repo)
	writeField(stdout, "Ledger", options.OutDir)
	if token == "" {
		writeField(stdout, "Auth", "unauthenticated")
	} else {
		writeField(stdout, "Auth", "authenticated")
	}
	fmt.Fprintln(stdout)

	showDetails := options.Verbose || options.VerboseShort
	var progressMu sync.Mutex
	client := github.NewClient(options.APIBase, token, options.Timeout).WithConcurrency(options.Concurrency).WithProgress(func(progress github.Progress) {
		progressMu.Lock()
		defer progressMu.Unlock()
		printImportProgress(stdout, progress, showDetails)
	})
	authLogin := ""
	if token != "" {
		login, err := client.AuthenticatedUser(ctx)
		if err != nil {
			return err
		}
		authLogin = login
	}
	imported, err := client.ImportRepository(ctx, owner, repo)
	if err != nil {
		return err
	}
	if existingSource, err := (ledger.Reader{Root: options.OutDir}).Source(imported.Source); err == nil {
		imported.Source.Operations = existingSource.Operations
	}
	imported.Source.Operations = append(imported.Source.Operations, model.SourceOperationRef{
		ID:        operationID,
		Command:   command,
		Path:      ledger.OperationPath(operationID),
		StartedAt: startedAt,
	})

	writer := ledger.Writer{Root: options.OutDir}
	diff, err := writer.DiffGitHubImport(imported)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, "- Writing ledger...")
	ledgerExisted := fileExists(filepath.Join(options.OutDir, "ledger.json"))
	if err := writer.WriteGitHubImport(imported); err != nil {
		return err
	}
	if err := addLedgerMetadataChange(&diff, options.OutDir, ledgerExisted); err != nil {
		return err
	}

	finishedAt := time.Now().UTC()
	operation := modelOperation(command, args, startedAt, finishedAt, owner+"/"+repo, options.OutDir, authMode, authLogin, options.IncludeLocal, imported, diff)
	operation.ID = operationID
	if err := writer.WriteOperation(operation); err != nil {
		return err
	}

	fmt.Fprintln(stdout)
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
	writeIndentedField(stdout, "Review comments", len(imported.ReviewComments))
	writeIndentedField(stdout, "Labels", len(imported.Labels))
	writeIndentedField(stdout, "Milestones", len(imported.Milestones))
	writeIndentedField(stdout, "Releases", len(imported.Releases))
	return nil
}

func credentialStore(plainFileStore bool) (auth.CredentialStore, error) {
	if !plainFileStore {
		return auth.DefaultStore(), nil
	}
	store, err := auth.DefaultPlaintextStore()
	if err != nil {
		return nil, err
	}
	return store, nil
}

func defaultGitHubClientID() string {
	if clientID := os.Getenv("OAUTH_CLIENT_ID"); clientID != "" {
		return clientID
	}
	return defaultGitHubOAuthClientID
}

func githubTokenFromEnvironment(tokenEnv string) string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	// #nosec G101 -- this compares environment variable names, not credential values.
	if tokenEnv != "" && tokenEnv != "GITHUB_TOKEN" {
		return os.Getenv(tokenEnv)
	}
	return ""
}

func modelOperation(command string, args []string, startedAt, finishedAt time.Time, source, root, authMode, authLogin string, includeLocal bool, imported model.GitHubImport, diff ledger.Diff) model.Operation {
	return model.Operation{
		ID:         ledger.NewOperationID(command, startedAt),
		Command:    command,
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), includeLocal),
		Auth: model.OperationAuth{
			Provider: "github",
			Mode:     authMode,
			Login:    authLogin,
		},
		Input: map[string]string{
			"source": "github:" + source,
		},
		Output: model.OperationOutput{
			Ledger:    root,
			Created:   diff.Created,
			Updated:   diff.Updated,
			Deleted:   diff.Deleted,
			Unchanged: diff.Unchanged,
			Summary: model.RecordSummary{
				Issues:         len(imported.Issues),
				Comments:       len(imported.Comments),
				PullRequests:   len(imported.PullRequests),
				ReviewComments: len(imported.ReviewComments),
				Labels:         len(imported.Labels),
				Milestones:     len(imported.Milestones),
				Releases:       len(imported.Releases),
			},
		},
		Changes: diff.Changes,
	}
}
