// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func runIssue(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printIssueUsage(stderr)
		return errors.New("missing issue command")
	}

	switch args[0] {
	case "create":
		return runIssueCreate(args[1:], stdout)
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

func runIssueCreate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone issue create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "source for the local issue, e.g. owner/repo or waystone:owner/repo")
	title := fs.String("title", "", "issue title")
	body := fs.String("body", "", "issue body")
	bodyFile := fs.String("body-file", "", "file containing issue body")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone issue create --source owner/repo --title <title> [--body <body> | --body-file <path>]")
	}
	if strings.TrimSpace(*sourceFlag) == "" {
		return errors.New("issue create requires --source owner/repo")
	}
	if strings.TrimSpace(*title) == "" {
		return errors.New("issue create requires --title")
	}
	if *body != "" && *bodyFile != "" {
		return errors.New("use --body or --body-file, not both")
	}
	source, err := parseLocalIssueSource(*sourceFlag)
	if err != nil {
		return err
	}
	if source.System != "waystone" {
		return fmt.Errorf("issue create only supports waystone sources, got %s", ledger.SourceSpec(source))
	}
	issueBody := *body
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile) // #nosec G304 -- body-file is an explicit user-provided input path.
		if err != nil {
			return err
		}
		issueBody = string(data)
	}

	startedAt := time.Now().UTC()
	reader := ledger.Reader{Root: *root}
	manifestSource := source
	if current, err := reader.Source(source); err == nil {
		manifestSource = current
	} else if !os.IsNotExist(err) {
		return err
	}
	number, err := nextIssueNumber(reader, source)
	if err != nil {
		return err
	}
	operationID := ledger.NewOperationID("issue create", startedAt)
	manifestSource.Operations = append(manifestSource.Operations, model.SourceOperationRef{
		ID:        operationID,
		Command:   "issue create",
		Path:      ledger.OperationPath(operationID),
		StartedAt: startedAt,
	})
	issue := localIssue(manifestSource, number, *title, issueBody, startedAt)
	writer := ledger.Writer{Root: *root}
	diff, err := writer.DiffLocalIssue(issue)
	if err != nil {
		return err
	}
	ledgerExisted := fileExists(filepath.Join(*root, "ledger.json"))
	if err := writer.WriteLocalIssue(issue); err != nil {
		return err
	}
	if err := addLedgerMetadataChange(&diff, *root, ledgerExisted); err != nil {
		return err
	}
	finishedAt := time.Now().UTC()
	operation := model.Operation{
		ID:         operationID,
		Command:    "issue create",
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), *includeLocal),
		Input:      map[string]string{"source": ledger.SourceSpec(source)},
		Output: model.OperationOutput{
			Ledger:  *root,
			Created: diff.Created,
			Updated: diff.Updated,
			Summary: model.RecordSummary{Issues: 1},
		},
		Changes: diff.Changes,
	}
	if err := writer.WriteOperation(operation); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Issue created")
	writeIndentedField(stdout, "Source", ledger.SourceSpec(source))
	writeIndentedField(stdout, "Number", fmt.Sprintf("#%d", issue.Number))
	writeIndentedField(stdout, "Title", issue.Title)
	return nil
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

func nextIssueNumber(reader ledger.Reader, source model.Source) (int, error) {
	issues, err := reader.SourceIssues(source)
	if err != nil {
		return 0, err
	}
	next := 1
	for _, issue := range issues {
		if issue.Number >= next {
			next = issue.Number + 1
		}
	}
	return next, nil
}

func parseLocalIssueSource(value string) (model.Source, error) {
	if !strings.Contains(value, ":") && strings.Count(value, "/") == 1 {
		value = "waystone:" + value
	}
	return ledger.ParseSourceSpec(value)
}

func localIssue(source model.Source, number int, title, body string, createdAt time.Time) model.Issue {
	source.URL = ""
	id := fmt.Sprintf("waystone:issue:%s/%s:%d", source.Owner, source.Repo, number)
	return model.Issue{
		Provenance: model.Provenance{
			ImportID: ledger.SourceSpec(source),
			Source:   source,
		},
		ID:        id,
		Number:    number,
		Title:     title,
		Body:      body,
		State:     "open",
		Author:    model.Author{Login: "local"},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
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
	if issue.OriginalURL != "" {
		writeField(stdout, "URL", issue.OriginalURL)
	}
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
