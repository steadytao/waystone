// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/steadytao/waystone/internal/auth"
	"github.com/steadytao/waystone/internal/github"
	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

const defaultGitHubOAuthClientID = "Ov23liWNheWsFXT3BnPf"

const Version = "0.0.0-dev"

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("missing command")
	}

	switch args[0] {
	case "github":
		return runGitHub(ctx, args[1:], stdout, stderr)
	case "issue":
		return runIssue(args[1:], stdout, stderr)
	case "label":
		return runLabel(args[1:], stdout, stderr)
	case "ledger":
		return runLedger(ctx, args[1:], stdout, stderr)
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

func runVersion(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone version [flags]")
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, map[string]string{"version": Version})
	}
	fmt.Fprintln(stdout, Version)
	return nil
}

func runLedger(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printLedgerUsage(stderr)
		return errors.New("missing ledger command")
	}

	switch args[0] {
	case "doctor":
		return runLedgerDoctor(args[1:], stdout)
	case "diff":
		return runLedgerDiff(args[1:], stdout)
	case "export":
		return runLedgerExport(args[1:], stdout)
	case "import":
		return runLedgerImport(ctx, args[1:], stdout)
	case "inspect":
		return runLedgerInspect(args[1:], stdout)
	case "summary":
		return runLedgerSummary(args[1:], stdout)
	case "status":
		return runLedgerStatus(args[1:], stdout)
	case "history":
		return runLedgerHistory(args[1:], stdout)
	case "show-operation":
		return runLedgerShowOperation(args[1:], stdout)
	case "verify":
		return runLedgerVerify(args[1:], stdout)
	default:
		printLedgerUsage(stderr)
		return fmt.Errorf("unknown ledger command %q", args[0])
	}
}

func runLedgerDoctor(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	staleAfter := fs.String("stale-after", "30d", "warn when a source has not been refreshed for this duration, or 0 to disable")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	staleDuration, err := parseStaleDuration(*staleAfter)
	if err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	findings := doctorLedger(reader, staleDuration, time.Now().UTC())
	if findings == nil {
		findings = []doctorFinding{}
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, findings)
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "No practical ledger issues found")
		return nil
	}
	writeField(stdout, "Findings", len(findings))
	fmt.Fprintln(stdout)
	for _, finding := range findings {
		fmt.Fprintf(stdout, "- %-8s %s\n", finding.Severity, finding.Message)
	}
	return nil
}

func runLedgerDiff(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger diff", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "source to diff, e.g. github:owner/repo")
	since := fs.String("since", "", "operation ID to diff after")
	includeVerified := fs.Bool("include-verified", false, "include verification-only changes")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sourceFlag == "" || *since == "" || fs.NArg() != 0 {
		return errors.New("usage: waystone ledger diff --source <source> --since <operation>")
	}
	reader := ledger.Reader{Root: *root}
	source, err := ledger.ParseSourceSpec(*sourceFlag)
	if err != nil {
		return err
	}
	source, err = reader.Source(source)
	if err != nil {
		return err
	}
	operation, err := reader.Operation(*since)
	if err != nil {
		return err
	}
	diff, err := ledgerDiffSince(reader, source, operation.ID, *includeVerified)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, diff)
	}
	writeField(stdout, "Source", diff.Source)
	writeField(stdout, "Since", diff.Since)
	writeField(stdout, "Operations", len(diff.Operations))
	writeField(stdout, "Changes", len(diff.Changes))
	fmt.Fprintln(stdout)
	if len(diff.Changes) == 0 {
		fmt.Fprintln(stdout, "No source changes found")
		return nil
	}
	fmt.Fprintf(stdout, "%-10s %-16s %-8s %-32s %s\n", "TYPE", "OBJECT", "NUMBER", "OPERATION", "PATH")
	for _, change := range diff.Changes {
		number := ""
		if change.Number > 0 {
			number = fmt.Sprintf("#%d", change.Number)
		}
		fmt.Fprintf(stdout, "%-10s %-16s %-8s %-32s %s\n", change.Type, change.Object, number, change.OperationID, change.Path)
	}
	return nil
}

func runLedgerVerify(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")
	strict := fs.Bool("strict", false, "strictly verify operation chain and recorded file hashes")
	operations := fs.Bool("operations", false, "strictly verify operation chain and recorded file hashes")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	startedAt := time.Now().UTC()
	reader := ledger.Reader{Root: *root}
	verification, err := reader.Verify()
	if err != nil {
		return err
	}
	var operationVerification ledger.OperationVerification
	verifyOperations := *strict || *operations
	if verifyOperations {
		operationVerification, err = reader.VerifyOperations()
		if err != nil {
			return err
		}
	}
	finishedAt := time.Now().UTC()
	changes := append([]model.ObjectChange(nil), verification.Changes...)
	command := "ledger verify"
	if *strict {
		command = "ledger verify --strict"
		changes = append(changes, operationVerification.Changes...)
	} else if verifyOperations {
		command = "ledger verify --operations"
		changes = append(changes, operationVerification.Changes...)
	}
	operation := model.Operation{
		ID:         ledger.NewOperationID(command, startedAt),
		Command:    command,
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), *includeLocal),
		Output: model.OperationOutput{
			Ledger:    *root,
			Unchanged: verification.Files,
		},
		Changes: changes,
	}
	if err := (ledger.Writer{Root: *root}).WriteOperation(operation); err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, map[string]any{
			"ledger":     *root,
			"files":      verification.Files,
			"checksum":   verification.Checksum,
			"operations": operationVerification,
			"operation":  operation.ID,
		})
	}
	writeField(stdout, "Ledger", *root)
	writeField(stdout, "JSON files", verification.Files)
	writeField(stdout, "Checksum", verification.Checksum)
	if verifyOperations {
		writeField(stdout, "Operations", operationVerification.Operations)
		writeField(stdout, "Recorded files", operationVerification.Files)
		writeField(stdout, "Operation checksum", operationVerification.Checksum)
	}
	writeField(stdout, "Operation", operation.ID)
	return nil
}

func runLedgerExport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	out := fs.String("out", "waystone-ledger", "export path")
	format := fs.String("format", "archive", "export format: archive or json")
	source := fs.String("source", "", "export only one source, e.g. github:owner/repo")
	compact := fs.Bool("compact", false, "write compact JSON when --format=json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *format {
	case "archive":
		if *source != "" {
			if err := ledger.ExportSourceArchive(*root, *source, *out); err != nil {
				return err
			}
		} else {
			if _, err := (ledger.Reader{Root: *root}).VerifyOperations(); err != nil {
				return err
			}
			if err := ledger.ExportArchive(*root, *out); err != nil {
				return err
			}
		}
	case "json":
		if *source != "" {
			return errors.New("--source is only supported for archive export")
		}
		if err := ledger.ExportJSON(*root, *out, *compact); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported export format %q", *format)
	}
	writeField(stdout, "Ledger", *root)
	writeField(stdout, "Format", *format)
	writeField(stdout, "Output", *out)
	fmt.Fprintln(stdout, "Export complete")
	return nil
}

func runLedgerInspect(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone ledger inspect [flags] <archive>")
	}
	inspection, err := ledger.InspectArchive(fs.Arg(0))
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, inspection)
	}
	writeField(stdout, "Format", inspection.Format)
	writeField(stdout, "Files", inspection.Files)
	writeField(stdout, "Sources", inspection.Sources)
	writeField(stdout, "Operations", inspection.Operations)
	writeField(stdout, "Strict", inspection.Strict)
	writeField(stdout, "Checksum", inspection.Checksum)
	return nil
}

func runLedgerImport(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	unsafe := fs.Bool("unsafe", false, "skip remote source confirmation before importing")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable containing a GitHub token")
	apiBase := fs.String("api-base", "https://api.github.com", "GitHub API base URL")
	timeout := fs.Duration("timeout", 2*time.Minute, "request timeout")
	plainFileStore := fs.Bool("plain-file-store", false, "read stored token from a plaintext local file instead of the OS credential store")
	normalizedArgs, err := normalizeLedgerImportArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone ledger import [flags] <archive>")
	}
	if *unsafe {
		if err := ledger.ImportArchive(fs.Arg(0), *root); err != nil {
			return err
		}
		writeField(stdout, "Ledger", *root)
		fmt.Fprintln(stdout, "Import complete")
		return nil
	}

	tempParent := filepath.Dir(*root)
	if err := os.MkdirAll(tempParent, 0o700); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(tempParent, ".waystone-verified-import-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	tempRoot := filepath.Join(tempDir, ".waystone")
	if err := ledger.ImportArchive(fs.Arg(0), tempRoot); err != nil {
		return err
	}
	if err := confirmGitHubSources(ctx, ledger.Reader{Root: tempRoot}, *tokenEnv, *plainFileStore, *apiBase, *timeout); err != nil {
		return err
	}
	if err := ensureImportDestination(*root); err != nil {
		return err
	}
	if err := os.Rename(tempRoot, *root); err != nil {
		return err
	}
	writeField(stdout, "Ledger", *root)
	fmt.Fprintln(stdout, "Import complete")
	return nil
}

