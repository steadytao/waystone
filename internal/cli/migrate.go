// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

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
	case "plan":
		return runMigratePlan(args[1:], stdout)
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
	strategyFlag := fs.String("numbering-strategy", defaultMigrationNumberingStrategy, "numbering strategy")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *fromFlag == "" || *toFlag == "" {
		return errors.New("usage: waystone migrate report --from <source> --to <source>")
	}
	strategy, err := normalizeMigrationNumberingStrategy(*strategyFlag)
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

func runMigratePlan(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone migrate plan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	fromFlag := fs.String("from", "", "source to plan from, e.g. github:owner/repo")
	toFlag := fs.String("to", "", "source to plan to, e.g. waystone:owner/repo")
	strategyFlag := fs.String("numbering-strategy", defaultMigrationNumberingStrategy, "numbering strategy")
	out := fs.String("out", "", "migration plan output path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *fromFlag == "" || *toFlag == "" || *out == "" {
		return errors.New("usage: waystone migrate plan --from <source> --to <source> --numbering-strategy preserve-source-numbering --out <file>")
	}
	strategy, err := normalizeMigrationNumberingStrategy(*strategyFlag)
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
	plan, err := buildMigrationPlan(reader, from, to, strategy, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := writeMigrationPlan(*out, plan); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Migration plan written")
	writeIndentedField(stdout, "From", plan.From)
	writeIndentedField(stdout, "To", plan.To)
	writeIndentedField(stdout, "Records", len(plan.Records))
	writeIndentedField(stdout, "Output", *out)
	return nil
}

func normalizeMigrationNumberingStrategy(value string) (string, error) {
	if value == defaultMigrationNumberingStrategy {
		return value, nil
	}
	return "", fmt.Errorf("only preserve-source-numbering is supported for migration numbering, got %q", value)
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

func buildMigrationPlan(reader ledger.Reader, from, to model.Source, numberingStrategy string, createdAt time.Time) (model.MigrationPlan, error) {
	records, err := sourceMigrationPlanRecords(reader, from, to, numberingStrategy)
	if err != nil {
		return model.MigrationPlan{}, err
	}
	return model.MigrationPlan{
		Version:     "waystone.migration_plan.v1",
		CreatedAt:   createdAt,
		ToolVersion: Version,
		From:        ledger.SourceSpec(from),
		To:          ledger.SourceSpec(to),
		Strategy:    defaultMigrationPlanStrategy(numberingStrategy),
		Records:     records,
		Warnings: []string{
			"Attachments are recorded as link-only evidence",
			"Unsupported records are reported and not silently dropped",
			"Target writes are disabled",
		},
	}, nil
}

func defaultMigrationPlanStrategy(numberingStrategy string) model.MigrationPlanStrategy {
	return model.MigrationPlanStrategy{
		Numbering:         numberingStrategy,
		AuthorMapping:     "preserve-source-author",
		LabelMapping:      "preserve-source-labels",
		MilestoneMapping:  "preserve-source-milestones",
		StateMapping:      "preserve",
		ChangeProposal:    "preserve-source-term",
		Timestamp:         "preserve-source-time",
		Collision:         "fail",
		Attachment:        "link-only",
		Visibility:        "preserve-where-supported",
		Comment:           "preserve-order",
		UnsupportedRecord: "report",
		TargetWrite:       "none",
	}
}

func sourceMigrationPlanRecords(reader ledger.Reader, from, to model.Source, numberingStrategy string) ([]model.MigrationPlanRecord, error) {
	var records []model.MigrationPlanRecord
	issues, err := reader.SourceIssues(from)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		records = append(records, migrationPlanRecord("issue", issue.ID, issue.Number, issue.OriginalURL, issue.ID, to, numberingStrategy))
	}
	pullRequests, err := reader.SourcePullRequests(from)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		comments, err := reader.SourceComments(from, issue.Number)
		if err != nil {
			return nil, err
		}
		for _, comment := range comments {
			records = append(records, migrationPlanRecord("comment", comment.ID, 0, comment.OriginalURL, comment.ID, to, numberingStrategy))
		}
	}
	for _, pullRequest := range pullRequests {
		comments, err := reader.SourceComments(from, pullRequest.Number)
		if err != nil {
			return nil, err
		}
		for _, comment := range comments {
			records = append(records, migrationPlanRecord("comment", comment.ID, 0, comment.OriginalURL, comment.ID, to, numberingStrategy))
		}
		records = append(records, migrationPlanRecord("pull_request", pullRequest.ID, pullRequest.Number, pullRequest.OriginalURL, pullRequest.ID, to, numberingStrategy))
		reviewComments, err := reader.SourceReviewComments(from, pullRequest.Number)
		if err != nil {
			return nil, err
		}
		for _, comment := range reviewComments {
			records = append(records, migrationPlanRecord("review_comment", comment.ID, 0, comment.OriginalURL, comment.ID, to, numberingStrategy))
		}
	}
	labels, err := reader.SourceLabels(from)
	if err != nil {
		return nil, err
	}
	for _, label := range labels {
		records = append(records, migrationPlanRecord("label", label.ID, 0, "", label.ID, to, numberingStrategy))
	}
	milestones, err := reader.SourceMilestones(from)
	if err != nil {
		return nil, err
	}
	for _, milestone := range milestones {
		records = append(records, migrationPlanRecord("milestone", milestone.ID, milestone.Number, milestone.OriginalURL, milestone.ID, to, numberingStrategy))
	}
	releases, err := reader.SourceReleases(from)
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		records = append(records, migrationPlanRecord("release", release.ID, 0, release.OriginalURL, release.ID, to, numberingStrategy))
	}
	sort.SliceStable(records, func(i, j int) bool {
		return migrationPlanRecordSortKey(records[i]) < migrationPlanRecordSortKey(records[j])
	})
	return records, nil
}

func migrationPlanRecord(object, sourceID string, sourceNumber int, sourceURL, waystoneID string, to model.Source, numberingStrategy string) model.MigrationPlanRecord {
	return model.MigrationPlanRecord{
		Object:            object,
		SourceID:          sourceID,
		SourceNumber:      sourceNumber,
		SourceURL:         sourceURL,
		WaystoneID:        waystoneID,
		TargetSource:      ledger.SourceSpec(to),
		TargetKey:         migrationPlanTargetKey(object, sourceID, sourceNumber),
		NumberingStrategy: numberingStrategy,
	}
}

func migrationPlanTargetKey(object, sourceID string, sourceNumber int) string {
	if sourceNumber > 0 {
		return fmt.Sprintf("%s:%d", object, sourceNumber)
	}
	return object + ":" + sourceID
}

func migrationPlanRecordSortKey(record model.MigrationPlanRecord) string {
	order := map[string]string{
		"issue":          "01",
		"comment":        "02",
		"pull_request":   "03",
		"review_comment": "04",
		"label":          "05",
		"milestone":      "06",
		"release":        "07",
	}
	prefix := order[record.Object]
	if prefix == "" {
		prefix = "99"
	}
	if record.SourceNumber > 0 {
		return fmt.Sprintf("%s:%012d:%s", prefix, record.SourceNumber, record.SourceID)
	}
	return prefix + ":" + record.SourceID
}

func writeMigrationPlan(path string, plan model.MigrationPlan) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.Create(path) // #nosec G304 -- output path is an explicit user-provided migration plan path.
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(plan)
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
