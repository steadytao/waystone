// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

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
