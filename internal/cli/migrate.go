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
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

const defaultMigrationNumberingStrategy = "preserve-source-numbering"
const migrationPlanVersion = "waystone.migration_plan.v1"

type migrationReport struct {
	From              string                      `json:"from"`
	To                string                      `json:"to"`
	Records           migrationRecordCounts       `json:"records"`
	Sources           []migrationSourceReport     `json:"sources,omitempty"`
	Identity          migrationIdentityReport     `json:"identity"`
	LocalContinuation migrationContinuationCounts `json:"local_continuation"`
	Warnings          []string                    `json:"warnings"`
}

type migrationSourceReport struct {
	Source  string                `json:"source"`
	Records migrationRecordCounts `json:"records"`
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
	case "inspect":
		return runMigrateInspect(args[1:], stdout)
	case "plan":
		return runMigratePlan(args[1:], stdout)
	case "report":
		return runMigrateReport(args[1:], stdout)
	case "verify":
		return runMigrateVerify(args[1:], stdout)
	default:
		printMigrateUsage(stderr)
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}

func runMigrateInspect(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone migrate inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	allowUnknown := fs.Bool("allow-unknown", false, "allow unknown migration plan versions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone migrate inspect [--allow-unknown] <plan>")
	}
	plan, err := readMigrationPlan(fs.Arg(0))
	if err != nil {
		return err
	}
	if err := validateMigrationPlan(plan, *allowUnknown); err != nil {
		return err
	}
	writeMigrationPlanInspection(stdout, plan)
	return nil
}

func runMigrateVerify(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone migrate verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone migrate verify <plan>")
	}
	plan, err := readMigrationPlan(fs.Arg(0))
	if err != nil {
		return err
	}
	if err := validateMigrationPlan(plan, false); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Migration plan verified")
	writeIndentedField(stdout, "Version", plan.Version)
	writeIndentedField(stdout, "Records", len(plan.Records))
	writeIndentedField(stdout, "Target writes", plan.Strategy.TargetWrite)
	return nil
}

