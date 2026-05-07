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
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/github"
	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

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
	signatures := fs.Bool("signatures", false, "verify operation record signatures")
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
	var signatureVerification ledger.OperationSignatureVerification
	var sourceSignatureVerification ledger.SourceSignatureVerification
	if *signatures {
		signatureVerification, err = reader.VerifyOperationSignatures()
		if err != nil {
			return err
		}
		sourceSignatureVerification, err = reader.VerifySourceSignatures()
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
	if *signatures {
		command += " --signatures"
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
			"ledger":            *root,
			"files":             verification.Files,
			"checksum":          verification.Checksum,
			"operations":        operationVerification,
			"signatures":        signatureVerification,
			"source_signatures": sourceSignatureVerification,
			"operation":         operation.ID,
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
	if *signatures {
		writeField(stdout, "Signatures", signatureVerification.Operations)
		writeField(stdout, "Valid signatures", signatureVerification.Valid)
		writeField(stdout, "Trusted signatures", signatureVerification.Trusted)
		writeField(stdout, "Untrusted signatures", signatureVerification.Untrusted)
		writeField(stdout, "Unsigned", signatureVerification.Unsigned)
		writeField(stdout, "Source signatures", sourceSignatureVerification.Sources)
		writeField(stdout, "Valid source signatures", sourceSignatureVerification.Valid)
		writeField(stdout, "Trusted source signatures", sourceSignatureVerification.Trusted)
		writeField(stdout, "Untrusted source signatures", sourceSignatureVerification.Untrusted)
		writeField(stdout, "Unsigned sources", sourceSignatureVerification.Unsigned)
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
	writeField(stdout, "Manifest", inspection.Manifest)
	writeField(stdout, "Signed", inspection.Signed)
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