func confirmGitHubSources(ctx context.Context, reader ledger.Reader, tokenEnv string, plainFileStore bool, apiBase string, timeout time.Duration) error {
	sources, err := reader.Sources()
	if err != nil {
		return err
	}
	var githubSources []model.Source
	for _, source := range sources {
		if source.System == "github" {
			githubSources = append(githubSources, source)
		}
	}
	if len(githubSources) == 0 {
		return nil
	}
	token := githubTokenFromEnvironment(tokenEnv)
	if token == "" {
		store, err := credentialStore(plainFileStore)
		if err == nil {
			stored, err := store.GitHubToken()
			if err == nil {
				token = stored.AccessToken
			}
		}
	}
	if token == "" {
		return errors.New("safe import requires GitHub authentication for GitHub source confirmation; use github auth login, GITHUB_TOKEN or --unsafe")
	}
	client := github.NewClient(apiBase, token, timeout)
	for _, source := range githubSources {
		if err := client.ConfirmRepository(ctx, source.Owner, source.Repo); err != nil {
			return fmt.Errorf("confirm github source %s/%s: %w", source.Owner, source.Repo, err)
		}
	}
	return nil
}

func ensureImportDestination(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("ledger %s already exists and is not empty", root)
	}
	return nil
}

func normalizeLedgerImportArgs(args []string) ([]string, error) {
	var flags []string
	var archive string
	boolFlags := map[string]bool{
		"--unsafe":           true,
		"-unsafe":            true,
		"--plain-file-store": true,
		"-plain-file-store":  true,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if strings.Contains(arg, "=") || boolFlags[arg] {
				continue
			}
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags = append(flags, args[i])
			continue
		}
		if archive != "" {
			return nil, fmt.Errorf("unexpected extra argument %q", arg)
		}
		archive = arg
	}
	if archive == "" {
		return flags, nil
	}
	return append(flags, archive), nil
}

func runLedgerHistory(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger history", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	operations, err := (ledger.Reader{Root: *root}).Operations()
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, operations)
	}
	fmt.Fprintf(stdout, "%-32s %-18s %-10s %s\n", "OPERATION", "COMMAND", "CHANGES", "FINISHED")
	for _, operation := range operations {
		actions := operation.Output.Created + operation.Output.Updated + operation.Output.Deleted + operation.Output.Unchanged
		fmt.Fprintf(stdout, "%-32s %-18s %-10d %s\n", operation.ID, operation.Command, actions, operation.FinishedAt.Format(time.RFC3339))
	}
	return nil
}

func runLedgerShowOperation(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger show-operation", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone ledger show-operation [flags] <operation-id>")
	}
	operation, err := (ledger.Reader{Root: *root}).Operation(fs.Arg(0))
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, operation)
	}
	writeField(stdout, "Operation", operation.ID)
	writeField(stdout, "Command", operation.Command)
	writeField(stdout, "Started", operation.StartedAt.Format(time.RFC3339))
	writeField(stdout, "Finished", operation.FinishedAt.Format(time.RFC3339))
	writeField(stdout, "Actor", operation.Actor.User)
	writeField(stdout, "Host", operation.Actor.Hostname)
	writeField(stdout, "Auth", operation.Auth.Mode)
	writeField(stdout, "Ledger", operation.Output.Ledger)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Changes")
	writeIndentedField(stdout, "Created", operation.Output.Created)
	writeIndentedField(stdout, "Updated", operation.Output.Updated)
	writeIndentedField(stdout, "Deleted", operation.Output.Deleted)
	writeIndentedField(stdout, "Unchanged", operation.Output.Unchanged)
	if len(operation.Changes) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Objects")
		for _, change := range operation.Changes {
			ref := change.ID
			if change.Number > 0 {
				ref = fmt.Sprintf("#%d", change.Number)
			}
			fmt.Fprintf(stdout, "  %-8s %-14s %-8s %s\n", change.Type, change.Object, ref, change.Path)
		}
	}
	return nil
}

func runLedgerSummary(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger summary", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	summary, err := (ledger.Reader{Root: *root}).Summary()
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, summary)
	}
	writeField(stdout, "Projects", len(summary.Projects))
	writeField(stdout, "Sources", len(summary.Sources))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Records")
	writeIndentedField(stdout, "Issues", summary.Issues)
	writeIndentedField(stdout, "Comments", summary.Comments)
	writeIndentedField(stdout, "Pull requests", summary.PullRequests)
	writeIndentedField(stdout, "Review comments", summary.ReviewComments)
	writeIndentedField(stdout, "Labels", summary.Labels)
	writeIndentedField(stdout, "Milestones", summary.Milestones)
	writeIndentedField(stdout, "Releases", summary.Releases)
	return nil
}

func runLedgerStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone ledger status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reader := ledger.Reader{Root: *root}
	summary, err := reader.Summary()
	if err != nil {
		return err
	}
	operations, err := reader.Operations()
	if err != nil {
		return err
	}
	verification, err := reader.Verify()
	if err != nil {
		return err
	}
	status := map[string]any{
		"ledger":     *root,
		"projects":   summary.Projects,
		"sources":    summary.Sources,
		"records":    summary,
		"operations": len(operations),
		"files":      verification.Files,
		"checksum":   verification.Checksum,
	}
	if len(operations) > 0 {
		status["last_operation"] = operations[len(operations)-1]
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, status)
	}

	writeField(stdout, "Projects", len(summary.Projects))
	writeField(stdout, "Ledger", *root)
	writeField(stdout, "Sources", len(summary.Sources))
	writeField(stdout, "Operations", len(operations))
	writeField(stdout, "JSON files", verification.Files)
	writeField(stdout, "Checksum", verification.Checksum)
	if len(operations) > 0 {
		last := operations[len(operations)-1]
		writeField(stdout, "Last command", last.Command)
		writeField(stdout, "Last operation", last.ID)
	}
	return nil
}

func runIssue(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printIssueUsage(stderr)
		return errors.New("missing issue command")
	}

	switch args[0] {
	case "list":
		return runIssueList(args[1:], stdout)
	case "search":
		return runIssueSearch(args[1:], stdout)
	case "show":
		return runIssueShow(args[1:], stdout)
	case "comments":
		return runIssueComments(args[1:], stdout)
	case "timeline":
		return runIssueTimeline(args[1:], stdout)
	default:
		printIssueUsage(stderr)
		return fmt.Errorf("unknown issue command %q", args[0])
	}
}

func runIssueList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone issue list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var issues []model.Issue
	if sourceSet {
		issues, err = reader.SourceIssues(source)
	} else {
		issues, err = reader.Issues()
	}
	if err != nil {
		return err
	}
	writeField(stdout, "Issues", len(issues))
	fmt.Fprintln(stdout)
	if sourceSet {
		fmt.Fprintf(stdout, "%-8s %-8s %s\n", "NUMBER", "STATE", "TITLE")
	} else {
		fmt.Fprintf(stdout, "%-28s %-8s %-8s %s\n", "SOURCE", "NUMBER", "STATE", "TITLE")
	}
	for _, issue := range issues {
		if sourceSet {
			fmt.Fprintf(stdout, "#%-7d %-8s %s\n", issue.Number, issue.State, issue.Title)
		} else {
			fmt.Fprintf(stdout, "%-28s #%-7d %-8s %s\n", ledger.SourceSpec(issue.Source), issue.Number, issue.State, issue.Title)
		}
	}
	return nil
}

func runIssueShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone issue show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	withComments := fs.Bool("with-comments", false, "include issue comments")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone issue show [flags] <number>")
	}
	number, err := parseNumber(fs.Arg(0))
	if err != nil {
		return err
	}

	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var issue model.Issue
	if sourceSet {
		issue, err = reader.SourceIssue(source, number)
	} else {
		issue, err = reader.Issue(number)
	}
	if err != nil {
		return err
	}
	var comments []model.Comment
	if *withComments {
		comments, err = reader.SourceComments(issue.Source, number)
		if err != nil {
			return err
		}
	}
	if *jsonOutput {
		if *withComments {
			return writeJSONOutput(stdout, map[string]any{
				"issue":    issue,
				"comments": comments,
			})
		}
		return writeJSONOutput(stdout, issue)
	}
	writeField(stdout, "Number", fmt.Sprintf("#%d", issue.Number))
	writeField(stdout, "Source", ledger.SourceSpec(issue.Source))
	writeField(stdout, "Title", issue.Title)
	writeField(stdout, "State", issue.State)
	writeField(stdout, "Author", issue.Author.Login)
	writeField(stdout, "URL", issue.OriginalURL)
	if issue.Milestone != "" {
		writeField(stdout, "Milestone", issue.Milestone)
	}
	if len(issue.Labels) > 0 {
		writeField(stdout, "Labels", strings.Join(issue.Labels, ", "))
	}
	if issue.Body != "" {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Body")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, issue.Body)
	}
	if *withComments {
		writeIssueComments(stdout, issue.Number, comments)
	}
	return nil
}

func runIssueComments(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone issue comments", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone issue comments [flags] <number>")
	}
	number, err := parseNumber(fs.Arg(0))
	if err != nil {
		return err
	}

	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var comments []model.Comment
	if sourceSet {
		comments, err = reader.SourceComments(source, number)
	} else {
		if _, err := reader.Issue(number); err != nil {
			return err
		}
		comments, err = reader.Comments(number)
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, comments)
	}
	writeIssueComments(stdout, number, comments)
	return nil
}

func runIssueTimeline(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone issue timeline", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone issue timeline [flags] <number>")
	}
	number, err := parseNumber(fs.Arg(0))
	if err != nil {
		return err
	}

	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var issue model.Issue
	if sourceSet {
		issue, err = reader.SourceIssue(source, number)
	} else {
		issue, err = reader.Issue(number)
	}
	if err != nil {
		return err
	}
	comments, err := reader.SourceComments(issue.Source, number)
	if err != nil {
		return err
	}
	events := issueTimeline(issue, comments)
	if *jsonOutput {
		return writeJSONOutput(stdout, events)
	}
	writeTimeline(stdout, "Issue", issue.Number, ledger.SourceSpec(issue.Source), events)
	return nil
}

func runIssueSearch(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone issue search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	var fields searchFieldsFlag
	fs.Var(&fields, "field", "field to search: title, description, author, state, label, milestone, url or all")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	normalizedArgs, err := normalizeSingleValueCommandArgs(args, "text", map[string]bool{"--json": true, "-json": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone issue search [flags] <text>")
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var issues []model.Issue
	if sourceSet {
		issues, err = reader.SourceIssues(source)
	} else {
		issues, err = reader.Issues()
	}
	if err != nil {
		return err
	}
	selection, err := normalizeSearchFields(fields, issueSearchFields())
	if err != nil {
		return err
	}
	matches := searchIssues(issues, fs.Arg(0), selection)
	if *jsonOutput {
		return writeJSONOutput(stdout, matches)
	}
	writeField(stdout, "Issues", len(matches))
	fmt.Fprintln(stdout)
	if sourceSet {
		fmt.Fprintf(stdout, "%-8s %-8s %-12s %s\n", "NUMBER", "STATE", "MATCH", "TITLE")
	} else {
		fmt.Fprintf(stdout, "%-28s %-8s %-8s %-12s %s\n", "SOURCE", "NUMBER", "STATE", "MATCH", "TITLE")
	}
	for _, issue := range matches {
		match := matchingIssueField(issue, fs.Arg(0), selection)
		if sourceSet {
			fmt.Fprintf(stdout, "#%-7d %-8s %-12s %s\n", issue.Number, issue.State, match, issue.Title)
		} else {
			fmt.Fprintf(stdout, "%-28s #%-7d %-8s %-12s %s\n", ledger.SourceSpec(issue.Source), issue.Number, issue.State, match, issue.Title)
		}
	}
	return nil
}

func runPullRequest(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printPullRequestUsage(stderr)
		return errors.New("missing pr command")
	}

	switch args[0] {
	case "list":
		return runPullRequestList(args[1:], stdout)
	case "search":
		return runPullRequestSearch(args[1:], stdout)
	case "show":
		return runPullRequestShow(args[1:], stdout)
	case "comments":
		return runPullRequestComments(args[1:], stdout)
	case "timeline":
		return runPullRequestTimeline(args[1:], stdout)
	default:
		printPullRequestUsage(stderr)
		return fmt.Errorf("unknown pr command %q", args[0])
	}
}

func runPullRequestShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone pr show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	withComments := fs.Bool("with-comments", false, "include review comments")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone pr show [flags] <number>")
	}
	number, err := parseNumber(fs.Arg(0))
	if err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var pr model.PullRequest
	if sourceSet {
		pr, err = reader.SourcePullRequest(source, number)
	} else {
		pr, err = reader.PullRequest(number)
	}
	if err != nil {
		return err
	}
	var comments []model.ReviewComment
	if *withComments {
		comments, err = reader.SourceReviewComments(pr.Source, number)
		if err != nil {
			return err
		}
	}
	if *jsonOutput {
		if *withComments {
			return writeJSONOutput(stdout, map[string]any{
				"pull_request":    pr,
				"review_comments": comments,
			})
		}
		return writeJSONOutput(stdout, pr)
	}
	writeField(stdout, "Number", fmt.Sprintf("#%d", pr.Number))
	writeField(stdout, "Source", ledger.SourceSpec(pr.Source))
	writeField(stdout, "Title", pr.Title)
	writeField(stdout, "State", pr.State)
	writeField(stdout, "Author", pr.Author.Login)
	writeField(stdout, "URL", pr.OriginalURL)
	writeField(stdout, "Base", pr.BaseRef)
	writeField(stdout, "Head", pr.HeadRef)
	writeField(stdout, "Merged", pr.Merged)
	if pr.Body != "" {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Body")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, pr.Body)
	}
	if *withComments {
		writePullRequestComments(stdout, pr.Number, comments)
	}
	return nil
}

func runPullRequestComments(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone pr comments", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone pr comments [flags] <number>")
	}
	number, err := parseNumber(fs.Arg(0))
	if err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var comments []model.ReviewComment
	if sourceSet {
		comments, err = reader.SourceReviewComments(source, number)
	} else {
		if _, err := reader.PullRequest(number); err != nil {
			return err
		}
		comments, err = reader.ReviewComments(number)
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, comments)
	}
	writePullRequestComments(stdout, number, comments)
	return nil
}

func runPullRequestTimeline(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone pr timeline", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone pr timeline [flags] <number>")
	}
	number, err := parseNumber(fs.Arg(0))
	if err != nil {
		return err
	}

	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var pr model.PullRequest
	if sourceSet {
		pr, err = reader.SourcePullRequest(source, number)
	} else {
		pr, err = reader.PullRequest(number)
	}
	if err != nil {
		return err
	}
	comments, err := reader.SourceReviewComments(pr.Source, number)
	if err != nil {
		return err
	}
	conversationComments, err := reader.SourceComments(pr.Source, number)
	if err != nil {
		return err
	}
	events := pullRequestTimeline(pr, conversationComments, comments)
	if *jsonOutput {
		return writeJSONOutput(stdout, events)
	}
	writeTimeline(stdout, "Pull request", pr.Number, ledger.SourceSpec(pr.Source), events)
	return nil
}

