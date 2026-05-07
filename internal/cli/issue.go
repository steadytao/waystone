// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

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
