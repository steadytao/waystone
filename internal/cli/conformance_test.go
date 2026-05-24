// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steadytao/waystone/internal/ledger"
)

func TestConformanceFixtureLedgersVerify(t *testing.T) {
	for _, name := range []string{
		"single-github-ledger",
		"multi-source-ledger",
		"local-labelled-issues-ledger",
	} {
		t.Run(name, func(t *testing.T) {
			reader := ledger.Reader{Root: filepath.Join("..", "..", "testdata", "conformance", name, ".waystone")}
			if _, err := reader.Verify(); err != nil {
				t.Fatalf("fixture verify returned error: %v", err)
			}
			if _, err := reader.VerifyOperations(); err != nil {
				t.Fatalf("fixture operation verify returned error: %v", err)
			}
		})
	}
}

func TestConformanceFixturesDoNotContainZeroOptionalTimestamps(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "conformance")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("0001-01-01T00:00:00Z")) {
			t.Fatalf("%s contains zero optional timestamp sentinel", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk conformance fixtures: %v", err)
	}
}

func TestConformanceSingleGitHubFixtureCoversMigrationCommands(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "conformance", "single-github-ledger", ".waystone")
	source := "github:example/project"
	target := "waystone:example/project"

	var reportOut bytes.Buffer
	if err := Run(context.Background(), []string{"migrate", "report", "--ledger", root, "--from", source, "--to", target, "--json"}, &reportOut, io.Discard); err != nil {
		t.Fatalf("migrate report returned error: %v", err)
	}
	var report migrationReport
	if err := json.Unmarshal(reportOut.Bytes(), &report); err != nil {
		t.Fatalf("decode migration report: %v", err)
	}
	if report.Records.Issues != 1 || report.Records.Labels != 1 || report.Records.Milestones != 1 {
		t.Fatalf("report records = %#v, want one issue, label and milestone", report.Records)
	}

	planPath := filepath.Join(t.TempDir(), "migration-plan.json")
	if err := Run(context.Background(), []string{"migrate", "plan", "--ledger", root, "--from", source, "--to", target, "--out", planPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("migrate plan returned error: %v", err)
	}
	if err := Run(context.Background(), []string{"migrate", "verify", planPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("migrate verify returned error: %v", err)
	}
	var inspectOut bytes.Buffer
	if err := Run(context.Background(), []string{"migrate", "inspect", planPath}, &inspectOut, io.Discard); err != nil {
		t.Fatalf("migrate inspect returned error: %v", err)
	}
	if !strings.Contains(inspectOut.String(), "Records          3") {
		t.Fatalf("inspect output = %q, want three migration records", inspectOut.String())
	}
}

func TestConformanceMultiSourceFixtureCoversMigrationCommands(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "conformance", "multi-source-ledger", ".waystone")
	strategyPath := filepath.Join("..", "..", "testdata", "conformance", "strategy-safe-read-only.json")
	sources := []string{
		"github:example/project",
		"gitlab:example/project",
		"forgejo:example/project",
		"gitea:example/project",
		"waystone:example/project",
	}

	var reportOut bytes.Buffer
	args := []string{"migrate", "report", "--ledger", root, "--to", "waystone:example/project", "--json"}
	for _, source := range sources {
		args = append(args, "--from", source)
	}
	if err := Run(context.Background(), args, &reportOut, io.Discard); err != nil {
		t.Fatalf("migrate report returned error: %v", err)
	}
	var report migrationReport
	if err := json.Unmarshal(reportOut.Bytes(), &report); err != nil {
		t.Fatalf("decode migration report: %v", err)
	}
	if len(report.Sources) != len(sources) {
		t.Fatalf("report sources = %d, want %d", len(report.Sources), len(sources))
	}
	if report.Records.Issues != len(sources) {
		t.Fatalf("report issues = %d, want one issue per source", report.Records.Issues)
	}
	for _, warning := range []string{
		"Number collision: issue #1 appears",
		"Label name overlap",
		"Milestone title overlap",
		"Author identity ambiguity",
	} {
		if !migrationWarningsContain(report.Warnings, warning) {
			t.Fatalf("report warnings = %#v, want warning containing %q", report.Warnings, warning)
		}
	}

	planPath := filepath.Join(t.TempDir(), "migration-plan.json")
	planArgs := []string{
		"migrate", "plan",
		"--ledger", root,
		"--to", "waystone:example/project",
		"--strategy-file", strategyPath,
		"--out", planPath,
	}
	for _, source := range sources {
		planArgs = append(planArgs, "--from", source)
	}
	if err := Run(context.Background(), planArgs, io.Discard, io.Discard); err != nil {
		t.Fatalf("migrate plan returned error: %v", err)
	}
	if err := Run(context.Background(), []string{"migrate", "verify", planPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("migrate verify returned error: %v", err)
	}
	var inspectOut bytes.Buffer
	if err := Run(context.Background(), []string{"migrate", "inspect", planPath}, &inspectOut, io.Discard); err != nil {
		t.Fatalf("migrate inspect returned error: %v", err)
	}
	if !strings.Contains(inspectOut.String(), "Target writes    none") {
		t.Fatalf("inspect output = %q, want disabled target writes", inspectOut.String())
	}
	plan := readMigrationPlanFixture(t, planPath)
	var issueRecords int
	for _, record := range plan.Records {
		if record.Object == "issue" {
			issueRecords++
		}
	}
	if issueRecords != len(sources) {
		t.Fatalf("plan issue records = %d, want one issue record per source", issueRecords)
	}
	for _, record := range plan.Records {
		if record.Object == "issue" && !strings.HasPrefix(record.TargetKey, record.Source+":issue:") {
			t.Fatalf("record target key = %q, want source-scoped issue key for %q", record.TargetKey, record.Source)
		}
	}

	var lossOut bytes.Buffer
	lossArgs := []string{"migrate", "loss-report", "--ledger", root, "--to", "waystone:example/project", "--json"}
	for _, source := range sources {
		lossArgs = append(lossArgs, "--from", source)
	}
	if err := Run(context.Background(), lossArgs, &lossOut, io.Discard); err != nil {
		t.Fatalf("migrate loss-report returned error: %v", err)
	}
	var loss migrationLossReport
	if err := json.Unmarshal(lossOut.Bytes(), &loss); err != nil {
		t.Fatalf("decode loss report: %v", err)
	}
	for _, category := range []string{"attachments", "review_threads", "ci_history", "workflows", "permissions", "branch_protections", "user_mapping", "release_assets", "visibility"} {
		if !lossReportHasCategory(loss, category) {
			t.Fatalf("loss report losses = %#v, want category %q", loss.Losses, category)
		}
	}
}

func TestConformancePlanFixtureVerifies(t *testing.T) {
	planPath := filepath.Join("..", "..", "testdata", "conformance", "migration-plan-v1", "migration-plan.json")
	if err := Run(context.Background(), []string{"migrate", "verify", planPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("migrate verify fixture returned error: %v", err)
	}
	plan := readMigrationPlanFixture(t, planPath)
	if plan.Version != "waystone.migration_plan.v1" {
		t.Fatalf("plan version = %q, want migration plan v1", plan.Version)
	}
	if plan.Strategy.TargetWrite != "none" {
		t.Fatalf("target write strategy = %q, want none", plan.Strategy.TargetWrite)
	}
}

func TestConformanceLocalLabelledIssuesFixtureReadsLocalRecords(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "conformance", "local-labelled-issues-ledger", ".waystone")
	source := "waystone:example/project"

	var issuesOut bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "list", "--ledger", root, "--source", source}, &issuesOut, io.Discard); err != nil {
		t.Fatalf("issue list returned error: %v", err)
	}
	if !strings.Contains(issuesOut.String(), "Local labelled issue") {
		t.Fatalf("issue list output = %q, want local issue", issuesOut.String())
	}

	var labelsOut bytes.Buffer
	if err := Run(context.Background(), []string{"label", "list", "--ledger", root, "--source", source}, &labelsOut, io.Discard); err != nil {
		t.Fatalf("label list returned error: %v", err)
	}
	if !strings.Contains(labelsOut.String(), "Bug") {
		t.Fatalf("label list output = %q, want local label", labelsOut.String())
	}

	var reportOut bytes.Buffer
	if err := Run(context.Background(), []string{"migrate", "report", "--ledger", root, "--from", source, "--to", source, "--json"}, &reportOut, io.Discard); err != nil {
		t.Fatalf("migrate report returned error: %v", err)
	}
	var report migrationReport
	if err := json.Unmarshal(reportOut.Bytes(), &report); err != nil {
		t.Fatalf("decode migration report: %v", err)
	}
	if report.Records.Issues != 1 || report.Records.Labels != 1 {
		t.Fatalf("report records = %#v, want one local issue and one label", report.Records)
	}
}

