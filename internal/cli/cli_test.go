// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steadytao/waystone/internal/github"
	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func TestDefaultGitHubClientID(t *testing.T) {
	t.Setenv("OAUTH_CLIENT_ID", "")

	if got := defaultGitHubClientID(); got != defaultGitHubOAuthClientID {
		t.Fatalf("defaultGitHubClientID() = %q, want built-in client ID", got)
	}
}

func TestDefaultGitHubClientIDAllowsOverride(t *testing.T) {
	t.Setenv("OAUTH_CLIENT_ID", "custom-client-id")

	if got := defaultGitHubClientID(); got != "custom-client-id" {
		t.Fatalf("defaultGitHubClientID() = %q, want custom-client-id", got)
	}
}

func TestGitHubTokenAlwaysPrefersGitHubToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "primary-token")
	t.Setenv("OTHER_GITHUB_TOKEN", "secondary-token")

	if got := githubTokenFromEnvironment("OTHER_GITHUB_TOKEN"); got != "primary-token" {
		t.Fatalf("githubTokenFromEnvironment() = %q, want primary-token", got)
	}
}

func TestGitHubTokenFallsBackToCustomEnvironment(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("OTHER_GITHUB_TOKEN", "secondary-token")

	if got := githubTokenFromEnvironment("OTHER_GITHUB_TOKEN"); got != "secondary-token" {
		t.Fatalf("githubTokenFromEnvironment() = %q, want secondary-token", got)
	}
}

func TestGlobalAuthCommandIsNotSupported(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"auth"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), `unknown command "auth"`) {
		t.Fatalf("Run error = %v, want unknown auth command", err)
	}
}

func TestGitHubAuthUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"github", "auth"}, &stdout, &stderr)
	if err == nil || !strings.Contains(stderr.String(), "waystone github auth login") {
		t.Fatalf("stderr = %q err = %v, want github auth usage", stderr.String(), err)
	}
}