func runPullRequestSearch(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone pr search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	var fields searchFieldsFlag
	fs.Var(&fields, "field", "field to search: title, description, author, state, branch, url or all")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	normalizedArgs, err := normalizeSingleValueCommandArgs(args, "text", map[string]bool{"--json": true, "-json": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone pr search [flags] <text>")
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var prs []model.PullRequest
	if sourceSet {
		prs, err = reader.SourcePullRequests(source)
	} else {
		prs, err = reader.PullRequests()
	}
	if err != nil {
		return err
	}
	selection, err := normalizeSearchFields(fields, pullRequestSearchFields())
	if err != nil {
		return err
	}
	matches := searchPullRequests(prs, fs.Arg(0), selection)
	if *jsonOutput {
		return writeJSONOutput(stdout, matches)
	}
	writeField(stdout, "Pull requests", len(matches))
	fmt.Fprintln(stdout)
	if sourceSet {
		fmt.Fprintf(stdout, "%-8s %-8s %-12s %s\n", "NUMBER", "STATE", "MATCH", "TITLE")
	} else {
		fmt.Fprintf(stdout, "%-28s %-8s %-8s %-12s %s\n", "SOURCE", "NUMBER", "STATE", "MATCH", "TITLE")
	}
	for _, pr := range matches {
		match := matchingPullRequestField(pr, fs.Arg(0), selection)
		if sourceSet {
			fmt.Fprintf(stdout, "#%-7d %-8s %-12s %s\n", pr.Number, pr.State, match, pr.Title)
		} else {
			fmt.Fprintf(stdout, "%-28s #%-7d %-8s %-12s %s\n", ledger.SourceSpec(pr.Source), pr.Number, pr.State, match, pr.Title)
		}
	}
	return nil
}

func runLabel(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printLabelUsage(stderr)
		return errors.New("missing label command")
	}
	switch args[0] {
	case "list":
		return runLabelList(args[1:], stdout)
	default:
		printLabelUsage(stderr)
		return fmt.Errorf("unknown label command %q", args[0])
	}
}

func runLabelList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone label list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var labels []model.Label
	if sourceSet {
		labels, err = reader.SourceLabels(source)
	} else {
		labels, err = reader.Labels()
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, labels)
	}
	if sourceSet {
		fmt.Fprintf(stdout, "%-24s %-8s %s\n", "NAME", "COLOR", "DESCRIPTION")
	} else {
		fmt.Fprintf(stdout, "%-28s %-24s %-8s %s\n", "SOURCE", "NAME", "COLOR", "DESCRIPTION")
	}
	for _, label := range labels {
		if sourceSet {
			fmt.Fprintf(stdout, "%-24s %-8s %s\n", label.Name, label.Color, label.Description)
		} else {
			fmt.Fprintf(stdout, "%-28s %-24s %-8s %s\n", ledger.SourceSpec(label.Source), label.Name, label.Color, label.Description)
		}
	}
	return nil
}

func runMilestone(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printMilestoneUsage(stderr)
		return errors.New("missing milestone command")
	}
	switch args[0] {
	case "list":
		return runMilestoneList(args[1:], stdout)
	default:
		printMilestoneUsage(stderr)
		return fmt.Errorf("unknown milestone command %q", args[0])
	}
}

func runMilestoneList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone milestone list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var milestones []model.Milestone
	if sourceSet {
		milestones, err = reader.SourceMilestones(source)
	} else {
		milestones, err = reader.Milestones()
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, milestones)
	}
	if sourceSet {
		fmt.Fprintf(stdout, "%-8s %-8s %s\n", "NUMBER", "STATE", "TITLE")
	} else {
		fmt.Fprintf(stdout, "%-28s %-8s %-8s %s\n", "SOURCE", "NUMBER", "STATE", "TITLE")
	}
	for _, milestone := range milestones {
		if sourceSet {
			fmt.Fprintf(stdout, "#%-7d %-8s %s\n", milestone.Number, milestone.State, milestone.Title)
		} else {
			fmt.Fprintf(stdout, "%-28s #%-7d %-8s %s\n", ledger.SourceSpec(milestone.Source), milestone.Number, milestone.State, milestone.Title)
		}
	}
	return nil
}

func runSource(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printSourceUsage(stderr)
		return errors.New("missing source command")
	}
	switch args[0] {
	case "default":
		return runSourceDefault(args[1:], stdout)
	case "list":
		return runSourceList(args[1:], stdout)
	case "refresh":
		return runSourceRefresh(ctx, args[1:], stdout)
	case "show":
		return runSourceShow(args[1:], stdout)
	case "inspect":
		return runSourceInspect(args[1:], stdout)
	case "status":
		return runSourceStatus(args[1:], stdout)
	default:
		printSourceUsage(stderr)
		return fmt.Errorf("unknown source command %q", args[0])
	}
}

func runSourceList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone source list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sources, err := (ledger.Reader{Root: *root}).Sources()
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, sources)
	}
	fmt.Fprintf(stdout, "%-28s %-8s %-10s %s\n", "SOURCE", "OBJECTS", "OPERATIONS", "URL")
	for _, source := range sources {
		fmt.Fprintf(stdout, "%-28s %-8d %-10d %s\n", ledger.SourceSpec(source), len(source.Objects), len(source.Operations), source.URL)
	}
	return nil
}

func runSourceDefault(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone source default", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	clear := fs.Bool("clear", false, "clear the default source")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	if *clear {
		if fs.NArg() != 0 {
			return errors.New("usage: waystone source default [flags] [source]")
		}
		startedAt := time.Now().UTC()
		if err := (ledger.Writer{Root: *root}).SetDefaultSource(model.Source{}); err != nil {
			return err
		}
		if err := writeSourceDefaultOperation(*root, args, startedAt, model.Source{}, *includeLocal); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "Default source cleared")
		return nil
	}
	switch fs.NArg() {
	case 0:
		current, err := reader.Ledger()
		if err != nil {
			return err
		}
		if current.DefaultSource == nil {
			fmt.Fprintln(stdout, "Default source is not set")
			return nil
		}
		writeField(stdout, "Default source", ledger.SourceSpec(*current.DefaultSource))
		return nil
	case 1:
		source, err := ledger.ParseSourceSpec(fs.Arg(0))
		if err != nil {
			return err
		}
		startedAt := time.Now().UTC()
		if err := (ledger.Writer{Root: *root}).SetDefaultSource(source); err != nil {
			return err
		}
		if err := writeSourceDefaultOperation(*root, args, startedAt, source, *includeLocal); err != nil {
			return err
		}
		writeField(stdout, "Default source", ledger.SourceSpec(source))
		return nil
	default:
		return errors.New("usage: waystone source default [flags] [source]")
	}
}

func runSourceShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone source show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone source show [flags] <source>")
	}
	source, err := ledger.ParseSourceSpec(fs.Arg(0))
	if err != nil {
		return err
	}
	source, err = (ledger.Reader{Root: *root}).Source(source)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, source)
	}
	writeField(stdout, "Source", ledger.SourceSpec(source))
	writeField(stdout, "URL", source.URL)
	writeField(stdout, "Objects", len(source.Objects))
	writeField(stdout, "Operations", len(source.Operations))
	return nil
}

func runSourceInspect(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone source inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	staleAfter := fs.String("stale-after", "30d", "mark source stale after this duration, or 0 to disable")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	normalizedArgs, err := normalizeSingleValueCommandArgs(args, "source", map[string]bool{"--json": true, "-json": true})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone source inspect [flags] <source>")
	}
	staleDuration, err := parseStaleDuration(*staleAfter)
	if err != nil {
		return err
	}
	source, err := ledger.ParseSourceSpec(fs.Arg(0))
	if err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	source, err = reader.Source(source)
	if err != nil {
		return err
	}
	inspection, err := inspectSource(*root, source, staleDuration, time.Now().UTC())
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, inspection)
	}
	writeField(stdout, "Source", inspection.Spec)
	writeField(stdout, "URL", inspection.URL)
	writeField(stdout, "Manifest", inspection.ManifestPath)
	writeField(stdout, "Manifest hash", inspection.ManifestSHA256)
	writeField(stdout, "Objects", inspection.Objects)
	writeField(stdout, "Operations", inspection.Operations)
	writeField(stdout, "Last refresh", inspection.LastRefreshText)
	writeField(stdout, "Age", inspection.Age)
	writeField(stdout, "Stale", inspection.Stale)
	writeField(stdout, "Missing objects", inspection.MissingObjects)
	writeField(stdout, "Changed objects", inspection.ChangedObjects)
	if len(inspection.ObjectTypes) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Object types")
		for _, objectType := range sortedCountKeys(inspection.ObjectTypes) {
			writeIndentedField(stdout, objectType, inspection.ObjectTypes[objectType])
		}
	}
	if len(inspection.Hints) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Hints")
		for _, hint := range inspection.Hints {
			fmt.Fprintf(stdout, "- %s\n", hint)
		}
	}
	return nil
}