func runMigrateReport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone migrate report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	toFlag := fs.String("to", "", "source to report to, e.g. waystone:owner/repo")
	strategyFlag := fs.String("numbering-strategy", defaultMigrationNumberingStrategy, "numbering strategy")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	var fromFlags valueListFlag
	fs.Var(&fromFlags, "from", "source to report from, e.g. github:owner/repo; repeatable or comma-separated")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || len(fromFlags) == 0 || *toFlag == "" {
		return errors.New("usage: waystone migrate report --from <source> --to <source>")
	}
	strategy, err := normalizeMigrationNumberingStrategy(*strategyFlag)
	if err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	from, err := parseMigrationSources(reader, fromFlags)
	if err != nil {
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
	toFlag := fs.String("to", "", "source to plan to, e.g. waystone:owner/repo")
	strategyFlag := fs.String("numbering-strategy", defaultMigrationNumberingStrategy, "numbering strategy")
	out := fs.String("out", "", "migration plan output path")
	var fromFlags valueListFlag
	fs.Var(&fromFlags, "from", "source to plan from, e.g. github:owner/repo; repeatable or comma-separated")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || len(fromFlags) == 0 || *toFlag == "" || *out == "" {
		return errors.New("usage: waystone migrate plan --from <source> --to <source> --numbering-strategy preserve-source-numbering --out <file>")
	}
	strategy, err := normalizeMigrationNumberingStrategy(*strategyFlag)
	if err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	from, err := parseMigrationSources(reader, fromFlags)
	if err != nil {
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

func parseMigrationSources(reader ledger.Reader, values []string) ([]model.Source, error) {
	sources := make([]model.Source, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
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

func buildMigrationReport(reader ledger.Reader, from []model.Source, to model.Source, strategy string) (migrationReport, error) {
	var records migrationRecordCounts
	sourceReports := make([]migrationSourceReport, 0, len(from))
	for _, source := range from {
		sourceRecords, err := sourceMigrationRecordCounts(reader, source)
		if err != nil {
			return migrationReport{}, err
		}
		records = addMigrationRecordCounts(records, sourceRecords)
		sourceReports = append(sourceReports, migrationSourceReport{
			Source:  ledger.SourceSpec(source),
			Records: sourceRecords,
		})
	}
	continuation, err := sourceMigrationContinuationCounts(reader, to)
	if err != nil {
		return migrationReport{}, err
	}
	warnings, err := migrationReportWarnings(reader, from)
	if err != nil {
		return migrationReport{}, err
	}
	return migrationReport{
		From:    migrationSourceSpecs(from),
		To:      ledger.SourceSpec(to),
		Records: records,
		Sources: sourceReports,
		Identity: migrationIdentityReport{
			SourceIDs: "preserved",
			TargetIDs: "not assigned",
			Strategy:  strategy,
		},
		LocalContinuation: continuation,
		Warnings:          warnings,
	}, nil
}

func addMigrationRecordCounts(a, b migrationRecordCounts) migrationRecordCounts {
	return migrationRecordCounts{
		Issues:         a.Issues + b.Issues,
		PullRequests:   a.PullRequests + b.PullRequests,
		Comments:       a.Comments + b.Comments,
		ReviewComments: a.ReviewComments + b.ReviewComments,
		Labels:         a.Labels + b.Labels,
		Milestones:     a.Milestones + b.Milestones,
		Releases:       a.Releases + b.Releases,
	}
}

func migrationSourceSpecs(sources []model.Source) string {
	specs := make([]string, 0, len(sources))
	for _, source := range sources {
		specs = append(specs, ledger.SourceSpec(source))
	}
	return strings.Join(specs, ", ")
}

func buildMigrationPlan(reader ledger.Reader, from []model.Source, to model.Source, numberingStrategy string, createdAt time.Time) (model.MigrationPlan, error) {
	var records []model.MigrationPlanRecord
	sources := make([]model.MigrationPlanSource, 0, len(from))
	for _, source := range from {
		sourceRecords, err := sourceMigrationPlanRecords(reader, source, to, numberingStrategy)
		if err != nil {
			return model.MigrationPlan{}, err
		}
		records = append(records, sourceRecords...)
		sources = append(sources, model.MigrationPlanSource{Source: ledger.SourceSpec(source)})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return migrationPlanRecordSortKey(records[i]) < migrationPlanRecordSortKey(records[j])
	})
	warnings, err := migrationReportWarnings(reader, from)
	if err != nil {
		return model.MigrationPlan{}, err
	}
	warnings = append(warnings,
		"Unsupported records are reported and not silently dropped",
		"Target writes are disabled",
	)
	return model.MigrationPlan{
		Version:     migrationPlanVersion,
		CreatedAt:   createdAt,
		ToolVersion: Version,
		From:        migrationSourceSpecs(from),
		Sources:     sources,
		To:          ledger.SourceSpec(to),
		Strategy:    defaultMigrationPlanStrategy(numberingStrategy),
		Records:     records,
		Warnings:    warnings,
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
		records = append(records, migrationPlanRecord("issue", from, issue.ID, issue.Number, issue.OriginalURL, issue.ID, to, numberingStrategy))
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
			records = append(records, migrationPlanRecord("comment", from, comment.ID, 0, comment.OriginalURL, comment.ID, to, numberingStrategy))
		}
	}
	for _, pullRequest := range pullRequests {
		comments, err := reader.SourceComments(from, pullRequest.Number)
		if err != nil {
			return nil, err
		}
		for _, comment := range comments {
			records = append(records, migrationPlanRecord("comment", from, comment.ID, 0, comment.OriginalURL, comment.ID, to, numberingStrategy))
		}
		records = append(records, migrationPlanRecord("pull_request", from, pullRequest.ID, pullRequest.Number, pullRequest.OriginalURL, pullRequest.ID, to, numberingStrategy))
		reviewComments, err := reader.SourceReviewComments(from, pullRequest.Number)
		if err != nil {
			return nil, err
		}
		for _, comment := range reviewComments {
			records = append(records, migrationPlanRecord("review_comment", from, comment.ID, 0, comment.OriginalURL, comment.ID, to, numberingStrategy))
		}
	}
	labels, err := reader.SourceLabels(from)
	if err != nil {
		return nil, err
	}
	for _, label := range labels {
		records = append(records, migrationPlanRecord("label", from, label.ID, 0, "", label.ID, to, numberingStrategy))
	}
	milestones, err := reader.SourceMilestones(from)
	if err != nil {
		return nil, err
	}
	for _, milestone := range milestones {
		records = append(records, migrationPlanRecord("milestone", from, milestone.ID, milestone.Number, milestone.OriginalURL, milestone.ID, to, numberingStrategy))
	}
	releases, err := reader.SourceReleases(from)
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		records = append(records, migrationPlanRecord("release", from, release.ID, 0, release.OriginalURL, release.ID, to, numberingStrategy))
	}
	return records, nil
}

func migrationPlanRecord(object string, from model.Source, sourceID string, sourceNumber int, sourceURL, waystoneID string, to model.Source, numberingStrategy string) model.MigrationPlanRecord {
	source := ledger.SourceSpec(from)
	return model.MigrationPlanRecord{
		Object:            object,
		Source:            source,
		SourceID:          sourceID,
		SourceNumber:      sourceNumber,
		SourceURL:         sourceURL,
		WaystoneID:        waystoneID,
		TargetSource:      ledger.SourceSpec(to),
		TargetKey:         migrationPlanTargetKey(source, object, sourceID, sourceNumber),
		NumberingStrategy: numberingStrategy,
	}
}

func migrationPlanTargetKey(source, object, sourceID string, sourceNumber int) string {
	if sourceNumber > 0 {
		return fmt.Sprintf("%s:%s:%d", source, object, sourceNumber)
	}
	return fmt.Sprintf("%s:%s:%s", source, object, sourceID)
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
		return fmt.Sprintf("%s:%s:%012d:%s", record.Source, prefix, record.SourceNumber, record.SourceID)
	}
	return record.Source + ":" + prefix + ":" + record.SourceID
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

func readMigrationPlan(path string) (model.MigrationPlan, error) {
	file, err := os.Open(path) // #nosec G304 -- path is an explicit user-provided migration plan path.
	if err != nil {
		return model.MigrationPlan{}, err
	}
	defer file.Close()
	var plan model.MigrationPlan
	if err := json.NewDecoder(file).Decode(&plan); err != nil {
		return model.MigrationPlan{}, err
	}
	return plan, nil
}

func validateMigrationPlan(plan model.MigrationPlan, allowUnknownVersion bool) error {
	if plan.Version != migrationPlanVersion && !allowUnknownVersion {
		return fmt.Errorf("unsupported migration plan version %q", plan.Version)
	}
	if strings.TrimSpace(plan.From) == "" {
		return errors.New("migration plan from is required")
	}
	if strings.TrimSpace(plan.To) == "" {
		return errors.New("migration plan to is required")
	}
	if err := validateMigrationPlanSources(plan.Sources); err != nil {
		return err
	}
	if err := validateMigrationPlanStrategy(plan.Strategy); err != nil {
		return err
	}
	seen := map[string]bool{}
	sources := migrationPlanSourceSet(plan.Sources)
	for i, record := range plan.Records {
		if err := validateMigrationPlanRecord(i, record, plan.Strategy, sources); err != nil {
			return err
		}
		key := record.Source + "\x00" + record.Object + "\x00" + record.SourceID
		if seen[key] {
			return fmt.Errorf("duplicate migration plan record for source %q object %q source_id %q", record.Source, record.Object, record.SourceID)
		}
		seen[key] = true
	}
	return nil
}

func validateMigrationPlanSources(sources []model.MigrationPlanSource) error {
	if len(sources) == 0 {
		return errors.New("migration plan sources are required")
	}
	seen := map[string]bool{}
	for i, source := range sources {
		if strings.TrimSpace(source.Source) == "" {
			return fmt.Errorf("source %d source is required", i)
		}
		if seen[source.Source] {
			return fmt.Errorf("duplicate migration plan source %q", source.Source)
		}
		seen[source.Source] = true
	}
	return nil
}

func migrationPlanSourceSet(sources []model.MigrationPlanSource) map[string]bool {
	set := map[string]bool{}
	for _, source := range sources {
		set[source.Source] = true
	}
	return set
}

func validateMigrationPlanStrategy(strategy model.MigrationPlanStrategy) error {
	expected := defaultMigrationPlanStrategy(defaultMigrationNumberingStrategy)
	checks := []struct {
		name string
		got  string
		want string
	}{
		{"numbering_strategy", strategy.Numbering, expected.Numbering},
		{"author_mapping_strategy", strategy.AuthorMapping, expected.AuthorMapping},
		{"label_mapping_strategy", strategy.LabelMapping, expected.LabelMapping},
		{"milestone_mapping_strategy", strategy.MilestoneMapping, expected.MilestoneMapping},
		{"state_mapping_strategy", strategy.StateMapping, expected.StateMapping},
		{"change_proposal_strategy", strategy.ChangeProposal, expected.ChangeProposal},
		{"timestamp_strategy", strategy.Timestamp, expected.Timestamp},
		{"collision_strategy", strategy.Collision, expected.Collision},
		{"attachment_strategy", strategy.Attachment, expected.Attachment},
		{"visibility_strategy", strategy.Visibility, expected.Visibility},
		{"comment_strategy", strategy.Comment, expected.Comment},
		{"unsupported_record_strategy", strategy.UnsupportedRecord, expected.UnsupportedRecord},
	}
	for _, check := range checks {
		if check.got != check.want {
			return fmt.Errorf("unsupported %s %q", check.name, check.got)
		}
	}
	if strategy.TargetWrite != "none" {
		return fmt.Errorf("target_write_strategy must be none, got %q", strategy.TargetWrite)
	}
	return nil
}

func validateMigrationPlanRecord(index int, record model.MigrationPlanRecord, strategy model.MigrationPlanStrategy, sources map[string]bool) error {
	if strings.TrimSpace(record.Object) == "" {
		return fmt.Errorf("record %d object is required", index)
	}
	if strings.TrimSpace(record.Source) == "" {
		return fmt.Errorf("record %d source is required", index)
	}
	if !sources[record.Source] {
		return fmt.Errorf("record %d source %q is not declared in plan sources", index, record.Source)
	}
	if strings.TrimSpace(record.SourceID) == "" {
		return fmt.Errorf("record %d source_id is required", index)
	}
	if strings.TrimSpace(record.WaystoneID) == "" {
		return fmt.Errorf("record %d waystone_id is required", index)
	}
	if strings.TrimSpace(record.TargetSource) == "" {
		return fmt.Errorf("record %d target_source is required", index)
	}
	if strings.TrimSpace(record.TargetKey) == "" {
		return fmt.Errorf("record %d target_key is required", index)
	}
	if record.NumberingStrategy != strategy.Numbering {
		return fmt.Errorf("record %d numbering_strategy %q does not match plan numbering_strategy %q", index, record.NumberingStrategy, strategy.Numbering)
	}
	if strategy.Numbering == defaultMigrationNumberingStrategy {
		expected := migrationPlanTargetKey(record.Source, record.Object, record.SourceID, record.SourceNumber)
		if record.TargetKey != expected {
			return fmt.Errorf("record %d target key %q does not match deterministic target key %q", index, record.TargetKey, expected)
		}
	}
	return nil
}

func writeMigrationPlanInspection(stdout io.Writer, plan model.MigrationPlan) {
	counts := migrationPlanRecordCounts(plan.Records)
	fmt.Fprintln(stdout, "Migration plan")
	writeIndentedField(stdout, "Version", plan.Version)
	writeIndentedField(stdout, "From", plan.From)
	writeIndentedField(stdout, "To", plan.To)
	writeIndentedField(stdout, "Records", len(plan.Records))
	writeIndentedField(stdout, "Target writes", plan.Strategy.TargetWrite)
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Sources")
	for _, source := range plan.Sources {
		fmt.Fprintln(stdout, source.Source)
	}
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Strategy")
	writeIndentedField(stdout, "Numbering", plan.Strategy.Numbering)
	writeIndentedField(stdout, "Author mapping", plan.Strategy.AuthorMapping)
	writeIndentedField(stdout, "Label mapping", plan.Strategy.LabelMapping)
	writeIndentedField(stdout, "Target writes", plan.Strategy.TargetWrite)
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Records")
	writeIndentedField(stdout, "Issues", counts.Issues)
	writeIndentedField(stdout, "Pull requests", counts.PullRequests)
	writeIndentedField(stdout, "Comments", counts.Comments)
	writeIndentedField(stdout, "Review comments", counts.ReviewComments)
	writeIndentedField(stdout, "Labels", counts.Labels)
	writeIndentedField(stdout, "Milestones", counts.Milestones)
	writeIndentedField(stdout, "Releases", counts.Releases)
	fmt.Fprintln(stdout)

	fmt.Fprintln(stdout, "Warnings")
	if len(plan.Warnings) == 0 {
		fmt.Fprintln(stdout, "- None")
		return
	}
	for _, warning := range plan.Warnings {
		fmt.Fprintf(stdout, "- %s\n", warning)
	}
}

func migrationPlanRecordCounts(records []model.MigrationPlanRecord) migrationRecordCounts {
	var counts migrationRecordCounts
	for _, record := range records {
		switch record.Object {
		case "issue":
			counts.Issues++
		case "pull_request":
			counts.PullRequests++
		case "comment":
			counts.Comments++
		case "review_comment":
			counts.ReviewComments++
		case "label":
			counts.Labels++
		case "milestone":
			counts.Milestones++
		case "release":
			counts.Releases++
		}
	}
	return counts
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

type migrationSourceFacts struct {
	Source             string
	IssueNumbers       []int
	PullRequestNumbers []int
	LabelNames         []string
	MilestoneTitles    []string
	AuthorLogins       []string
}

func migrationReportWarnings(reader ledger.Reader, sources []model.Source) ([]string, error) {
	warnings := []string{
		"Attachments are not yet represented",
		"Review thread semantics are only partially represented",
		"Users are not yet mapped to local identities",
		"CI history is not represented",
	}
	facts := make([]migrationSourceFacts, 0, len(sources))
	for _, source := range sources {
		sourceFacts, err := collectMigrationSourceFacts(reader, source)
		if err != nil {
			return nil, err
		}
		facts = append(facts, sourceFacts)
	}
	warnings = append(warnings, numberCollisionWarnings(facts, "issue")...)
	warnings = append(warnings, numberCollisionWarnings(facts, "pull request")...)
	warnings = append(warnings, nameOverlapWarnings(facts, "label")...)
	warnings = append(warnings, nameOverlapWarnings(facts, "milestone")...)
	warnings = append(warnings, authorAmbiguityWarnings(facts)...)
	return warnings, nil
}

func collectMigrationSourceFacts(reader ledger.Reader, source model.Source) (migrationSourceFacts, error) {
	facts := migrationSourceFacts{Source: ledger.SourceSpec(source)}
	issues, err := reader.SourceIssues(source)
	if err != nil {
		return facts, err
	}
	pullRequests, err := reader.SourcePullRequests(source)
	if err != nil {
		return facts, err
	}
	labels, err := reader.SourceLabels(source)
	if err != nil {
		return facts, err
	}
	milestones, err := reader.SourceMilestones(source)
	if err != nil {
		return facts, err
	}
	authorSet := map[string]bool{}
	for _, issue := range issues {
		facts.IssueNumbers = append(facts.IssueNumbers, issue.Number)
		addAuthorLogin(authorSet, issue.Author.Login)
	}
	for _, pullRequest := range pullRequests {
		facts.PullRequestNumbers = append(facts.PullRequestNumbers, pullRequest.Number)
		addAuthorLogin(authorSet, pullRequest.Author.Login)
	}
	numbers := map[int]bool{}
	for _, issue := range issues {
		numbers[issue.Number] = true
	}
	for _, pullRequest := range pullRequests {
		numbers[pullRequest.Number] = true
	}
	for number := range numbers {
		comments, err := reader.SourceComments(source, number)
		if err != nil {
			return facts, err
		}
		for _, comment := range comments {
			addAuthorLogin(authorSet, comment.Author.Login)
		}
	}
	for _, pullRequest := range pullRequests {
		reviewComments, err := reader.SourceReviewComments(source, pullRequest.Number)
		if err != nil {
			return facts, err
		}
		for _, comment := range reviewComments {
			addAuthorLogin(authorSet, comment.Author.Login)
		}
	}
	for _, label := range labels {
		if label.Name != "" {
			facts.LabelNames = append(facts.LabelNames, label.Name)
		}
	}
	for _, milestone := range milestones {
		if milestone.Title != "" {
			facts.MilestoneTitles = append(facts.MilestoneTitles, milestone.Title)
		}
	}
	for login := range authorSet {
		facts.AuthorLogins = append(facts.AuthorLogins, login)
	}
	sort.Ints(facts.IssueNumbers)
	sort.Ints(facts.PullRequestNumbers)
	sort.Strings(facts.LabelNames)
	sort.Strings(facts.MilestoneTitles)
	sort.Strings(facts.AuthorLogins)
	return facts, nil
}

func addAuthorLogin(authors map[string]bool, login string) {
	login = strings.TrimSpace(login)
	if login != "" {
		authors[login] = true
	}
}

func numberCollisionWarnings(facts []migrationSourceFacts, object string) []string {
	seen := map[int]string{}
	var warnings []string
	for _, sourceFacts := range facts {
		var numbers []int
		switch object {
		case "issue":
			numbers = sourceFacts.IssueNumbers
		case "pull request":
			numbers = sourceFacts.PullRequestNumbers
		}
		sourceSeen := map[int]bool{}
		for _, number := range numbers {
			if sourceSeen[number] {
				continue
			}
			sourceSeen[number] = true
			if previous, ok := seen[number]; ok && previous != sourceFacts.Source {
				warnings = append(warnings, fmt.Sprintf("Number collision: %s #%d appears in %s and %s", object, number, previous, sourceFacts.Source))
				continue
			}
			seen[number] = sourceFacts.Source
		}
	}
	return warnings
}

func nameOverlapWarnings(facts []migrationSourceFacts, object string) []string {
	seen := map[string]namedMigrationValue{}
	var warnings []string
	for _, sourceFacts := range facts {
		var names []string
		switch object {
		case "label":
			names = sourceFacts.LabelNames
		case "milestone":
			names = sourceFacts.MilestoneTitles
		}
		sourceSeen := map[string]bool{}
		for _, name := range names {
			key := strings.ToLower(name)
			if sourceSeen[key] {
				continue
			}
			sourceSeen[key] = true
			if previous, ok := seen[key]; ok && previous.Source != sourceFacts.Source {
				noun := object + " name"
				if object == "milestone" {
					noun = "milestone title"
				}
				warnings = append(warnings, fmt.Sprintf("%s overlap: %q appears in %s and %s", titleCase(noun), previous.Name, previous.Source, sourceFacts.Source))
				continue
			}
			seen[key] = namedMigrationValue{Name: name, Source: sourceFacts.Source}
		}
	}
	return warnings
}

type namedMigrationValue struct {
	Name   string
	Source string
}

func authorAmbiguityWarnings(facts []migrationSourceFacts) []string {
	seen := map[string]namedMigrationValue{}
	var warnings []string
	for _, sourceFacts := range facts {
		sourceSeen := map[string]bool{}
		for _, login := range sourceFacts.AuthorLogins {
			key := strings.ToLower(login)
			if sourceSeen[key] {
				continue
			}
			sourceSeen[key] = true
			if previous, ok := seen[key]; ok && previous.Source != sourceFacts.Source {
				warnings = append(warnings, fmt.Sprintf("Author identity ambiguity: %q appears in %s and %s", previous.Name, previous.Source, sourceFacts.Source))
				continue
			}
			seen[key] = namedMigrationValue{Name: login, Source: sourceFacts.Source}
		}
	}
	return warnings
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
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

	if len(report.Sources) > 1 {
		fmt.Fprintln(stdout, "Sources")
		for _, source := range report.Sources {
			fmt.Fprintln(stdout, source.Source)
			writeIndentedField(stdout, "Issues", source.Records.Issues)
			writeIndentedField(stdout, "Pull requests", source.Records.PullRequests)
			writeIndentedField(stdout, "Comments", source.Records.Comments)
			writeIndentedField(stdout, "Review comments", source.Records.ReviewComments)
			writeIndentedField(stdout, "Labels", source.Records.Labels)
			writeIndentedField(stdout, "Milestones", source.Records.Milestones)
			writeIndentedField(stdout, "Releases", source.Records.Releases)
		}
		fmt.Fprintln(stdout)
	}

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