func TestIdentityInitShowAndOperationSigning(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"identity", "init", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("identity init returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Identity") {
		t.Fatalf("stdout = %q, want identity output", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"identity", "show", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("identity show returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "ed25519") {
		t.Fatalf("stdout = %q, want identity algorithm", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "verify", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("ledger verify returned error: %v", err)
	}
	operations, err := (ledger.Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	last := operations[len(operations)-1]
	if last.Signature == nil || last.Signature.Value == "" || last.Signature.IdentityID == "" {
		t.Fatalf("operation signature = %#v, want signed operation", last.Signature)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "verify", "--strict", "--signatures", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("strict signature verify returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Signatures") || !strings.Contains(stdout.String(), "Unsigned") {
		t.Fatalf("stdout = %q, want signature verification summary", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Source signatures") {
		t.Fatalf("stdout = %q, want source signature verification summary", stdout.String())
	}
}

func TestIdentityTrustCommands(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"identity", "init", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("identity init returned error: %v", err)
	}
	identity, err := ledger.DefaultIdentity(root)
	if err != nil {
		t.Fatalf("DefaultIdentity returned error: %v", err)
	}

	stdout.Reset()
	if err := Run(context.Background(), []string{"identity", "list", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("identity list returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), identity.ID) || !strings.Contains(stdout.String(), "trusted") {
		t.Fatalf("stdout = %q, want trusted identity", stdout.String())
	}

	stdout.Reset()
	if err := Run(context.Background(), []string{"identity", "untrust", "--ledger", root, identity.ID}, &stdout, &stderr); err != nil {
		t.Fatalf("identity untrust returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"identity", "status", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("identity status returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Untrusted") {
		t.Fatalf("stdout = %q, want untrusted summary", stdout.String())
	}

	stdout.Reset()
	if err := Run(context.Background(), []string{"identity", "trust", "--ledger", root, identity.ID}, &stdout, &stderr); err != nil {
		t.Fatalf("identity trust returned error: %v", err)
	}
	operations, err := (ledger.Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	if operations[len(operations)-1].Command != "identity trust" {
		t.Fatalf("last operation = %q, want identity trust", operations[len(operations)-1].Command)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"identity", "status", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("identity status returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Trusted") {
		t.Fatalf("stdout = %q, want trusted summary", stdout.String())
	}
}

func TestGitHubAuditCommandPrintsExitReadinessReport(t *testing.T) {
	apiBase := githubAuditTestServer(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"github", "audit", "--api-base", apiBase, "--no-write", "example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v stderr=%q", err, stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"Repository",
		"example/project",
		"Portable",
		"- issues",
		"Needs migration plan",
		"- GitHub Actions workflows",
		"Evidence",
		"- workflow .github/workflows/ci.yml",
		"Actions",
		"remote 1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, want %q", output, want)
		}
	}
	if strings.Contains(output, "- action actions/checkout@v4") {
		t.Fatalf("stdout = %q, did not expect verbose action listing", output)
	}
}

func TestGitHubAuditCommandVerbosePrintsActionEvidence(t *testing.T) {
	apiBase := githubAuditTestServer(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"github", "audit", "--api-base", apiBase, "--verbose", "--no-write", "example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v stderr=%q", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "- action actions/checkout@v4") {
		t.Fatalf("stdout = %q, want verbose action evidence", stdout.String())
	}
}

func TestGitHubAuditCommandWritesJSON(t *testing.T) {
	apiBase := githubAuditTestServer(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"github", "audit", "--api-base", apiBase, "--json", "--no-write", "example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v stderr=%q", err, stderr.String())
	}
	var payload struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Workflows []struct {
			Path string `json:"path"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON output: %v\n%s", err, stdout.String())
	}
	if payload.Repository.FullName != "example/project" {
		t.Fatalf("repository = %q, want example/project", payload.Repository.FullName)
	}
	if len(payload.Workflows) != 1 {
		t.Fatalf("workflows = %d, want 1", len(payload.Workflows))
	}
}

func TestGitHubAuditCommandWritesLedgerRecord(t *testing.T) {
	apiBase := githubAuditTestServer(t)
	root := t.TempDir()
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"github", "audit", "--api-base", apiBase, "--ledger", root, "example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v stderr=%q", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Operation") {
		t.Fatalf("stdout = %q, want operation summary", stdout.String())
	}
	audits, err := (ledger.Reader{Root: root}).Audits()
	if err != nil {
		t.Fatalf("Audits returned error: %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("audits = %d, want 1", len(audits))
	}
	operations, err := (ledger.Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	if len(operations) != 1 || operations[0].Command != "github audit" {
		t.Fatalf("operations = %#v, want github audit operation", operations)
	}
	if !operationHasObjectChange(operations[0], "source") {
		t.Fatalf("operation changes = %#v, want source manifest change", operations[0].Changes)
	}
	source, err := (ledger.Reader{Root: root}).Source(model.Source{System: "github", Owner: "example", Repo: "project"})
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(source.Objects) != 1 || source.Objects[0].Object != "audit" {
		t.Fatalf("source objects = %#v, want audit ref", source.Objects)
	}
}

func operationHasObjectChange(operation model.Operation, object string) bool {
	for _, change := range operation.Changes {
		if change.Object == object {
			return true
		}
	}
	return false
}

func TestGitHubAuditCommandNoWriteLeavesLedgerEmpty(t *testing.T) {
	apiBase := githubAuditTestServer(t)
	root := t.TempDir()
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"github", "audit", "--api-base", apiBase, "--ledger", root, "--no-write", "example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v stderr=%q", err, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(root, "ledger.json")); !os.IsNotExist(err) {
		t.Fatalf("ledger.json stat error = %v, want not exist", err)
	}
}

func TestGitHubAuditCommandPrintsInaccessibleEvidence(t *testing.T) {
	apiBase := githubAuditInaccessibleTestServer(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"github", "audit", "--api-base", apiBase, "--no-write", "example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v stderr=%q", err, stderr.String())
	}

	for _, want := range []string{
		"Branch protection inaccessible status=403",
		"Repository secrets inaccessible status=403",
		"Repository variables inaccessible status=403",
		"Environments inaccessible status=403",
		"GitHub Pages inaccessible status=403",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestAuditListAndShowCommands(t *testing.T) {
	root := writeTestLedger(t)
	audit := model.GitHubAudit{
		ID:          "github-audit-20260101t000000.000000000z",
		Source:      model.Source{System: "github", Owner: "example", Repo: "project", URL: "https://github.com/example/project"},
		GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Repository:  model.GitHubAuditRepository{FullName: "example/project", URL: "https://github.com/example/project", DefaultBranch: "main"},
		Portable:    []string{"issues"},
		Workflows:   []model.GitHubWorkflow{{Name: "ci.yml", Path: ".github/workflows/ci.yml", Actions: 1}},
		Actions:     []model.GitHubActionUse{{Workflow: ".github/workflows/ci.yml", Value: "actions/checkout@v4", Kind: "remote"}},
	}
	if err := (ledger.Writer{Root: root}).WriteGitHubAudit(audit); err != nil {
		t.Fatalf("WriteGitHubAudit returned error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"audit", "list", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("audit list returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "example/project") || !strings.Contains(stdout.String(), audit.ID) {
		t.Fatalf("stdout = %q, want audit row", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"audit", "show", "--ledger", root, audit.ID}, &stdout, &stderr); err != nil {
		t.Fatalf("audit show returned error: %v", err)
	}
	for _, want := range []string{"Repository", "example/project", "Actions", "remote 1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != Version {
		t.Fatalf("stdout = %q, want %q", stdout.String(), Version)
	}
}

func TestLedgerSummaryCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "summary", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Projects       1") {
		t.Fatalf("stdout = %q, want project summary", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Issues           1") {
		t.Fatalf("stdout = %q, want issue count", stdout.String())
	}
}

func TestLedgerHistoryCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "history", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "github import") {
		t.Fatalf("stdout = %q, want operation history", stdout.String())
	}
}

func TestLedgerShowOperationCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "show-operation", "--ledger", root, "github-import-20260101t000000.000000000z"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Command        github import") {
		t.Fatalf("stdout = %q, want operation detail", stdout.String())
	}
}

func TestLedgerVerifyCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "verify", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Checksum") {
		t.Fatalf("stdout = %q, want checksum", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Operation") {
		t.Fatalf("stdout = %q, want operation ID", stdout.String())
	}
	operations, err := (ledger.Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	var found bool
	for _, operation := range operations {
		if operation.Command == "ledger verify" {
			found = true
			if len(operation.Changes) == 0 {
				t.Fatal("verify operation did not record verified actions")
			}
		}
	}
	if !found {
		t.Fatal("verify operation was not written")
	}
}

func TestLedgerVerifyOperationsCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "verify", "--operations", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Operations", "Recorded files", "Operation checksum"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestLedgerVerifyStrictCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "verify", "--strict", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Operation checksum") {
		t.Fatalf("stdout = %q, want strict operation checksum", stdout.String())
	}
}

func TestLedgerStatusCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "status", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Projects       1", "Operations", "Checksum"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestLedgerDoctorCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "doctor", "--ledger", root, "--stale-after", "0"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No practical ledger issues found") {
		t.Fatalf("stdout = %q, want clean doctor output", stdout.String())
	}
}

func TestLedgerDoctorStaleSource(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "doctor", "--ledger", root, "--stale-after", "1h"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "last refreshed") {
		t.Fatalf("stdout = %q, want stale-source warning", stdout.String())
	}
}

func TestLedgerDoctorJSONCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"ledger", "doctor", "--ledger", root, "--stale-after", "1h", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	var findings []doctorFinding
	if err := json.Unmarshal(stdout.Bytes(), &findings); err != nil {
		t.Fatalf("doctor JSON did not decode: %v\n%s", err, stdout.String())
	}
	if len(findings) == 0 || findings[0].Severity == "" {
		t.Fatalf("doctor JSON findings = %#v, want populated findings", findings)
	}
}

func TestLedgerDiffCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer
	source := model.Source{System: "github", Owner: "example", Repo: "project"}
	operation := model.Operation{
		ID:         "source-refresh-20260102t000000.000000000z",
		Command:    "source refresh",
		StartedAt:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 2, 0, 1, 0, 0, time.UTC),
		Actor:      model.OperationActor{Source: "local"},
		Output:     model.OperationOutput{Ledger: root, Updated: 1},
		Changes: []model.ObjectChange{
			{Type: "updated", Object: "source", Path: ledger.SourcePath(source)},
		},
	}
	if err := (ledger.Writer{Root: root}).WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}
	err := Run(context.Background(), []string{"ledger", "diff", "--ledger", root, "--source", "github:example/project", "--since", "github-import-20260101t000000.000000000z"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "source-refresh") || !strings.Contains(stdout.String(), "imports/github/example-project") {
		t.Fatalf("stdout = %q, want source-scoped diff", stdout.String())
	}
}

func TestMigrateReportCommandPrintsReadOnlyReport(t *testing.T) {
	root := writeTestLedger(t)
	createLocalIssueAndLabel(t, root)
	if err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"migrate", "report", "--ledger", root, "--from", "github:example/project", "--to", "waystone:steadytao/waystone"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("migrate report returned error: %v", err)
	}
	for _, want := range []string{
		"Migration report",
		"From             github:example/project",
		"To               waystone:steadytao/waystone",
		"Records",
		"Issues           1",
		"Pull requests    1",
		"Comments         2",
		"Labels           1",
		"Milestones       1",
		"Releases         0",
		"Identity",
		"Source IDs       preserved",
		"Target IDs       not assigned",
		"Strategy         preserve-source-numbering",
		"Local continuation",
		"Local issues     1",
		"Local labels     1",
		"Local events     1",
		"Warnings",
		"- Attachments are not yet represented",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	operations, err := (ledger.Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	if operations[len(operations)-1].Command == "migrate report" {
		t.Fatal("migrate report wrote an operation, want read-only command")
	}
}

func TestMigrateReportCommandWritesJSON(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"migrate", "report", "--ledger", root, "--from", "github:example/project", "--to", "waystone:steadytao/waystone", "--numbering-strategy", "preserve-source-numbering", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("migrate report returned error: %v", err)
	}
	var report migrationReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decoding migration report JSON: %v\n%s", err, stdout.String())
	}
	if report.From != "github:example/project" || report.To != "waystone:steadytao/waystone" {
		t.Fatalf("report sources = %q -> %q, want github:example/project -> waystone:steadytao/waystone", report.From, report.To)
	}
	if report.Records.Issues != 1 || report.Identity.Strategy != "preserve-source-numbering" {
		t.Fatalf("report = %#v, want source counts and identity strategy", report)
	}
}

func TestMigrateReportCommandRejectsUnknownStrategy(t *testing.T) {
	root := writeTestLedger(t)
	err := Run(context.Background(), []string{"migrate", "report", "--ledger", root, "--from", "github:example/project", "--to", "waystone:steadytao/waystone", "--numbering-strategy", "squash"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "only preserve-source-numbering is supported") {
		t.Fatalf("migrate report error = %v, want unsupported strategy error", err)
	}
}

func TestMigratePlanCommandWritesDeterministicPlan(t *testing.T) {
	root := writeTestLedger(t)
	out := filepath.Join(t.TempDir(), "waystone-migration-plan.json")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"migrate", "plan", "--ledger", root, "--from", "github:example/project", "--to", "waystone:steadytao/waystone", "--numbering-strategy", "preserve-source-numbering", "--out", out}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("migrate plan returned error: %v", err)
	}
	for _, want := range []string{"Migration plan written", "Records          7", out} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read migration plan: %v", err)
	}
	var plan model.MigrationPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatalf("decoding migration plan: %v\n%s", err, data)
	}
	if plan.Version != "waystone.migration_plan.v1" || plan.ToolVersion != Version {
		t.Fatalf("plan version/tool = %q/%q, want waystone.migration_plan.v1/%s", plan.Version, plan.ToolVersion, Version)
	}
	if plan.From != "github:example/project" || plan.To != "waystone:steadytao/waystone" {
		t.Fatalf("plan sources = %q -> %q, want github:example/project -> waystone:steadytao/waystone", plan.From, plan.To)
	}
	if plan.Strategy.Numbering != "preserve-source-numbering" || plan.Strategy.TargetWrite != "none" {
		t.Fatalf("plan strategy = %#v, want safe read-only defaults", plan.Strategy)
	}
	if got := migrationPlanRecordKeys(plan.Records); strings.Join(got, ",") != "issue:1,comment:20,comment:21,pull_request:2,review_comment:40,label:50,milestone:1" {
		t.Fatalf("record order = %v, want deterministic source record order", got)
	}
	first := plan.Records[0]
	if first.SourceID != "github:issue:10" || first.SourceNumber != 1 || first.SourceURL != "https://github.com/example/project/issues/1" || first.WaystoneID != "github:issue:10" {
		t.Fatalf("first record = %#v, want issue source identity", first)
	}
	if first.TargetSource != "waystone:steadytao/waystone" || first.TargetKey == "" || first.NumberingStrategy != "preserve-source-numbering" {
		t.Fatalf("first target mapping = %#v, want read-only target projection", first)
	}
	operations, err := (ledger.Reader{Root: root}).Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	if operations[len(operations)-1].Command == "migrate plan" {
		t.Fatal("migrate plan wrote an operation, want saved artefact only")
	}
}

func TestMigratePlanCommandRejectsUnknownNumberingStrategy(t *testing.T) {
	root := writeTestLedger(t)
	out := filepath.Join(t.TempDir(), "waystone-migration-plan.json")

	err := Run(context.Background(), []string{"migrate", "plan", "--ledger", root, "--from", "github:example/project", "--to", "waystone:steadytao/waystone", "--numbering-strategy", "chronological-renumber", "--out", out}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "only preserve-source-numbering is supported") {
		t.Fatalf("migrate plan error = %v, want unsupported v0 strategy error", err)
	}
}

func TestLedgerExportImportCommands(t *testing.T) {
	root := writeTestLedger(t)
	archive := filepath.Join(t.TempDir(), "waystone-ledger.tar.zst")
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"ledger", "export", "--ledger", root, "--out", archive}, &stdout, &stderr); err != nil {
		t.Fatalf("export returned error: %v", err)
	}
	imported := filepath.Join(t.TempDir(), ".waystone")
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "import", archive, "--ledger", imported, "--unsafe"}, &stdout, &stderr); err != nil {
		t.Fatalf("import returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Import complete") {
		t.Fatalf("stdout = %q, want import complete", stdout.String())
	}
}

func TestLedgerExportDefaultsToExtensionlessArchive(t *testing.T) {
	root := writeTestLedger(t)
	workdir := t.TempDir()
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"ledger", "export", "--ledger", root, "--out", filepath.Join(workdir, "waystone-ledger")}, &stdout, &stderr); err != nil {
		t.Fatalf("export returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Format         archive") {
		t.Fatalf("stdout = %q, want archive format", stdout.String())
	}
}

func TestLedgerExportJSONCommand(t *testing.T) {
	root := writeTestLedger(t)
	out := filepath.Join(t.TempDir(), "waystone-ledger.json")
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"ledger", "export", "--ledger", root, "--format", "json", "--out", out}, &stdout, &stderr); err != nil {
		t.Fatalf("json export returned error: %v", err)
	}
	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read json export: %v", err)
	}
	if !bytes.Contains(content, []byte(`"version": "waystone.export.v1"`)) {
		t.Fatalf("json export = %s, want version", content)
	}
}

func TestLedgerInspectCommand(t *testing.T) {
	root := writeTestLedger(t)
	archive := filepath.Join(t.TempDir(), "waystone-ledger")
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"ledger", "export", "--ledger", root, "--out", archive}, &stdout, &stderr); err != nil {
		t.Fatalf("export returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "inspect", archive}, &stdout, &stderr); err != nil {
		t.Fatalf("inspect returned error: %v", err)
	}
	for _, want := range []string{"Format", "Sources", "Operations", "Strict"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestLedgerExportSourceCommand(t *testing.T) {
	root := writeTestLedger(t)
	archive := filepath.Join(t.TempDir(), "waystone-source")
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"ledger", "export", "--ledger", root, "--source", "github:example/project", "--out", archive}, &stdout, &stderr); err != nil {
		t.Fatalf("source export returned error: %v", err)
	}
	imported := filepath.Join(t.TempDir(), ".waystone")
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "import", archive, "--ledger", imported, "--unsafe"}, &stdout, &stderr); err != nil {
		t.Fatalf("source import returned error: %v", err)
	}
}

func TestLocalLabelledIssueRoundTripValidation(t *testing.T) {
	root := t.TempDir()
	source := model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}
	var stdout, stderr bytes.Buffer

	createLocalIssueAndLabel(t, root)
	if err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}
	if err := Run(context.Background(), []string{"issue", "show", "--ledger", root, "--source", "waystone:steadytao/waystone", "1"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue show returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Software Issue (bug)") {
		t.Fatalf("issue show stdout = %q, want resolved label", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "search", "--ledger", root, "--source", "waystone:steadytao/waystone", "--field", "label", "bug"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue search returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Issues         1") {
		t.Fatalf("issue search stdout = %q, want label match", stdout.String())
	}
	if err := Run(context.Background(), []string{"issue", "label", "remove", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label remove returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "timeline", "--ledger", root, "--source", "waystone:steadytao/waystone", "1"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue timeline returned error: %v", err)
	}
	for _, want := range []string{"issue.opened", "issue.labeled", "issue.unlabeled", "Software Issue (bug)"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("issue timeline stdout = %q, want %q", stdout.String(), want)
		}
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "verify", "--strict", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("ledger verify returned error: %v", err)
	}
	archive := filepath.Join(t.TempDir(), "waystone-labelled")
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "export", "--ledger", root, "--out", archive}, &stdout, &stderr); err != nil {
		t.Fatalf("ledger export returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "inspect", archive}, &stdout, &stderr); err != nil {
		t.Fatalf("ledger inspect returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Manifest") {
		t.Fatalf("ledger inspect stdout = %q, want manifest evidence", stdout.String())
	}

	importedRoot := filepath.Join(t.TempDir(), ".waystone")
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "import", archive, "--ledger", importedRoot, "--unsafe"}, &stdout, &stderr); err != nil {
		t.Fatalf("ledger import returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"ledger", "verify", "--strict", "--ledger", importedRoot}, &stdout, &stderr); err != nil {
		t.Fatalf("imported ledger verify returned error: %v", err)
	}
	labels, err := (ledger.Reader{Root: importedRoot}).SourceLabels(source)
	if err != nil {
		t.Fatalf("SourceLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].Slug != "bug" || labels[0].Name != "Software Issue" || !strings.HasPrefix(labels[0].ID, "lbl_") {
		t.Fatalf("imported labels = %#v, want stable local label", labels)
	}
	events, err := (ledger.Reader{Root: importedRoot}).SourceIssueEvents(source, 1)
	if err != nil {
		t.Fatalf("SourceIssueEvents returned error: %v", err)
	}
	if !hasIssueEvent(events, "issue.labeled", labels[0].ID) || !hasIssueEvent(events, "issue.unlabeled", labels[0].ID) {
		t.Fatalf("imported events = %#v, want labelled and unlabelled events with label ID", events)
	}
}

func TestSourceListAndShowCommands(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"source", "list", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("source list returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "github:example/project") {
		t.Fatalf("stdout = %q, want source spec", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"source", "show", "--ledger", root, "github:example/project"}, &stdout, &stderr); err != nil {
		t.Fatalf("source show returned error: %v", err)
	}
	for _, want := range []string{"Objects", "Operations"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestSourceInspectCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"source", "inspect", "--ledger", root, "--stale-after", "0", "github:example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Manifest hash", "Object types", "issue"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestSourceDefaultCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"source", "default", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("source default show returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Default source is not set") {
		t.Fatalf("stdout = %q, want unset default", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"source", "default", "--ledger", root, "github:example/project"}, &stdout, &stderr); err != nil {
		t.Fatalf("source default set returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "github:example/project") {
		t.Fatalf("stdout = %q, want default source", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"source", "default", "--ledger", root, "--clear"}, &stdout, &stderr); err != nil {
		t.Fatalf("source default clear returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Default source cleared") {
		t.Fatalf("stdout = %q, want clear confirmation", stdout.String())
	}
}

func TestSourceRefreshRequiresSources(t *testing.T) {
	root := writeEmptyLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"source", "refresh", "--ledger", root}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "ledger has no sources to refresh") {
		t.Fatalf("Run error = %v, want missing sources error", err)
	}
}

func TestSourceStatusCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"source", "status", "--ledger", root, "--stale-after", "0"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"github:example/project", "LAST REFRESH", "no"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestResolveRefreshSources(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	reader := ledger.Reader{Root: root}

	all, err := resolveRefreshSources(reader, nil)
	if err != nil {
		t.Fatalf("resolveRefreshSources returned error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("all sources = %d, want 2", len(all))
	}
	selected, err := resolveRefreshSources(reader, []string{"github:example/project"})
	if err != nil {
		t.Fatalf("resolveRefreshSources selected returned error: %v", err)
	}
	if len(selected) != 1 || ledger.SourceSpec(selected[0]) != "github:example/project" {
		t.Fatalf("selected sources = %#v, want github:example/project", selected)
	}
}

func TestIssueListCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "list", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Issues         1", "NUMBER", "#1       open     issue"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestIssueCreateCommandCreatesLocalIssue(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer

	err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "waystone:steadytao/waystone", "--title", "Example issue", "--body", "Issue body"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, want := range []string{"Issue created", "Source           waystone:steadytao/waystone", "Number           #1", "Title            Example issue"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	}

	reader := ledger.Reader{Root: root}
	source := model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}
	issue, err := reader.SourceIssue(source, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if issue.Title != "Example issue" || issue.Body != "Issue body" || issue.State != "open" {
		t.Fatalf("issue = %#v", issue)
	}
	if issue.Source.System != "waystone" || issue.Source.Owner != "steadytao" || issue.Source.Repo != "waystone" {
		t.Fatalf("issue source = %#v", issue.Source)
	}

	manifest, err := reader.Source(source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(manifest.Objects) != 1 || manifest.Objects[0].Object != "issue" || manifest.Objects[0].Number != 1 {
		t.Fatalf("source objects = %#v", manifest.Objects)
	}
	if len(manifest.Operations) != 1 || manifest.Operations[0].Command != "issue create" {
		t.Fatalf("source operations = %#v", manifest.Operations)
	}

	operations, err := reader.Operations()
	if err != nil {
		t.Fatalf("Operations returned error: %v", err)
	}
	if len(operations) != 1 || operations[0].Command != "issue create" {
		t.Fatalf("operations = %#v", operations)
	}
	if operations[0].Output.Created == 0 || operations[0].Output.Summary.Issues != 1 {
		t.Fatalf("operation output = %#v", operations[0].Output)
	}
	if _, err := reader.VerifyOperations(); err != nil {
		t.Fatalf("VerifyOperations returned error: %v", err)
	}
}

func TestIssueCreateCommandAllocatesNextIssueNumber(t *testing.T) {
	root := t.TempDir()
	for _, title := range []string{"First", "Second"} {
		if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "waystone:steadytao/waystone", "--title", title}, io.Discard, io.Discard); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	}

	issues, err := (ledger.Reader{Root: root}).SourceIssues(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"})
	if err != nil {
		t.Fatalf("SourceIssues returned error: %v", err)
	}
	if len(issues) != 2 || issues[0].Number != 1 || issues[1].Number != 2 {
		t.Fatalf("issues = %#v", issues)
	}
	source, err := (ledger.Reader{Root: root}).Source(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"})
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(source.Objects) != 2 || len(source.Operations) != 2 {
		t.Fatalf("source refs = objects:%#v operations:%#v", source.Objects, source.Operations)
	}
}

func TestIssueCreateCommandDefaultsBareSourceToWaystone(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer

	err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "Bare source"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Source           waystone:steadytao/waystone") {
		t.Fatalf("stdout = %q, want waystone source", out.String())
	}
	if _, err := (ledger.Reader{Root: root}).SourceIssue(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}, 1); err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
}

func TestIssueCreateCommandRefusesImportedSources(t *testing.T) {
	root := t.TempDir()
	err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "github:steadytao/waystone", "--title", "Bad source"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !strings.Contains(err.Error(), "only supports waystone sources") {
		t.Fatalf("error = %q", err)
	}
}

func TestIssueCreateCommandDoesNotTreatSystemPathAsLocalShorthand(t *testing.T) {
	root := t.TempDir()
	err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "github/steadytao/waystone", "--title", "Bad source"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !strings.Contains(err.Error(), "only supports waystone sources") {
		t.Fatalf("error = %q", err)
	}
}

func TestIssueCreateCommandReadsBodyFile(t *testing.T) {
	root := t.TempDir()
	bodyFile := filepath.Join(t.TempDir(), "issue.md")
	if err := os.WriteFile(bodyFile, []byte("File body\n"), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "waystone:steadytao/waystone", "--title", "Body file", "--body-file", bodyFile}, io.Discard, io.Discard); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	issue, err := (ledger.Reader{Root: root}).SourceIssue(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if issue.Body != "File body\n" {
		t.Fatalf("body = %q", issue.Body)
	}
}

func TestIssueCreateCommandWorksWithExistingIssueViews(t *testing.T) {
	root := t.TempDir()
	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "waystone:steadytao/waystone", "--title", "Find me"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, command := range [][]string{
		{"issue", "list", "--ledger", root, "--source", "waystone:steadytao/waystone"},
		{"issue", "show", "--ledger", root, "--source", "waystone:steadytao/waystone", "1"},
		{"issue", "search", "--ledger", root, "--source", "waystone:steadytao/waystone", "Find"},
	} {
		var out bytes.Buffer
		if err := Run(context.Background(), command, &out, io.Discard); err != nil {
			t.Fatalf("Run(%v) returned error: %v", command, err)
		}
		if !strings.Contains(out.String(), "Find me") {
			t.Fatalf("stdout = %q, want local issue title", out.String())
		}
	}
}

func TestIssueCommentCommandCreatesLocalComment(t *testing.T) {
	root := t.TempDir()
	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "Discuss me"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue create returned error: %v", err)
	}

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "comment", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "--body", "Local comment"}, &out, io.Discard); err != nil {
		t.Fatalf("issue comment returned error: %v", err)
	}
	for _, want := range []string{"Comment created", "Source           waystone:steadytao/waystone", "Issue            #1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	}

	source := model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}
	comments, err := (ledger.Reader{Root: root}).SourceComments(source, 1)
	if err != nil {
		t.Fatalf("SourceComments returned error: %v", err)
	}
	if len(comments) != 1 || comments[0].Body != "Local comment" {
		t.Fatalf("comments = %#v", comments)
	}
	issue, err := (ledger.Reader{Root: root}).SourceIssue(source, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if issue.Comments != 1 {
		t.Fatalf("issue comments = %d, want 1", issue.Comments)
	}
	manifest, err := (ledger.Reader{Root: root}).Source(source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(manifest.Operations) != 2 {
		t.Fatalf("source operations = %#v, want 2 operations", manifest.Operations)
	}
	if !sourceManifestHasObject(manifest, "comment") {
		t.Fatalf("source objects = %#v, want comment ref", manifest.Objects)
	}
}

func TestIssueLifecycleCommandsUpdateLocalStateAndTimeline(t *testing.T) {
	root := t.TempDir()
	source := model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}
	commands := [][]string{
		{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "Lifecycle issue"},
		{"issue", "comment", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "--body", "First comment"},
		{"issue", "close", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1"},
		{"issue", "reopen", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1"},
	}
	for _, command := range commands {
		if err := Run(context.Background(), command, io.Discard, io.Discard); err != nil {
			t.Fatalf("Run(%v) returned error: %v", command, err)
		}
	}

	issue, err := (ledger.Reader{Root: root}).SourceIssue(source, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if issue.State != "open" || !issue.ClosedAt.IsZero() {
		t.Fatalf("issue = %#v, want reopened issue", issue)
	}
	manifest, err := (ledger.Reader{Root: root}).Source(source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(manifest.Operations) != 4 {
		t.Fatalf("source operations = %#v, want 4 operations", manifest.Operations)
	}
	if !sourceManifestHasObject(manifest, "issue_event") {
		t.Fatalf("source objects = %#v, want issue event refs", manifest.Objects)
	}

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "timeline", "--ledger", root, "--source", "waystone:steadytao/waystone", "1"}, &out, io.Discard); err != nil {
		t.Fatalf("issue timeline returned error: %v", err)
	}
	for _, want := range []string{"issue.opened", "issue.comment", "issue.closed", "issue.reopened", "First comment"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	}
	if _, err := (ledger.Reader{Root: root}).VerifyOperations(); err != nil {
		t.Fatalf("VerifyOperations returned error: %v", err)
	}
}

func TestIssueEditCommandUpdatesLocalIssue(t *testing.T) {
	root := t.TempDir()
	source := model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}
	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "Old title", "--body", "Old body"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue create returned error: %v", err)
	}

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "edit", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "--title", "New title", "--body", "New body"}, &out, io.Discard); err != nil {
		t.Fatalf("issue edit returned error: %v", err)
	}
	for _, want := range []string{"Issue edited", "Source           waystone:steadytao/waystone", "Issue            #1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	}

	issue, err := (ledger.Reader{Root: root}).SourceIssue(source, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if issue.Title != "New title" || issue.Body != "New body" {
		t.Fatalf("issue = %#v, want edited title and body", issue)
	}
	if !issue.UpdatedAt.After(issue.CreatedAt) && !issue.UpdatedAt.Equal(issue.CreatedAt) {
		t.Fatalf("issue updated_at = %s created_at = %s", issue.UpdatedAt, issue.CreatedAt)
	}
	manifest, err := (ledger.Reader{Root: root}).Source(source)
	if err != nil {
		t.Fatalf("Source returned error: %v", err)
	}
	if len(manifest.Operations) != 2 {
		t.Fatalf("source operations = %#v, want 2 operations", manifest.Operations)
	}
	if !sourceManifestHasObject(manifest, "issue_event") {
		t.Fatalf("source objects = %#v, want issue edit event ref", manifest.Objects)
	}

	var timeline bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "timeline", "--ledger", root, "--source", "waystone:steadytao/waystone", "1"}, &timeline, io.Discard); err != nil {
		t.Fatalf("issue timeline returned error: %v", err)
	}
	for _, want := range []string{"issue.opened", "issue.edited", "New title"} {
		if !strings.Contains(timeline.String(), want) {
			t.Fatalf("timeline = %q, want %q", timeline.String(), want)
		}
	}
	if _, err := (ledger.Reader{Root: root}).VerifyOperations(); err != nil {
		t.Fatalf("VerifyOperations returned error: %v", err)
	}
}

func TestIssueEditCommandReadsBodyFile(t *testing.T) {
	root := t.TempDir()
	bodyFile := filepath.Join(t.TempDir(), "issue.md")
	if err := os.WriteFile(bodyFile, []byte("Edited body\n"), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "Title", "--body", "Old body"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue create returned error: %v", err)
	}
	if err := Run(context.Background(), []string{"issue", "edit", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "--body-file", bodyFile}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue edit returned error: %v", err)
	}
	issue, err := (ledger.Reader{Root: root}).SourceIssue(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if issue.Title != "Title" || issue.Body != "Edited body\n" {
		t.Fatalf("issue = %#v, want body-file update only", issue)
	}
}

func TestIssueEditCommandRequiresChange(t *testing.T) {
	root := t.TempDir()
	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "Title"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue create returned error: %v", err)
	}
	err := Run(context.Background(), []string{"issue", "edit", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !strings.Contains(err.Error(), "requires --title, --body or --body-file") {
		t.Fatalf("error = %q", err)
	}
}

func TestIssueLifecycleCommandsRefuseImportedSources(t *testing.T) {
	root := t.TempDir()
	for _, command := range [][]string{
		{"issue", "edit", "--ledger", root, "--source", "github:steadytao/waystone", "--issue", "1", "--title", "Bad"},
		{"issue", "comment", "--ledger", root, "--source", "github:steadytao/waystone", "--issue", "1", "--body", "Bad"},
		{"issue", "close", "--ledger", root, "--source", "github:steadytao/waystone", "--issue", "1"},
		{"issue", "reopen", "--ledger", root, "--source", "github:steadytao/waystone", "--issue", "1"},
	} {
		err := Run(context.Background(), command, io.Discard, io.Discard)
		if err == nil {
			t.Fatalf("Run(%v) returned nil error", command)
		}
		if !strings.Contains(err.Error(), "only supports waystone sources") {
			t.Fatalf("Run(%v) error = %q", command, err)
		}
	}
}

func sourceManifestHasObject(source model.Source, object string) bool {
	for _, ref := range source.Objects {
		if ref.Object == object {
			return true
		}
	}
	return false
}

func TestDefaultSourceFiltersIssueList(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"source", "default", "--ledger", root, "github:example/project"}, &stdout, &stderr); err != nil {
		t.Fatalf("source default returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "list", "--ledger", root}, &stdout, &stderr); err != nil {
		t.Fatalf("issue list returned error: %v", err)
	}
	if strings.Contains(stdout.String(), "github:example/other") || strings.Contains(stdout.String(), "SOURCE") {
		t.Fatalf("stdout = %q, want default-source scoped output", stdout.String())
	}
}

func TestIssueListSourceFilter(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "list", "--ledger", root, "--source", "github:example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Issues         1", "NUMBER", "#1       open     issue"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	if strings.Contains(stdout.String(), "github:example/other") {
		t.Fatalf("stdout = %q, want only selected source", stdout.String())
	}
}

func TestIssueListStateFilter(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "closed local issue"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue create returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "close", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue close returned error: %v", err)
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "list", "--ledger", root, "--state", "closed"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue list returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Issues         1") || !strings.Contains(stdout.String(), "closed local issue") {
		t.Fatalf("stdout = %q, want closed local issue", stdout.String())
	}
	if strings.Contains(stdout.String(), "#1       open") {
		t.Fatalf("stdout = %q, want open issue excluded", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "list", "--ledger", root, "--source", "github:example/project", "--state", "closed"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue list returned error with source filter: %v", err)
	}
	if !strings.Contains(stdout.String(), "Issues         0") || strings.Contains(stdout.String(), "issue") {
		t.Fatalf("stdout = %q, want no closed issues for selected source", stdout.String())
	}
}

func TestIssueSearchCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "search", "--ledger", root, "issue"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Issues         1", "MATCH", "#1       open     title        issue"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestIssueSearchFields(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"issue", "search", "--ledger", root, "alice"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Issues         0") {
		t.Fatalf("stdout = %q, want default search to skip author", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "search", "--ledger", root, "alice", "--field", "author"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error with author field: %v", err)
	}
	if !strings.Contains(stdout.String(), "author") || !strings.Contains(stdout.String(), "Issues         1") {
		t.Fatalf("stdout = %q, want author match", stdout.String())
	}
}

func TestIssueSearchSourceAndStateFilters(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"issue", "search", "--ledger", root, "--source", "github:example/other", "issue"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue search returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Issues         1") || !strings.Contains(stdout.String(), "#1       open") {
		t.Fatalf("stdout = %q, want selected source issue", stdout.String())
	}
	if strings.Contains(stdout.String(), "github:example/project") {
		t.Fatalf("stdout = %q, want only selected source", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"issue", "search", "--ledger", root, "--state", "closed", "issue"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue search returned error with state filter: %v", err)
	}
	if !strings.Contains(stdout.String(), "Issues         0") {
		t.Fatalf("stdout = %q, want no closed issue matches", stdout.String())
	}
}

func TestIssueShowCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "show", "--ledger", root, "1"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Number         #1") {
		t.Fatalf("stdout = %q, want issue number", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Title          issue") {
		t.Fatalf("stdout = %q, want labelled title", stdout.String())
	}
}

func TestIssueShowWithComments(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "show", "--ledger", root, "--with-comments", "1"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Title          issue", "Comments", "comment"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestIssueShowRequiresSourceWhenAmbiguous(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "show", "--ledger", root, "1"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "exists in multiple sources") {
		t.Fatalf("Run error = %v, want ambiguity error", err)
	}
	stdout.Reset()
	err = Run(context.Background(), []string{"issue", "show", "--ledger", root, "--source", "github:example/project", "1"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error with source: %v", err)
	}
	if !strings.Contains(stdout.String(), "Source         github:example/project") {
		t.Fatalf("stdout = %q, want selected source", stdout.String())
	}
}

func TestIssueCommentsCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "comments", "--ledger", root, "1"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Comments") {
		t.Fatalf("stdout = %q, want comments output", stdout.String())
	}
}

func TestIssueTimelineCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "timeline", "--ledger", root, "1"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Issue", "issue.opened", "issue.comment", "comment"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestIssueCommentsRequiresSourceWhenAmbiguous(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"issue", "comments", "--ledger", root, "1"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "exists in multiple sources") {
		t.Fatalf("Run error = %v, want ambiguity error", err)
	}
	stdout.Reset()
	err = Run(context.Background(), []string{"issue", "comments", "--ledger", root, "--source", "github:example/project", "1"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error with source: %v", err)
	}
	if !strings.Contains(stdout.String(), "Source           github:example/project") {
		t.Fatalf("stdout = %q, want selected source", stdout.String())
	}
}

func TestPullRequestListCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "list", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Pull requests  1", "NUMBER", "#2       closed   pr"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestPullRequestListSourceFilter(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "list", "--ledger", root, "--source", "github:example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Pull requests  1", "NUMBER", "#2       closed   pr"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	if strings.Contains(stdout.String(), "github:example/other") {
		t.Fatalf("stdout = %q, want only selected source", stdout.String())
	}
}

func TestPullRequestSearchCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "search", "--ledger", root, "pr"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Pull requests  1", "MATCH", "#2       closed   title        pr"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestPullRequestSearchFields(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"pr", "search", "--ledger", root, "alice"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Pull requests  0") {
		t.Fatalf("stdout = %q, want default search to skip author", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), []string{"pr", "search", "--ledger", root, "alice", "--field", "author"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error with author field: %v", err)
	}
	if !strings.Contains(stdout.String(), "author") || !strings.Contains(stdout.String(), "Pull requests  1") {
		t.Fatalf("stdout = %q, want author match", stdout.String())
	}
}

func TestPullRequestShowCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "show", "--ledger", root, "2"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Title          pr") {
		t.Fatalf("stdout = %q, want PR title", stdout.String())
	}
}

func TestPullRequestShowWithComments(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "show", "--ledger", root, "--with-comments", "2"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Title          pr", "Review comments", "review"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestPullRequestShowRequiresSourceWhenAmbiguous(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "show", "--ledger", root, "2"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "exists in multiple sources") {
		t.Fatalf("Run error = %v, want ambiguity error", err)
	}
	stdout.Reset()
	err = Run(context.Background(), []string{"pr", "show", "--ledger", root, "--source", "github:example/project", "2"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error with source: %v", err)
	}
	if !strings.Contains(stdout.String(), "Source         github:example/project") {
		t.Fatalf("stdout = %q, want selected source", stdout.String())
	}
}

func TestPullRequestCommentsCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "comments", "--ledger", root, "2"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Review comments") {
		t.Fatalf("stdout = %q, want review comments", stdout.String())
	}
}

func TestPullRequestTimelineCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "timeline", "--ledger", root, "2"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{"Pull request", "pull_request.opened", "pull_request.comment", "pull_request.review_comment", "pr conversation", "review"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestPullRequestCommentsRequiresSourceWhenAmbiguous(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"pr", "comments", "--ledger", root, "2"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "exists in multiple sources") {
		t.Fatalf("Run error = %v, want ambiguity error", err)
	}
	stdout.Reset()
	err = Run(context.Background(), []string{"pr", "comments", "--ledger", root, "--source", "github:example/project", "2"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error with source: %v", err)
	}
	if !strings.Contains(stdout.String(), "Source           github:example/project") {
		t.Fatalf("stdout = %q, want selected source", stdout.String())
	}
}

func TestLabelListCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"label", "list", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "bug") {
		t.Fatalf("stdout = %q, want label", stdout.String())
	}
}

func TestLabelListSourceFilter(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"label", "list", "--ledger", root, "--source", "github:example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "bug") {
		t.Fatalf("stdout = %q, want label", stdout.String())
	}
	if strings.Contains(stdout.String(), "github:example/other") {
		t.Fatalf("stdout = %q, want only selected source", stdout.String())
	}
}

func TestLabelCreateCommandCreatesLocalLabel(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"label", "create", "--ledger", root, "--source", "steadytao/waystone", "--slug", "bug", "--name", "Software Issue", "--color", "d73a4a", "--description", "Something broken"}, &stdout, &stderr); err != nil {
		t.Fatalf("label create returned error: %v", err)
	}
	for _, want := range []string{"Label created", "Source           waystone:steadytao/waystone", "ID               lbl_", "Slug             bug", "Name             Software Issue"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	labels, err := (ledger.Reader{Root: root}).SourceLabels(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"})
	if err != nil {
		t.Fatalf("SourceLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].Slug != "bug" || labels[0].Name != "Software Issue" || labels[0].ID == "bug" {
		t.Fatalf("labels = %#v, want local label with opaque ID", labels)
	}
	if _, err := (ledger.Reader{Root: root}).VerifyOperations(); err != nil {
		t.Fatalf("VerifyOperations returned error: %v", err)
	}
}

func TestLabelCreateCommandDefaultsBareSourceToWaystone(t *testing.T) {
	root := t.TempDir()
	if err := Run(context.Background(), []string{"label", "create", "--ledger", root, "--source", "steadytao/waystone", "--slug", "bug", "--name", "Bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("label create returned error: %v", err)
	}
	if _, err := (ledger.Reader{Root: root}).Source(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}); err != nil {
		t.Fatalf("Source returned error for waystone shorthand: %v", err)
	}
}

func TestLabelCreateCommandRefusesImportedSources(t *testing.T) {
	root := writeTestLedger(t)
	err := Run(context.Background(), []string{"label", "create", "--ledger", root, "--source", "github:example/project", "--slug", "bug", "--name", "Bug"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "only supports waystone sources") {
		t.Fatalf("label create error = %v, want imported source refusal", err)
	}
}

func TestLabelCreateCommandRejectsDuplicateSlug(t *testing.T) {
	root := t.TempDir()
	command := []string{"label", "create", "--ledger", root, "--source", "steadytao/waystone", "--slug", "bug", "--name", "Bug"}
	if err := Run(context.Background(), command, io.Discard, io.Discard); err != nil {
		t.Fatalf("first label create returned error: %v", err)
	}
	err := Run(context.Background(), command, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "label slug already exists") {
		t.Fatalf("second label create error = %v, want duplicate slug error", err)
	}
}

func TestLabelCreateCommandRejectsInvalidColor(t *testing.T) {
	root := t.TempDir()
	err := Run(context.Background(), []string{"label", "create", "--ledger", root, "--source", "steadytao/waystone", "--slug", "bug", "--name", "Bug", "--color", "pink"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "six hex") {
		t.Fatalf("label create error = %v, want invalid colour error", err)
	}
}

func TestIssueLabelAddCommandUpdatesLocalIssue(t *testing.T) {
	root := t.TempDir()
	createLocalIssueAndLabel(t, root)
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}
	for _, want := range []string{"Label added", "Issue            #1", "Label            Software Issue (bug)"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	issue, err := (ledger.Reader{Root: root}).SourceIssue(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if len(issue.Labels) != 1 || !strings.HasPrefix(issue.Labels[0], "lbl_") {
		t.Fatalf("issue labels = %#v, want label ID", issue.Labels)
	}
	if _, err := (ledger.Reader{Root: root}).VerifyOperations(); err != nil {
		t.Fatalf("VerifyOperations returned error: %v", err)
	}
}

func TestIssueLabelAddCommandRefusesImportedSources(t *testing.T) {
	root := writeTestLedger(t)
	err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "github:example/project", "--issue", "1", "bug"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "only supports waystone sources") {
		t.Fatalf("issue label add error = %v, want imported source refusal", err)
	}
}

func TestIssueLabelAddCommandRejectsDuplicateApplication(t *testing.T) {
	root := t.TempDir()
	createLocalIssueAndLabel(t, root)
	command := []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}
	if err := Run(context.Background(), command, io.Discard, io.Discard); err != nil {
		t.Fatalf("first issue label add returned error: %v", err)
	}
	err := Run(context.Background(), command, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "already applied") {
		t.Fatalf("second issue label add error = %v, want duplicate application error", err)
	}
}

func TestIssueLabelRemoveCommandUpdatesLocalIssue(t *testing.T) {
	root := t.TempDir()
	createLocalIssueAndLabel(t, root)
	if err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "label", "remove", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue label remove returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Label removed") || !strings.Contains(stdout.String(), "Software Issue (bug)") {
		t.Fatalf("stdout = %q, want removed label output", stdout.String())
	}
	issue, err := (ledger.Reader{Root: root}).SourceIssue(model.Source{System: "waystone", Owner: "steadytao", Repo: "waystone"}, 1)
	if err != nil {
		t.Fatalf("SourceIssue returned error: %v", err)
	}
	if len(issue.Labels) != 0 {
		t.Fatalf("issue labels = %#v, want label removed", issue.Labels)
	}
}

func TestIssueLabelRemoveCommandRejectsMissingLabel(t *testing.T) {
	root := t.TempDir()
	createLocalIssueAndLabel(t, root)
	err := Run(context.Background(), []string{"issue", "label", "remove", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "not applied") {
		t.Fatalf("issue label remove error = %v, want missing label error", err)
	}
}

func TestIssueTimelineIncludesLabelEvents(t *testing.T) {
	root := t.TempDir()
	createLocalIssueAndLabel(t, root)
	if err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}
	if err := Run(context.Background(), []string{"issue", "label", "remove", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label remove returned error: %v", err)
	}
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "timeline", "--ledger", root, "--source", "waystone:steadytao/waystone", "1"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue timeline returned error: %v", err)
	}
	for _, want := range []string{"issue.labeled", "issue.unlabeled", "Label            Software Issue (bug)"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestIssueShowResolvesLocalLabelNames(t *testing.T) {
	root := t.TempDir()
	createLocalIssueAndLabel(t, root)
	if err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"issue", "show", "--ledger", root, "--source", "waystone:steadytao/waystone", "1"}, &stdout, &stderr); err != nil {
		t.Fatalf("issue show returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Labels         Software Issue (bug)") {
		t.Fatalf("stdout = %q, want resolved label display", stdout.String())
	}
}

func TestIssueSearchMatchesLocalLabelSlugAndName(t *testing.T) {
	root := t.TempDir()
	createLocalIssueAndLabel(t, root)
	if err := Run(context.Background(), []string{"issue", "label", "add", "--ledger", root, "--source", "steadytao/waystone", "--issue", "1", "bug"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue label add returned error: %v", err)
	}
	for _, query := range []string{"bug", "Software"} {
		var stdout, stderr bytes.Buffer
		if err := Run(context.Background(), []string{"issue", "search", "--ledger", root, "--source", "waystone:steadytao/waystone", "--field", "label", query}, &stdout, &stderr); err != nil {
			t.Fatalf("issue search returned error: %v", err)
		}
		if !strings.Contains(stdout.String(), "Issues         1") {
			t.Fatalf("stdout = %q, want label match for %q", stdout.String(), query)
		}
	}
}

func TestMilestoneListCommand(t *testing.T) {
	root := writeTestLedger(t)
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"milestone", "list", "--ledger", root}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "v1") {
		t.Fatalf("stdout = %q, want milestone", stdout.String())
	}
}

func TestMilestoneListSourceFilter(t *testing.T) {
	root := writeTestLedger(t)
	writeTestLedgerSource(t, root, "example", "other")
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"milestone", "list", "--ledger", root, "--source", "github:example/project"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "v1") {
		t.Fatalf("stdout = %q, want milestone", stdout.String())
	}
	if strings.Contains(stdout.String(), "github:example/other") {
		t.Fatalf("stdout = %q, want only selected source", stdout.String())
	}
}

func TestImportProgressOmitsDetailsByDefault(t *testing.T) {
	var stdout bytes.Buffer
	printImportProgress(&stdout, githubProgress(true, "Fetching issue #1"), false)

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no output", stdout.String())
	}
}

func TestImportProgressShowsDetailsWhenVerbose(t *testing.T) {
	var stdout bytes.Buffer
	printImportProgress(&stdout, githubProgress(false, "Fetching issues"), true)
	printImportProgress(&stdout, githubProgress(true, "Fetching issue #1"), true)

	for _, want := range []string{"- Fetching issues...", "  - Fetching issue #1..."} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestNormalizeImportArgsAllowsBooleanFlagsWithoutValue(t *testing.T) {
	got, err := normalizeImportArgs([]string{"steadytao/waymark", "--v", "--verbose", "--plain-file-store", "--local"})
	if err != nil {
		t.Fatalf("normalizeImportArgs returned error: %v", err)
	}
	want := []string{"--v", "--verbose", "--plain-file-store", "--local", "steadytao/waymark"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("normalized args = %v, want %v", got, want)
	}
}

func TestNormalizeImportArgsStillRequiresValuesForValueFlags(t *testing.T) {
	if _, err := normalizeImportArgs([]string{"steadytao/waymark", "--out"}); err == nil {
		t.Fatal("normalizeImportArgs returned nil error for missing --out value")
	}
}

func TestModelOperationUsesRemoteLoginAndOmitsLocalMachineByDefault(t *testing.T) {
	operation := modelOperation(
		"github import",
		[]string{"steadytao/waymark"},
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		"steadytao/waymark",
		".waystone",
		"stored",
		"steadytao",
		false,
		model.GitHubImport{},
		ledger.Diff{},
	)

	if operation.Auth.Login != "steadytao" {
		t.Fatalf("auth login = %q, want steadytao", operation.Auth.Login)
	}
	if operation.Actor.User != "" || operation.Actor.Hostname != "" {
		t.Fatalf("actor leaked local machine identity: %#v", operation.Actor)
	}
}

func githubProgress(detail bool, message string) github.Progress {
	return github.Progress{Detail: detail, Message: message}
}

func createLocalIssueAndLabel(t *testing.T, root string) {
	t.Helper()
	if err := Run(context.Background(), []string{"issue", "create", "--ledger", root, "--source", "steadytao/waystone", "--title", "Labelled issue"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("issue create returned error: %v", err)
	}
	if err := Run(context.Background(), []string{"label", "create", "--ledger", root, "--source", "steadytao/waystone", "--slug", "bug", "--name", "Software Issue", "--color", "d73a4a"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("label create returned error: %v", err)
	}
}

func hasIssueEvent(events []model.IssueEvent, eventType, labelID string) bool {
	for _, event := range events {
		if event.Type == eventType && event.LabelID == labelID {
			return true
		}
	}
	return false
}

func migrationPlanRecordKeys(records []model.MigrationPlanRecord) []string {
	keys := make([]string, 0, len(records))
	for _, record := range records {
		if record.SourceNumber > 0 {
			keys = append(keys, fmt.Sprintf("%s:%d", record.Object, record.SourceNumber))
			continue
		}
		id := record.SourceID
		if n := strings.LastIndex(id, ":"); n >= 0 {
			id = id[n+1:]
		}
		keys = append(keys, record.Object+":"+id)
	}
	return keys
}

func writeTestLedger(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestLedgerSource(t, root, "example", "project")
	return root
}

func writeEmptyLedger(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ledger.json"), []byte(`{"version":"waystone.ledger.v1","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write empty ledger: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "imports", "github"), 0o755); err != nil {
		t.Fatalf("create imports dir: %v", err)
	}
	return root
}

func writeTestLedgerSource(t *testing.T, root, owner, repo string) {
	t.Helper()
	source := model.Source{System: "github", Owner: owner, Repo: repo, URL: "https://github.com/" + owner + "/" + repo}
	source.Operations = []model.SourceOperationRef{
		{
			ID:        "github-import-20260101t000000.000000000z",
			Command:   "github import",
			Path:      "operations/github-import-20260101t000000.000000000z.json",
			StartedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	provenance := model.Provenance{ImportID: "github:example/project", Source: source}
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	imported := model.GitHubImport{
		Project: model.Project{
			ID:        "github:repo:1",
			Name:      owner + "/" + repo,
			URL:       source.URL,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		Source: source,
		Issues: []model.Issue{
			{Provenance: provenance, ID: "github:issue:10", SourceID: 10, Number: 1, Title: "issue", Body: "issue body", State: "open", Author: model.Author{Login: "alice"}, OriginalURL: "https://github.com/example/project/issues/1", CreatedAt: createdAt},
		},
		Comments: []model.Comment{
			{Provenance: provenance, ID: "github:issue_comment:20", SourceID: 20, IssueNumber: 1, Author: model.Author{Login: "bob"}, Body: "comment", OriginalURL: "https://github.com/example/project/issues/1#issuecomment-20", CreatedAt: createdAt.Add(time.Hour)},
			{Provenance: provenance, ID: "github:issue_comment:21", SourceID: 21, IssueNumber: 2, Author: model.Author{Login: "dave"}, Body: "pr conversation", OriginalURL: "https://github.com/example/project/pull/2#issuecomment-21", CreatedAt: createdAt.Add(30 * time.Minute)},
		},
		PullRequests: []model.PullRequest{
			{Provenance: provenance, ID: "github:pull_request:30", SourceID: 30, Number: 2, Title: "pr", Body: "pr body", State: "closed", Author: model.Author{Login: "alice"}, OriginalURL: "https://github.com/example/project/pull/2", CreatedAt: createdAt},
		},
		ReviewComments: []model.ReviewComment{
			{Provenance: provenance, ID: "github:review_comment:40", SourceID: 40, PullRequestNumber: 2, Author: model.Author{Login: "carol"}, Body: "review", OriginalURL: "https://github.com/example/project/pull/2#discussion_r40", CreatedAt: createdAt.Add(time.Hour)},
		},
		Labels: []model.Label{
			{Provenance: provenance, ID: "github:label:50", SourceID: 50, Name: "bug", Description: "Something broken"},
		},
		Milestones: []model.Milestone{
			{Provenance: provenance, ID: "github:milestone:60", SourceID: 60, Number: 1, Title: "v1", State: "open", OriginalURL: "https://github.com/example/project/milestone/1"},
		},
	}
	diff, err := (ledger.Writer{Root: root}).DiffGitHubImport(imported)
	if err != nil {
		t.Fatalf("DiffGitHubImport returned error: %v", err)
	}
	if err := (ledger.Writer{Root: root}).WriteGitHubImport(imported); err != nil {
		t.Fatalf("WriteGitHubImport returned error: %v", err)
	}
	operation := model.Operation{
		ID:         "github-import-20260101t000000.000000000z",
		Command:    "github import",
		StartedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
		Actor:      model.OperationActor{Source: "local", User: "tester"},
		Output:     model.OperationOutput{Ledger: root, Created: diff.Created},
		Changes:    diff.Changes,
	}
	if err := (ledger.Writer{Root: root}).WriteOperation(operation); err != nil {
		t.Fatalf("WriteOperation returned error: %v", err)
	}
}

func githubAuditTestServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			fmt.Fprint(w, `{"login":"tester"}`)
		case "/repos/example/project":
			fmt.Fprint(w, `{"id":1,"full_name":"example/project","description":"test project","html_url":"https://github.com/example/project","default_branch":"main","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}`)
		case "/repos/example/project/contents/.github/workflows":
			fmt.Fprint(w, `[{"name":"ci.yml","path":".github/workflows/ci.yml","type":"file"}]`)
		case "/repos/example/project/contents/.github/workflows/ci.yml":
			writeCLIContent(t, w, ".github/workflows/ci.yml", "name: CI\njobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n")
		case "/repos/example/project/releases":
			fmt.Fprint(w, `[]`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func githubAuditInaccessibleTestServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			fmt.Fprint(w, `{"login":"tester"}`)
		case "/repos/example/project":
			fmt.Fprint(w, `{"id":1,"full_name":"example/project","description":"test project","html_url":"https://github.com/example/project","default_branch":"main","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}`)
		case "/repos/example/project/contents/.github/workflows":
			http.NotFound(w, r)
		case "/repos/example/project/releases":
			fmt.Fprint(w, `[]`)
		case "/repos/example/project/branches/main/protection",
			"/repos/example/project/actions/secrets",
			"/repos/example/project/actions/variables",
			"/repos/example/project/environments",
			"/repos/example/project/pages":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"message":"forbidden"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func writeCLIContent(t *testing.T, w http.ResponseWriter, path, content string) {
	t.Helper()
	payload := map[string]string{
		"name":     path,
		"path":     path,
		"type":     "file",
		"encoding": "base64",
		"content":  base64.StdEncoding.EncodeToString([]byte(content)),
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode content: %v", err)
	}
}
