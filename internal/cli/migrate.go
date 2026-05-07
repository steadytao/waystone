// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

const defaultMigrationNumberingStrategy = "preserve-source-numbering"

type migrationReport struct {
	From              string                      `json:"from"`
	To                string                      `json:"to"`
	Records           migrationRecordCounts       `json:"records"`
	Identity          migrationIdentityReport     `json:"identity"`
	LocalContinuation migrationContinuationCounts `json:"local_continuation"`
	Warnings          []string                    `json:"warnings"`
}

type migrationRecordCounts struct {
	Issues         int `json:"issues"`
	PullRequests   int `json:"pull_requests"`
	Comments       int `json:"comments"`
	ReviewComments int `json:"review_comments"`
	Labels         int `json:"labels"`
	Milestones     int `json:"milestones"`
	Releases       int `json:"releases"`
}

type migrationIdentityReport struct {
	SourceIDs string `json:"source_ids"`
	TargetIDs string `json:"target_ids"`
	Strategy  string `json:"strategy"`
}

type migrationContinuationCounts struct {
	LocalIssues int `json:"local_issues"`
	LocalLabels int `json:"local_labels"`
	LocalEvents int `json:"local_events"`
}

func runMigrate(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printMigrateUsage(stderr)
		return errors.New("missing migrate command")
	}

	switch args[0] {
	case "report":
		return runMigrateReport(args[1:], stdout)
	default:
		printMigrateUsage(stderr)
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}

func runMigrateReport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone migrate report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	fromFlag := fs.String("from", "", "source to report from, e.g. github:owner/repo")
	toFlag := fs.String("to", "", "source to report to, e.g. waystone:owner/repo")
	strategyFlag := fs.String("strategy", defaultMigrationNumberingStrategy, "numbering strategy")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *fromFlag == "" || *toFlag == "" {
		return errors.New("usage: waystone migrate report --from <source> --to <source>")
	}
	strategy, err := normalizeMigrationStrategy(*strategyFlag)
	if err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	from, err := ledger.ParseSourceSpec(*fromFlag)
	if err != nil {
		return err
	}
	if _, err := reader.Source(from); err != nil {
		return err
	}
	to, err := ledger.ParseSourceSpec(*toFlag)
	if err != nil {
		return err
	}
	report, err := buildMigrationReport(reader, from, to, strategy)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, report)
	}
	writeMigrationReport(stdout, report)
	return nil
}

func normalizeMigrationStrategy(value string) (string, error) {
	switch value {
	case "preserve-source-numbering", "chronological-renumber", "source-priority-renumber":
		return value, nil
	default:
		return "", fmt.Errorf("unsupported migration numbering strategy %q", value)
	}
}

func buildMigrationReport(reader ledger.Reader, from, to model.Source, strategy string) (migrationReport, error) {
	records, err := sourceMigrationRecordCounts(reader, from)
	if err != nil {
		return migrationReport{}, err
	}
	continuation, err := sourceMigrationContinuationCounts(reader, to)
	if err != nil {
		return migrationReport{}, err
	}
	return migrationReport{
		From:    ledger.SourceSpec(from),
		To:      ledger.SourceSpec(to),
		Records: records,
		Identity: migrationIdentityReport{
			SourceIDs: "preserved",
			TargetIDs: "not assigned",
			Strategy:  strategy,
		},
		LocalContinuation: continuation,
		Warnings: []string{
			"Attachments are not yet represented",
			"Review thread semantics are only partially represented",
			"Users are not yet mapped to local identities",
			"CI history is not represented",
		},
	}, nil
}