func runSourceRefresh(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone source refresh", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable containing a GitHub token")
	apiBase := fs.String("api-base", "https://api.github.com", "GitHub API base URL")
	timeout := fs.Duration("timeout", 2*time.Minute, "request timeout")
	concurrency := fs.Int("concurrency", 6, "maximum concurrent GitHub detail requests")
	plainFileStore := fs.Bool("plain-file-store", false, "read stored token from a plaintext local file instead of the OS credential store")
	verbose := fs.Bool("verbose", false, "show detailed import progress")
	verboseShort := fs.Bool("v", false, "show detailed import progress")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")
	var sourceFlags valueListFlag
	fs.Var(&sourceFlags, "source", "source to refresh, e.g. github:owner/repo; repeatable or comma-separated")
	fs.Var(&sourceFlags, "sources", "sources to refresh, e.g. github:owner/repo,github:owner/other")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone source refresh [flags]")
	}
	reader := ledger.Reader{Root: *root}
	sources, err := resolveRefreshSources(reader, sourceFlags)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return errors.New("ledger has no sources to refresh")
	}
	options := githubImportOptions{
		OutDir:         *root,
		TokenEnv:       *tokenEnv,
		APIBase:        *apiBase,
		Timeout:        *timeout,
		Concurrency:    *concurrency,
		PlainFileStore: *plainFileStore,
		Verbose:        *verbose,
		VerboseShort:   *verboseShort,
		IncludeLocal:   *includeLocal,
	}
	for i, source := range sources {
		if source.System != "github" {
			return fmt.Errorf("source refresh only supports github sources, got %q", source.System)
		}
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		if err := runGitHubImportWithOptions(ctx, stdout, "source refresh", source.Owner, source.Repo, args, options); err != nil {
			return err
		}
	}
	return nil
}

func runSourceStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone source status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	staleAfter := fs.String("stale-after", "30d", "mark sources stale after this duration, or 0 to disable")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	staleDuration, err := parseStaleDuration(*staleAfter)
	if err != nil {
		return err
	}
	statuses, err := sourceStatuses(ledger.Reader{Root: *root}, staleDuration, time.Now().UTC())
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, statuses)
	}
	fmt.Fprintf(stdout, "%-28s %-8s %-10s %-20s %-10s %-6s %s\n", "SOURCE", "OBJECTS", "OPERATIONS", "LAST REFRESH", "AGE", "STALE", "URL")
	for _, status := range statuses {
		stale := "no"
		if status.Stale {
			stale = "yes"
		}
		fmt.Fprintf(stdout, "%-28s %-8d %-10d %-20s %-10s %-6s %s\n", status.Spec, status.Objects, status.Operations, status.LastRefreshText, status.Age, stale, status.URL)
	}
	return nil
}

func runPullRequestList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone pr list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var prs []model.PullRequest
	if sourceSet {
		prs, err = reader.SourcePullRequests(source)
	} else {
		prs, err = reader.PullRequests()
	}
	if err != nil {
		return err
	}
	writeField(stdout, "Pull requests", len(prs))
	fmt.Fprintln(stdout)
	if sourceSet {
		fmt.Fprintf(stdout, "%-8s %-8s %s\n", "NUMBER", "STATE", "TITLE")
	} else {
		fmt.Fprintf(stdout, "%-28s %-8s %-8s %s\n", "SOURCE", "NUMBER", "STATE", "TITLE")
	}
	for _, pr := range prs {
		if sourceSet {
			fmt.Fprintf(stdout, "#%-7d %-8s %s\n", pr.Number, pr.State, pr.Title)
		} else {
			fmt.Fprintf(stdout, "%-28s #%-7d %-8s %s\n", ledger.SourceSpec(pr.Source), pr.Number, pr.State, pr.Title)
		}
	}
	return nil
}

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

type doctorFinding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type sourceStatus struct {
	Spec            string    `json:"spec"`
	System          string    `json:"system"`
	Owner           string    `json:"owner"`
	Repo            string    `json:"repo"`
	URL             string    `json:"url"`
	Objects         int       `json:"objects"`
	Operations      int       `json:"operations"`
	LastRefresh     time.Time `json:"last_refresh,omitempty"`
	LastRefreshText string    `json:"last_refresh_text"`
	Age             string    `json:"age,omitempty"`
	AgeSeconds      int64     `json:"age_seconds,omitempty"`
	Stale           bool      `json:"stale"`
}

type sourceInspection struct {
	Spec            string         `json:"spec"`
	System          string         `json:"system"`
	Owner           string         `json:"owner"`
	Repo            string         `json:"repo"`
	URL             string         `json:"url"`
	ManifestPath    string         `json:"manifest_path"`
	ManifestSHA256  string         `json:"manifest_sha256"`
	Objects         int            `json:"objects"`
	Operations      int            `json:"operations"`
	LastRefresh     time.Time      `json:"last_refresh,omitempty"`
	LastRefreshText string         `json:"last_refresh_text"`
	Age             string         `json:"age,omitempty"`
	AgeSeconds      int64          `json:"age_seconds,omitempty"`
	Stale           bool           `json:"stale"`
	ObjectTypes     map[string]int `json:"object_types"`
	MissingObjects  int            `json:"missing_objects"`
	ChangedObjects  int            `json:"changed_objects"`
	Hints           []string       `json:"hints,omitempty"`
}

type ledgerDiff struct {
	Source     string             `json:"source"`
	Since      string             `json:"since"`
	Operations []string           `json:"operations"`
	Changes    []ledgerDiffChange `json:"changes"`
}

type ledgerDiffChange struct {
	OperationID string    `json:"operation_id"`
	FinishedAt  time.Time `json:"finished_at"`
	Type        string    `json:"type"`
	Object      string    `json:"object"`
	Number      int       `json:"number,omitempty"`
	ID          string    `json:"id,omitempty"`
	Path        string    `json:"path"`
	SHA256      string    `json:"sha256,omitempty"`
}

func doctorLedger(reader ledger.Reader, staleAfter time.Duration, now time.Time) []doctorFinding {
	var findings []doctorFinding
	current, err := reader.Ledger()
	if err != nil {
		return []doctorFinding{{Severity: "error", Message: "ledger metadata cannot be read: " + err.Error()}}
	}
	sources, err := reader.Sources()
	if err != nil {
		findings = append(findings, doctorFinding{Severity: "error", Message: "source manifests cannot be read: " + err.Error()})
	} else if len(sources) == 0 {
		findings = append(findings, doctorFinding{Severity: "warning", Message: "ledger has no imported sources"})
	}
	if current.DefaultSource == nil && len(sources) > 1 {
		findings = append(findings, doctorFinding{Severity: "info", Message: "multiple sources are imported but no default source is set"})
	}
	if current.DefaultSource != nil {
		if _, err := reader.Source(*current.DefaultSource); err != nil {
			findings = append(findings, doctorFinding{Severity: "warning", Message: "default source is not present in source manifests: " + ledger.SourceSpec(*current.DefaultSource)})
		}
	}
	operations, err := reader.Operations()
	if err != nil {
		findings = append(findings, doctorFinding{Severity: "error", Message: "operation history cannot be read: " + err.Error()})
	} else if len(operations) == 0 {
		findings = append(findings, doctorFinding{Severity: "warning", Message: "ledger has no operation records"})
	}
	if _, err := reader.Verify(); err != nil {
		findings = append(findings, doctorFinding{Severity: "error", Message: "JSON verification failed: " + err.Error()})
	}
	if _, err := reader.VerifyOperations(); err != nil {
		findings = append(findings, doctorFinding{Severity: "warning", Message: "strict operation verification failed: " + err.Error()})
	}
	if staleAfter > 0 {
		for _, source := range sources {
			lastRefresh, ok := lastSourceOperationTime(source)
			if !ok {
				findings = append(findings, doctorFinding{Severity: "warning", Message: fmt.Sprintf("%s has no recorded source refresh operation", ledger.SourceSpec(source))})
				continue
			}
			age := now.Sub(lastRefresh)
			if age > staleAfter {
				findings = append(findings, doctorFinding{Severity: "warning", Message: fmt.Sprintf("%s was last refreshed %s ago", ledger.SourceSpec(source), formatApproxDuration(age))})
			}
		}
	}
	if len(sources) > 1 && current.DefaultSource == nil {
		if issues, err := reader.Issues(); err == nil {
			for _, number := range duplicateIssueNumbers(issues) {
				findings = append(findings, doctorFinding{Severity: "info", Message: fmt.Sprintf("issue #%d exists in multiple sources; use --source or set a default source", number)})
			}
		}
		if prs, err := reader.PullRequests(); err == nil {
			for _, number := range duplicatePullRequestNumbers(prs) {
				findings = append(findings, doctorFinding{Severity: "info", Message: fmt.Sprintf("pull request #%d exists in multiple sources; use --source or set a default source", number)})
			}
		}
	}
	return findings
}