func TestConformanceLossReportFixtureMatchesGeneratedShape(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "conformance", "multi-source-ledger", ".waystone")
	fixturePath := filepath.Join("..", "..", "testdata", "conformance", "migration-loss-report-v1", "migration-loss-report.json")
	sources := []string{
		"github:example/project",
		"gitlab:example/project",
		"forgejo:example/project",
		"gitea:example/project",
		"waystone:example/project",
	}

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read loss-report fixture: %v", err)
	}
	var fixture migrationLossReport
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode loss-report fixture: %v", err)
	}

	var generatedOut bytes.Buffer
	args := []string{"migrate", "loss-report", "--ledger", root, "--to", "waystone:example/project", "--json"}
	for _, source := range sources {
		args = append(args, "--from", source)
	}
	if err := Run(context.Background(), args, &generatedOut, io.Discard); err != nil {
		t.Fatalf("generate loss report: %v", err)
	}
	var generated migrationLossReport
	if err := json.Unmarshal(generatedOut.Bytes(), &generated); err != nil {
		t.Fatalf("decode generated loss report: %v", err)
	}
	if !lossReportsEqual(fixture, generated) {
		t.Fatalf("loss-report fixture does not match generated shape\nfixture:  %#v\ngenerated:%#v", fixture, generated)
	}
}

func TestCompatibilityDocumentCoversCurrentContracts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "compatibility.md"))
	if err != nil {
		t.Fatalf("read compatibility document: %v", err)
	}
	doc := string(data)
	for _, want := range []string{
		"waystone.ledger.v1",
		"waystone.migration_plan.v1",
		"waystone.migration_strategy.v1",
		"waystone.migration_loss_report.v1",
		"RFC 8259",
		"RFC 3339",
		"Semantic Versioning",
		"Conformance Fixtures",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("compatibility.md missing %q", want)
		}
	}
}

func migrationWarningsContain(warnings []string, fragment string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, fragment) {
			return true
		}
	}
	return false
}

func lossReportsEqual(a, b migrationLossReport) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}