func sourceMigrationRecordCounts(reader ledger.Reader, source model.Source) (migrationRecordCounts, error) {
	issues, err := reader.SourceIssues(source)
	if err != nil {
		return migrationRecordCounts{}, err
	}
	pullRequests, err := reader.SourcePullRequests(source)
	if err != nil {
		return migrationRecordCounts{}, err
	}
	comments, err := sourceConversationCommentCount(reader, source, issues, pullRequests)
	if err != nil {
		return migrationRecordCounts{}, err
	}
	reviewComments, err := sourceReviewCommentCount(reader, source, pullRequests)
	if err != nil {
		return migrationRecordCounts{}, err
	}
	labels, err := reader.SourceLabels(source)
	if err != nil {
		return migrationRecordCounts{}, err
	}
	milestones, err := reader.SourceMilestones(source)
	if err != nil {
		return migrationRecordCounts{}, err
	}
	releases, err := reader.SourceReleases(source)
	if err != nil {
		return migrationRecordCounts{}, err
	}
	return migrationRecordCounts{
		Issues:         len(issues),
		PullRequests:   len(pullRequests),
		Comments:       comments,
		ReviewComments: reviewComments,
		Labels:         len(labels),
		Milestones:     len(milestones),
		Releases:       len(releases),
	}, nil
}

func sourceMigrationContinuationCounts(reader ledger.Reader, source model.Source) (migrationContinuationCounts, error) {
	if source.System != "waystone" {
		return migrationContinuationCounts{}, nil
	}
	if _, err := reader.Source(source); err != nil {
		if os.IsNotExist(err) {
			return migrationContinuationCounts{}, nil
		}
		return migrationContinuationCounts{}, err
	}
	issues, err := reader.SourceIssues(source)
	if err != nil {
		return migrationContinuationCounts{}, err
	}
	labels, err := reader.SourceLabels(source)
	if err != nil {
		return migrationContinuationCounts{}, err
	}
	var eventCount int
	for _, issue := range issues {
		events, err := reader.SourceIssueEvents(source, issue.Number)
		if err != nil {
			return migrationContinuationCounts{}, err
		}
		eventCount += len(events)
	}
	return migrationContinuationCounts{
		LocalIssues: len(issues),
		LocalLabels: len(labels),
		LocalEvents: eventCount,
	}, nil
}

func sourceConversationCommentCount(reader ledger.Reader, source model.Source, issues []model.Issue, pullRequests []model.PullRequest) (int, error) {
	numbers := map[int]bool{}
	for _, issue := range issues {
		numbers[issue.Number] = true
	}
	for _, pullRequest := range pullRequests {
		numbers[pullRequest.Number] = true
	}
	var count int
	for number := range numbers {
		comments, err := reader.SourceComments(source, number)
		if err != nil {
			return 0, err
		}
		count += len(comments)
	}
	return count, nil
}

func sourceReviewCommentCount(reader ledger.Reader, source model.Source, pullRequests []model.PullRequest) (int, error) {
	var count int
	for _, pullRequest := range pullRequests {
		comments, err := reader.SourceReviewComments(source, pullRequest.Number)
		if err != nil {
			return 0, err
		}
		count += len(comments)
	}
	return count, nil
}

func writeMigrationReport(stdout io.Writer, report migrationReport) {
	fmt.Fprintln(stdout, "Migration report")
	writeIndentedField(stdout, "From", report.From)
	writeIndentedField(stdout, "To", report.To)
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Records")
	writeIndentedField(stdout, "Issues", report.Records.Issues)
	writeIndentedField(stdout, "Pull requests", report.Records.PullRequests)
	writeIndentedField(stdout, "Comments", report.Records.Comments)
	writeIndentedField(stdout, "Review comments", report.Records.ReviewComments)
	writeIndentedField(stdout, "Labels", report.Records.Labels)
	writeIndentedField(stdout, "Milestones", report.Records.Milestones)
	writeIndentedField(stdout, "Releases", report.Records.Releases)
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Identity")
	writeIndentedField(stdout, "Source IDs", report.Identity.SourceIDs)
	writeIndentedField(stdout, "Target IDs", report.Identity.TargetIDs)
	writeIndentedField(stdout, "Strategy", report.Identity.Strategy)
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Local continuation")
	writeIndentedField(stdout, "Local issues", report.LocalContinuation.LocalIssues)
	writeIndentedField(stdout, "Local labels", report.LocalContinuation.LocalLabels)
	writeIndentedField(stdout, "Local events", report.LocalContinuation.LocalEvents)
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Warnings")
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "- %s\n", warning)
	}
}
