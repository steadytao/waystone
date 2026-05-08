// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

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
		return errors.New("usage: waystone pr show [--source <source>] [--with-comments] [--json] <number>")
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
		return errors.New("usage: waystone pr comments [--source <source>] [--json] <number>")
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
		return errors.New("usage: waystone pr timeline [--source <source>] [--json] <number>")
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
		return errors.New("usage: waystone pr search [--source <source>] [--field <field>] <text>")
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