func sourceStatuses(reader ledger.Reader, staleAfter time.Duration, now time.Time) ([]sourceStatus, error) {
	sources, err := reader.Sources()
	if err != nil {
		return nil, err
	}
	statuses := make([]sourceStatus, 0, len(sources))
	for _, source := range sources {
		status := sourceStatus{
			Spec:            ledger.SourceSpec(source),
			System:          source.System,
			Owner:           source.Owner,
			Repo:            source.Repo,
			URL:             source.URL,
			Objects:         len(source.Objects),
			Operations:      len(source.Operations),
			LastRefreshText: "never",
		}
		if lastRefresh, ok := lastSourceOperationTime(source); ok {
			age := now.Sub(lastRefresh)
			status.LastRefresh = lastRefresh
			status.LastRefreshText = lastRefresh.Format(time.RFC3339)
			status.Age = formatApproxDuration(age)
			status.AgeSeconds = int64(age.Seconds())
			status.Stale = staleAfter > 0 && age > staleAfter
		} else {
			status.Stale = staleAfter > 0
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func inspectSource(root string, source model.Source, staleAfter time.Duration, now time.Time) (sourceInspection, error) {
	manifestPath := ledger.SourcePath(source)
	manifestFile, err := ledger.SafeRootedPath(root, manifestPath)
	if err != nil {
		return sourceInspection{}, err
	}
	manifestHash, err := fileSHA256(manifestFile)
	if err != nil {
		return sourceInspection{}, err
	}
	inspection := sourceInspection{
		Spec:            ledger.SourceSpec(source),
		System:          source.System,
		Owner:           source.Owner,
		Repo:            source.Repo,
		URL:             source.URL,
		ManifestPath:    manifestPath,
		ManifestSHA256:  manifestHash,
		Objects:         len(source.Objects),
		Operations:      len(source.Operations),
		LastRefreshText: "never",
		ObjectTypes:     map[string]int{},
	}
	if lastRefresh, ok := lastSourceOperationTime(source); ok {
		age := now.Sub(lastRefresh)
		inspection.LastRefresh = lastRefresh
		inspection.LastRefreshText = lastRefresh.Format(time.RFC3339)
		inspection.Age = formatApproxDuration(age)
		inspection.AgeSeconds = int64(age.Seconds())
		inspection.Stale = staleAfter > 0 && age > staleAfter
	} else {
		inspection.Stale = staleAfter > 0
		inspection.Hints = append(inspection.Hints, "source has no recorded refresh operation")
	}
	for _, ref := range source.Objects {
		inspection.ObjectTypes[ref.Object]++
		objectFile, err := ledger.SafeRootedPath(root, ref.Path)
		if err != nil {
			return sourceInspection{}, err
		}
		sum, err := fileSHA256(objectFile)
		if err != nil {
			if os.IsNotExist(err) {
				inspection.MissingObjects++
				continue
			}
			return sourceInspection{}, err
		}
		if ref.SHA256 != "" && sum != ref.SHA256 {
			inspection.ChangedObjects++
		}
	}
	if inspection.Stale {
		inspection.Hints = append(inspection.Hints, "source is older than the configured stale threshold")
	}
	if inspection.MissingObjects > 0 {
		inspection.Hints = append(inspection.Hints, "source manifest references missing object files")
	}
	if inspection.ChangedObjects > 0 {
		inspection.Hints = append(inspection.Hints, "source manifest object hashes differ from local files")
	}
	return inspection, nil
}

func ledgerDiffSince(reader ledger.Reader, source model.Source, sinceID string, includeVerified bool) (ledgerDiff, error) {
	operations, err := reader.Operations()
	if err != nil {
		return ledgerDiff{}, err
	}
	seenSince := false
	diff := ledgerDiff{Source: ledger.SourceSpec(source), Since: sinceID}
	for _, operation := range operations {
		if !seenSince {
			if operation.ID == sinceID {
				seenSince = true
			}
			continue
		}
		var addedOperation bool
		for _, change := range operation.Changes {
			if change.Type == "verified" && !includeVerified {
				continue
			}
			if !sourceOwnsChange(source, change) {
				continue
			}
			if !addedOperation {
				diff.Operations = append(diff.Operations, operation.ID)
				addedOperation = true
			}
			diff.Changes = append(diff.Changes, ledgerDiffChange{
				OperationID: operation.ID,
				FinishedAt:  operation.FinishedAt,
				Type:        change.Type,
				Object:      change.Object,
				Number:      change.Number,
				ID:          change.ID,
				Path:        change.Path,
				SHA256:      change.SHA256,
			})
		}
	}
	if !seenSince {
		return ledgerDiff{}, fmt.Errorf("operation %q not found in operation sequence", sinceID)
	}
	return diff, nil
}

func sourceOwnsChange(source model.Source, change model.ObjectChange) bool {
	path := filepath.ToSlash(change.Path)
	return path == ledger.SourcePath(source) || strings.HasPrefix(path, ledger.SourceScopedPath(source)+"/")
}

func sortedCountKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func lastSourceOperationTime(source model.Source) (time.Time, bool) {
	var latest time.Time
	for _, operation := range source.Operations {
		if operation.StartedAt.IsZero() {
			continue
		}
		if latest.IsZero() || operation.StartedAt.After(latest) {
			latest = operation.StartedAt
		}
	}
	return latest, !latest.IsZero()
}

func parseStaleDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "0", "off", "none", "disabled":
		return 0, nil
	}
	if strings.HasSuffix(value, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(value, "d"))
		if err != nil || days < 0 {
			return 0, fmt.Errorf("invalid stale duration %q", value)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid stale duration %q: %w", value, err)
	}
	return duration, nil
}

func formatApproxDuration(duration time.Duration) string {
	if duration >= 24*time.Hour {
		days := int(duration / (24 * time.Hour))
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if duration >= time.Hour {
		hours := int(duration / time.Hour)
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	minutes := int(duration / time.Minute)
	if minutes <= 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func writeSourceDefaultOperation(root string, args []string, startedAt time.Time, source model.Source, includeLocal bool) error {
	finishedAt := time.Now().UTC()
	sum, err := fileSHA256(filepath.Join(root, "ledger.json"))
	if err != nil {
		return err
	}
	input := map[string]string{}
	command := "source default"
	if source.System == "" {
		command = "source default --clear"
	} else {
		input["source"] = ledger.SourceSpec(source)
	}
	operation := model.Operation{
		ID:         ledger.NewOperationID(command, startedAt),
		Command:    command,
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), includeLocal),
		Input:      input,
		Output: model.OperationOutput{
			Ledger:  root,
			Updated: 1,
		},
		Changes: []model.ObjectChange{
			{
				Type:   "updated",
				Object: "ledger",
				Path:   "ledger.json",
				SHA256: sum,
			},
		},
	}
	return (ledger.Writer{Root: root}).WriteOperation(operation)
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- checksum path comes from a source manifest object reference.
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func addLedgerMetadataChange(diff *ledger.Diff, root string, existed bool) error {
	sum, err := fileSHA256(filepath.Join(root, "ledger.json"))
	if err != nil {
		return err
	}
	changeType := "created"
	if existed {
		changeType = "updated"
	}
	if existed {
		diff.Updated++
	} else {
		diff.Created++
	}
	diff.Changes = append(diff.Changes, model.ObjectChange{
		Type:   changeType,
		Object: "ledger",
		Path:   "ledger.json",
		SHA256: sum,
	})
	return nil
}

func writeField(w io.Writer, label string, value any) {
	fmt.Fprintf(w, "%-14s %v\n", label, value)
}

func writeIndentedField(w io.Writer, label string, value any) {
	fmt.Fprintf(w, "  %-16s %v\n", label, value)
}

func writeIssueComments(w io.Writer, number int, comments []model.Comment) {
	fmt.Fprintln(w)
	writeField(w, "Issue", fmt.Sprintf("#%d", number))
	writeField(w, "Comments", len(comments))
	for _, comment := range comments {
		fmt.Fprintln(w)
		writeIndentedField(w, "Source", ledger.SourceSpec(comment.Source))
		writeIndentedField(w, "Author", comment.Author.Login)
		writeIndentedField(w, "Created", comment.CreatedAt.Format(time.RFC3339))
		writeIndentedField(w, "URL", comment.OriginalURL)
		if comment.Body != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, comment.Body)
		}
	}
}

func writePullRequestComments(w io.Writer, number int, comments []model.ReviewComment) {
	fmt.Fprintln(w)
	writeField(w, "Pull request", fmt.Sprintf("#%d", number))
	writeField(w, "Review comments", len(comments))
	for _, comment := range comments {
		fmt.Fprintln(w)
		writeIndentedField(w, "Source", ledger.SourceSpec(comment.Source))
		writeIndentedField(w, "Author", comment.Author.Login)
		writeIndentedField(w, "Path", comment.Path)
		writeIndentedField(w, "Line", comment.Line)
		writeIndentedField(w, "URL", comment.OriginalURL)
		if comment.Body != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, comment.Body)
		}
	}
}

type timelineEvent struct {
	Time   time.Time `json:"time"`
	Type   string    `json:"type"`
	Author string    `json:"author,omitempty"`
	Title  string    `json:"title,omitempty"`
	Body   string    `json:"body,omitempty"`
	URL    string    `json:"url,omitempty"`
	Path   string    `json:"path,omitempty"`
	Line   int       `json:"line,omitempty"`
}

func issueTimeline(issue model.Issue, comments []model.Comment) []timelineEvent {
	events := []timelineEvent{
		{
			Time:   issue.CreatedAt,
			Type:   "issue.opened",
			Author: issue.Author.Login,
			Title:  issue.Title,
			Body:   issue.Body,
			URL:    issue.OriginalURL,
		},
	}
	for _, comment := range comments {
		events = append(events, timelineEvent{
			Time:   comment.CreatedAt,
			Type:   "issue.comment",
			Author: comment.Author.Login,
			Body:   comment.Body,
			URL:    comment.OriginalURL,
		})
	}
	if !issue.ClosedAt.IsZero() {
		events = append(events, timelineEvent{
			Time:   issue.ClosedAt,
			Type:   "issue.closed",
			Author: issue.Author.Login,
			URL:    issue.OriginalURL,
		})
	}
	return sortTimeline(events)
}

func pullRequestTimeline(pr model.PullRequest, conversationComments []model.Comment, reviewComments []model.ReviewComment) []timelineEvent {
	events := []timelineEvent{
		{
			Time:   pr.CreatedAt,
			Type:   "pull_request.opened",
			Author: pr.Author.Login,
			Title:  pr.Title,
			Body:   pr.Body,
			URL:    pr.OriginalURL,
		},
	}
	for _, comment := range conversationComments {
		events = append(events, timelineEvent{
			Time:   comment.CreatedAt,
			Type:   "pull_request.comment",
			Author: comment.Author.Login,
			Body:   comment.Body,
			URL:    comment.OriginalURL,
		})
	}
	for _, comment := range reviewComments {
		events = append(events, timelineEvent{
			Time:   comment.CreatedAt,
			Type:   "pull_request.review_comment",
			Author: comment.Author.Login,
			Body:   comment.Body,
			URL:    comment.OriginalURL,
			Path:   comment.Path,
			Line:   comment.Line,
		})
	}
	if !pr.MergedAt.IsZero() {
		events = append(events, timelineEvent{
			Time:   pr.MergedAt,
			Type:   "pull_request.merged",
			Author: pr.Author.Login,
			URL:    pr.OriginalURL,
		})
	} else if !pr.ClosedAt.IsZero() {
		events = append(events, timelineEvent{
			Time:   pr.ClosedAt,
			Type:   "pull_request.closed",
			Author: pr.Author.Login,
			URL:    pr.OriginalURL,
		})
	}
	return sortTimeline(events)
}

func sortTimeline(events []timelineEvent) []timelineEvent {
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time.Equal(events[j].Time) {
			if events[i].Type == events[j].Type {
				return events[i].URL < events[j].URL
			}
			return events[i].Type < events[j].Type
		}
		return events[i].Time.Before(events[j].Time)
	})
	return events
}

func writeTimeline(w io.Writer, kind string, number int, source string, events []timelineEvent) {
	writeField(w, kind, fmt.Sprintf("#%d", number))
	writeField(w, "Source", source)
	writeField(w, "Events", len(events))
	for _, event := range events {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s  %s", event.Time.Format(time.RFC3339), event.Type)
		if event.Author != "" {
			fmt.Fprintf(w, "  %s", event.Author)
		}
		fmt.Fprintln(w)
		if event.Title != "" {
			writeIndentedField(w, "Title", event.Title)
		}
		if event.Path != "" {
			writeIndentedField(w, "Path", event.Path)
		}
		if event.Line > 0 {
			writeIndentedField(w, "Line", event.Line)
		}
		if event.URL != "" {
			writeIndentedField(w, "URL", event.URL)
		}
		if event.Body != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, event.Body)
		}
	}
}

func writeJSONOutput(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printImportProgress(w io.Writer, progress github.Progress, verbose bool) {
	if progress.Detail && !verbose {
		return
	}
	if progress.Detail {
		fmt.Fprintf(w, "  - %s...\n", progress.Message)
		return
	}
	fmt.Fprintf(w, "- %s...\n", progress.Message)
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

func gitConfig(key string) string {
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseRepo(value string) (string, string, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be owner/repo, got %q", value)
	}
	if _, err := ledger.ParseSourceSpec("github:" + value); err != nil {
		return "", "", fmt.Errorf("repository must be safe owner/repo, got %q", value)
	}
	return parts[0], parts[1], nil
}

func parseNumber(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("number must be a positive integer, got %q", value)
	}
	return number, nil
}

func parseOptionalSource(value string) (model.Source, bool, error) {
	if value == "" {
		return model.Source{}, false, nil
	}
	source, err := ledger.ParseSourceSpec(value)
	if err != nil {
		return model.Source{}, false, err
	}
	return source, true, nil
}

func resolveOptionalSource(reader ledger.Reader, value string) (model.Source, bool, error) {
	source, ok, err := parseOptionalSource(value)
	if err != nil || ok {
		return source, ok, err
	}
	current, err := reader.Ledger()
	if err != nil {
		return model.Source{}, false, nil
	}
	if current.DefaultSource == nil {
		return model.Source{}, false, nil
	}
	return *current.DefaultSource, true, nil
}

func resolveRefreshSources(reader ledger.Reader, requested []string) ([]model.Source, error) {
	if len(requested) == 0 {
		return reader.Sources()
	}
	var sources []model.Source
	seen := map[string]bool{}
	for _, value := range requested {
		source, err := ledger.ParseSourceSpec(value)
		if err != nil {
			return nil, err
		}
		source, err = reader.Source(source)
		if err != nil {
			return nil, err
		}
		spec := ledger.SourceSpec(source)
		if seen[spec] {
			continue
		}
		seen[spec] = true
		sources = append(sources, source)
	}
	return sources, nil
}

type valueListFlag []string

func (f *valueListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *valueListFlag) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			*f = append(*f, item)
		}
	}
	return nil
}

type searchFieldsFlag []string

func (f *searchFieldsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *searchFieldsFlag) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			*f = append(*f, item)
		}
	}
	return nil
}

func issueSearchFields() map[string]func(model.Issue) string {
	return map[string]func(model.Issue) string{
		"title":       func(issue model.Issue) string { return issue.Title },
		"description": func(issue model.Issue) string { return issue.Body },
		"body":        func(issue model.Issue) string { return issue.Body },
		"author":      func(issue model.Issue) string { return issue.Author.Login },
		"state":       func(issue model.Issue) string { return issue.State },
		"label":       func(issue model.Issue) string { return strings.Join(issue.Labels, " ") },
		"labels":      func(issue model.Issue) string { return strings.Join(issue.Labels, " ") },
		"milestone":   func(issue model.Issue) string { return issue.Milestone },
		"url":         func(issue model.Issue) string { return issue.OriginalURL },
	}
}

func pullRequestSearchFields() map[string]func(model.PullRequest) string {
	return map[string]func(model.PullRequest) string{
		"title":       func(pr model.PullRequest) string { return pr.Title },
		"description": func(pr model.PullRequest) string { return pr.Body },
		"body":        func(pr model.PullRequest) string { return pr.Body },
		"author":      func(pr model.PullRequest) string { return pr.Author.Login },
		"state":       func(pr model.PullRequest) string { return pr.State },
		"branch":      func(pr model.PullRequest) string { return pr.BaseRef + " " + pr.HeadRef },
		"base":        func(pr model.PullRequest) string { return pr.BaseRef },
		"head":        func(pr model.PullRequest) string { return pr.HeadRef },
		"url":         func(pr model.PullRequest) string { return pr.OriginalURL },
	}
}

func normalizeSearchFields[T any](fields []string, allowed map[string]func(T) string) ([]string, error) {
	if len(fields) == 0 {
		return []string{"title", "description"}, nil
	}
	var normalized []string
	seen := map[string]bool{}
	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if field == "all" {
			normalized = normalized[:0]
			for allowedField := range allowed {
				normalized = append(normalized, allowedField)
			}
			sort.Strings(normalized)
			return normalized, nil
		}
		if _, ok := allowed[field]; !ok {
			return nil, fmt.Errorf("unsupported search field %q", field)
		}
		if !seen[field] {
			seen[field] = true
			normalized = append(normalized, field)
		}
	}
	return normalized, nil
}

func searchIssues(issues []model.Issue, query string, fields []string) []model.Issue {
	query = strings.ToLower(query)
	searchable := issueSearchFields()
	var matches []model.Issue
	for _, issue := range issues {
		text := searchableText(issue, fields, searchable)
		if strings.Contains(text, query) {
			matches = append(matches, issue)
		}
	}
	return matches
}

func searchPullRequests(prs []model.PullRequest, query string, fields []string) []model.PullRequest {
	query = strings.ToLower(query)
	searchable := pullRequestSearchFields()
	var matches []model.PullRequest
	for _, pr := range prs {
		text := searchableText(pr, fields, searchable)
		if strings.Contains(text, query) {
			matches = append(matches, pr)
		}
	}
	return matches
}

func matchingIssueField(issue model.Issue, query string, fields []string) string {
	return matchingField(issue, query, fields, issueSearchFields())
}

func matchingPullRequestField(pr model.PullRequest, query string, fields []string) string {
	return matchingField(pr, query, fields, pullRequestSearchFields())
}

func matchingField[T any](value T, query string, fields []string, allowed map[string]func(T) string) string {
	query = strings.ToLower(query)
	for _, field := range fields {
		if read, ok := allowed[field]; ok && strings.Contains(strings.ToLower(read(value)), query) {
			if field == "body" {
				return "description"
			}
			return field
		}
	}
	return ""
}

func searchableText[T any](value T, fields []string, allowed map[string]func(T) string) string {
	var parts []string
	for _, field := range fields {
		if read, ok := allowed[field]; ok {
			parts = append(parts, read(value))
		}
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

func duplicateIssueNumbers(issues []model.Issue) []int {
	counts := map[int]int{}
	for _, issue := range issues {
		counts[issue.Number]++
	}
	return duplicateNumbers(counts)
}

func duplicatePullRequestNumbers(prs []model.PullRequest) []int {
	counts := map[int]int{}
	for _, pr := range prs {
		counts[pr.Number]++
	}
	return duplicateNumbers(counts)
}

func duplicateNumbers(counts map[int]int) []int {
	var duplicates []int
	for number, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, number)
		}
	}
	sort.Ints(duplicates)
	return duplicates
}

func normalizeImportArgs(args []string) ([]string, error) {
	var flags []string
	var repo string
	boolFlags := map[string]bool{
		"--plain-file-store": true,
		"-plain-file-store":  true,
		"--verbose":          true,
		"-verbose":           true,
		"--v":                true,
		"-v":                 true,
		"--local":            true,
		"-local":             true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if strings.Contains(arg, "=") {
				continue
			}
			if boolFlags[arg] {
				continue
			}
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags = append(flags, args[i])
			continue
		}
		if repo != "" {
			return nil, fmt.Errorf("unexpected extra argument %q", arg)
		}
		repo = arg
	}

	if repo == "" {
		return flags, nil
	}
	return append(flags, repo), nil
}

func normalizeSingleValueCommandArgs(args []string, valueName string, boolFlags map[string]bool) ([]string, error) {
	var flags []string
	var value string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if strings.Contains(arg, "=") || boolFlags[arg] {
				continue
			}
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags = append(flags, args[i])
			continue
		}
		if value != "" {
			return nil, fmt.Errorf("unexpected extra %s argument %q", valueName, arg)
		}
		value = arg
	}
	if value == "" {
		return flags, nil
	}
	return append(flags, value), nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone github auth login [flags]")
	fmt.Fprintln(w, "  waystone github auth logout [flags]")
	fmt.Fprintln(w, "  waystone github import [flags] owner/repo")
	fmt.Fprintln(w, "  waystone github refresh [flags] owner/repo")
	fmt.Fprintln(w, "  waystone issue list [flags]")
	fmt.Fprintln(w, "  waystone issue search [flags] <text>")
	fmt.Fprintln(w, "  waystone issue show [flags] <number>")
	fmt.Fprintln(w, "  waystone issue comments [flags] <number>")
	fmt.Fprintln(w, "  waystone issue timeline [flags] <number>")
	fmt.Fprintln(w, "  waystone label list [flags]")
	fmt.Fprintln(w, "  waystone milestone list [flags]")
	fmt.Fprintln(w, "  waystone ledger export [flags]")
	fmt.Fprintln(w, "  waystone ledger doctor [flags]")
	fmt.Fprintln(w, "  waystone ledger diff --source <source> --since <operation>")
	fmt.Fprintln(w, "  waystone ledger import [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger inspect [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger summary [flags]")
	fmt.Fprintln(w, "  waystone ledger status [flags]")
	fmt.Fprintln(w, "  waystone ledger history [flags]")
	fmt.Fprintln(w, "  waystone ledger show-operation [flags] <operation-id>")
	fmt.Fprintln(w, "  waystone ledger verify [flags]")
	fmt.Fprintln(w, "  waystone pr list [flags]")
	fmt.Fprintln(w, "  waystone pr search [flags] <text>")
	fmt.Fprintln(w, "  waystone pr show [flags] <number>")
	fmt.Fprintln(w, "  waystone pr comments [flags] <number>")
	fmt.Fprintln(w, "  waystone pr timeline [flags] <number>")
	fmt.Fprintln(w, "  waystone source default [flags] [source]")
	fmt.Fprintln(w, "  waystone source inspect [flags] <source>")
	fmt.Fprintln(w, "  waystone source list [flags]")
	fmt.Fprintln(w, "  waystone source refresh [flags]")
	fmt.Fprintln(w, "  waystone source show [flags] <source>")
	fmt.Fprintln(w, "  waystone source status [flags]")
	fmt.Fprintln(w, "  waystone version [flags]")
}

func printGitHubUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone github auth login [flags]")
	fmt.Fprintln(w, "  waystone github auth logout [flags]")
	fmt.Fprintln(w, "  waystone github import [flags] owner/repo")
	fmt.Fprintln(w, "  waystone github refresh [flags] owner/repo")
}

func printGitHubAuthUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone github auth login [flags]")
	fmt.Fprintln(w, "  waystone github auth logout [flags]")
}

func printIssueUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone issue list [flags]")
	fmt.Fprintln(w, "  waystone issue search [flags] <text>")
	fmt.Fprintln(w, "  waystone issue show [flags] <number>")
	fmt.Fprintln(w, "  waystone issue comments [flags] <number>")
	fmt.Fprintln(w, "  waystone issue timeline [flags] <number>")
}

func printLabelUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone label list [flags]")
}

func printMilestoneUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone milestone list [flags]")
}

func printLedgerUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone ledger doctor [flags]")
	fmt.Fprintln(w, "  waystone ledger diff --source <source> --since <operation>")
	fmt.Fprintln(w, "  waystone ledger export [flags]")
	fmt.Fprintln(w, "  waystone ledger import [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger inspect [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger summary [flags]")
	fmt.Fprintln(w, "  waystone ledger status [flags]")
	fmt.Fprintln(w, "  waystone ledger history [flags]")
	fmt.Fprintln(w, "  waystone ledger show-operation [flags] <operation-id>")
	fmt.Fprintln(w, "  waystone ledger verify [flags]")
}

func printSourceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone source default [flags] [source]")
	fmt.Fprintln(w, "  waystone source inspect [flags] <source>")
	fmt.Fprintln(w, "  waystone source list [flags]")
	fmt.Fprintln(w, "  waystone source refresh [flags]")
	fmt.Fprintln(w, "  waystone source show [flags] <source>")
	fmt.Fprintln(w, "  waystone source status [flags]")
}

func printPullRequestUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone pr list [flags]")
	fmt.Fprintln(w, "  waystone pr search [flags] <text>")
	fmt.Fprintln(w, "  waystone pr show [flags] <number>")
	fmt.Fprintln(w, "  waystone pr comments [flags] <number>")
	fmt.Fprintln(w, "  waystone pr timeline [flags] <number>")
}
