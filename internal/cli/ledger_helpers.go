// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

type doctorFinding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type sourceStatus struct {
	Spec            string    `json:"spec"`
	System          string    `json:"system"`
	Owner           string    `json:"owner"`
	Repo            string    `json:"repo"`
	URL             string    `json:"url"`
	Objects         int       `json:"objects"`
	Operations      int       `json:"operations"`
	LastRefresh     time.Time `json:"last_refresh,omitempty"`
	LastRefreshText string    `json:"last_refresh_text"`
	Age             string    `json:"age,omitempty"`
	AgeSeconds      int64     `json:"age_seconds,omitempty"`
	Stale           bool      `json:"stale"`
}

type sourceInspection struct {
	Spec            string         `json:"spec"`
	System          string         `json:"system"`
	Owner           string         `json:"owner"`
	Repo            string         `json:"repo"`
	URL             string         `json:"url"`
	ManifestPath    string         `json:"manifest_path"`
	ManifestSHA256  string         `json:"manifest_sha256"`
	Objects         int            `json:"objects"`
	Operations      int            `json:"operations"`
	LastRefresh     time.Time      `json:"last_refresh,omitempty"`
	LastRefreshText string         `json:"last_refresh_text"`
	Age             string         `json:"age,omitempty"`
	AgeSeconds      int64          `json:"age_seconds,omitempty"`
	Stale           bool           `json:"stale"`
	ObjectTypes     map[string]int `json:"object_types"`
	MissingObjects  int            `json:"missing_objects"`
	ChangedObjects  int            `json:"changed_objects"`
	Hints           []string       `json:"hints,omitempty"`
}

type ledgerDiff struct {
	Source     string             `json:"source"`
	Since      string             `json:"since"`
	Operations []string           `json:"operations"`
	Changes    []ledgerDiffChange `json:"changes"`
}

type ledgerDiffChange struct {
	OperationID string    `json:"operation_id"`
	FinishedAt  time.Time `json:"finished_at"`
	Type        string    `json:"type"`
	Object      string    `json:"object"`
	Number      int       `json:"number,omitempty"`
	ID          string    `json:"id,omitempty"`
	Path        string    `json:"path"`
	SHA256      string    `json:"sha256,omitempty"`
}

func doctorLedger(reader ledger.Reader, staleAfter time.Duration, now time.Time) []doctorFinding {
	var findings []doctorFinding
	current, err := reader.Ledger()
	if err != nil {
		return []doctorFinding{{Severity: "error", Message: "ledger metadata cannot be read: " + err.Error()}}
	}
	sources, err := reader.Sources()
	if err != nil {
		findings = append(findings, doctorFinding{Severity: "error", Message: "source manifests cannot be read: " + err.Error()})
	} else if len(sources) == 0 {
		findings = append(findings, doctorFinding{Severity: "warning", Message: "ledger has no imported sources"})
	}
	if current.DefaultSource == nil && len(sources) > 1 {
		findings = append(findings, doctorFinding{Severity: "info", Message: "multiple sources are imported but no default source is set"})
	}
	if current.DefaultSource != nil {
		if _, err := reader.Source(*current.DefaultSource); err != nil {
			findings = append(findings, doctorFinding{Severity: "warning", Message: "default source is not present in source manifests: " + ledger.SourceSpec(*current.DefaultSource)})
		}
	}
	operations, err := reader.Operations()
	if err != nil {
		findings = append(findings, doctorFinding{Severity: "error", Message: "operation history cannot be read: " + err.Error()})
	} else if len(operations) == 0 {
		findings = append(findings, doctorFinding{Severity: "warning", Message: "ledger has no operation records"})
	}
	if _, err := reader.Verify(); err != nil {
		findings = append(findings, doctorFinding{Severity: "error", Message: "JSON verification failed: " + err.Error()})
	}
	if _, err := reader.VerifyOperations(); err != nil {
		findings = append(findings, doctorFinding{Severity: "warning", Message: "strict operation verification failed: " + err.Error()})
	}
	if staleAfter > 0 {
		for _, source := range sources {
			lastRefresh, ok := lastSourceOperationTime(source)
			if !ok {
				findings = append(findings, doctorFinding{Severity: "warning", Message: fmt.Sprintf("%s has no recorded source refresh operation", ledger.SourceSpec(source))})
				continue
			}
			age := now.Sub(lastRefresh)
			if age > staleAfter {
				findings = append(findings, doctorFinding{Severity: "warning", Message: fmt.Sprintf("%s was last refreshed %s ago", ledger.SourceSpec(source), formatApproxDuration(age))})
			}
		}
	}
	if len(sources) > 1 && current.DefaultSource == nil {
		if issues, err := reader.Issues(); err == nil {
			for _, number := range duplicateIssueNumbers(issues) {
				findings = append(findings, doctorFinding{Severity: "info", Message: fmt.Sprintf("issue #%d exists in multiple sources; use --source or set a default source", number)})
			}
		}
		if prs, err := reader.PullRequests(); err == nil {
			for _, number := range duplicatePullRequestNumbers(prs) {
				findings = append(findings, doctorFinding{Severity: "info", Message: fmt.Sprintf("pull request #%d exists in multiple sources; use --source or set a default source", number)})
			}
		}
	}
	return findings
}

func sourceStatuses(reader ledger.Reader, staleAfter time.Duration, now time.Time) ([]sourceStatus, error) {
	sources, err := reader.Sources()
	if err != nil {
		return nil, err
	}
	statuses := make([]sourceStatus, 0, len(sources))
	for _, source := range sources {
		status := sourceStatus{
			Spec:            ledger.SourceSpec(source),
			System:          source.System,
			Owner:           source.Owner,
			Repo:            source.Repo,
			URL:             source.URL,
			Objects:         len(source.Objects),
			Operations:      len(source.Operations),
			LastRefreshText: "never",
		}
		if lastRefresh, ok := lastSourceOperationTime(source); ok {
			age := now.Sub(lastRefresh)
			status.LastRefresh = lastRefresh
			status.LastRefreshText = lastRefresh.Format(time.RFC3339)
			status.Age = formatApproxDuration(age)
			status.AgeSeconds = int64(age.Seconds())
			status.Stale = staleAfter > 0 && age > staleAfter
		} else {
			status.Stale = staleAfter > 0
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func inspectSource(root string, source model.Source, staleAfter time.Duration, now time.Time) (sourceInspection, error) {
	manifestPath := ledger.SourcePath(source)
	manifestFile, err := ledger.SafeRootedPath(root, manifestPath)
	if err != nil {
		return sourceInspection{}, err
	}
	manifestHash, err := fileSHA256(manifestFile)
	if err != nil {
		return sourceInspection{}, err
	}
	inspection := sourceInspection{
		Spec:            ledger.SourceSpec(source),
		System:          source.System,
		Owner:           source.Owner,
		Repo:            source.Repo,
		URL:             source.URL,
		ManifestPath:    manifestPath,
		ManifestSHA256:  manifestHash,
		Objects:         len(source.Objects),
		Operations:      len(source.Operations),
		LastRefreshText: "never",
		ObjectTypes:     map[string]int{},
	}
	if lastRefresh, ok := lastSourceOperationTime(source); ok {
		age := now.Sub(lastRefresh)
		inspection.LastRefresh = lastRefresh
		inspection.LastRefreshText = lastRefresh.Format(time.RFC3339)
		inspection.Age = formatApproxDuration(age)
		inspection.AgeSeconds = int64(age.Seconds())
		inspection.Stale = staleAfter > 0 && age > staleAfter
	} else {
		inspection.Stale = staleAfter > 0
		inspection.Hints = append(inspection.Hints, "source has no recorded refresh operation")
	}
	for _, ref := range source.Objects {
		inspection.ObjectTypes[ref.Object]++
		objectFile, err := ledger.SafeRootedPath(root, ref.Path)
		if err != nil {
			return sourceInspection{}, err
		}
		sum, err := fileSHA256(objectFile)
		if err != nil {
			if os.IsNotExist(err) {
				inspection.MissingObjects++
				continue
			}
			return sourceInspection{}, err
		}
		if ref.SHA256 != "" && sum != ref.SHA256 {
			inspection.ChangedObjects++
		}
	}
	if inspection.Stale {
		inspection.Hints = append(inspection.Hints, "source is older than the configured stale threshold")
	}
	if inspection.MissingObjects > 0 {
		inspection.Hints = append(inspection.Hints, "source manifest references missing object files")
	}
	if inspection.ChangedObjects > 0 {
		inspection.Hints = append(inspection.Hints, "source manifest object hashes differ from local files")
	}
	return inspection, nil
}

func ledgerDiffSince(reader ledger.Reader, source model.Source, sinceID string, includeVerified bool) (ledgerDiff, error) {
	operations, err := reader.Operations()
	if err != nil {
		return ledgerDiff{}, err
	}
	seenSince := false
	diff := ledgerDiff{Source: ledger.SourceSpec(source), Since: sinceID}
	for _, operation := range operations {
		if !seenSince {
			if operation.ID == sinceID {
				seenSince = true
			}
			continue
		}
		var addedOperation bool
		for _, change := range operation.Changes {
			if change.Type == "verified" && !includeVerified {
				continue
			}
			if !sourceOwnsChange(source, change) {
				continue
			}
			if !addedOperation {
				diff.Operations = append(diff.Operations, operation.ID)
				addedOperation = true
			}
			diff.Changes = append(diff.Changes, ledgerDiffChange{
				OperationID: operation.ID,
				FinishedAt:  operation.FinishedAt,
				Type:        change.Type,
				Object:      change.Object,
				Number:      change.Number,
				ID:          change.ID,
				Path:        change.Path,
				SHA256:      change.SHA256,
			})
		}
	}
	if !seenSince {
		return ledgerDiff{}, fmt.Errorf("operation %q not found in operation sequence", sinceID)
	}
	return diff, nil
}

func sourceOwnsChange(source model.Source, change model.ObjectChange) bool {
	path := filepath.ToSlash(change.Path)
	return path == ledger.SourcePath(source) || strings.HasPrefix(path, ledger.SourceScopedPath(source)+"/")
}

func sortedCountKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func lastSourceOperationTime(source model.Source) (time.Time, bool) {
	var latest time.Time
	for _, operation := range source.Operations {
		if operation.StartedAt.IsZero() {
			continue
		}
		if latest.IsZero() || operation.StartedAt.After(latest) {
			latest = operation.StartedAt
		}
	}
	return latest, !latest.IsZero()
}

func parseStaleDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "0", "off", "none", "disabled":
		return 0, nil
	}
	if strings.HasSuffix(value, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(value, "d"))
		if err != nil || days < 0 {
			return 0, fmt.Errorf("invalid stale duration %q", value)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid stale duration %q: %w", value, err)
	}
	return duration, nil
}

func formatApproxDuration(duration time.Duration) string {
	if duration >= 24*time.Hour {
		days := int(duration / (24 * time.Hour))
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if duration >= time.Hour {
		hours := int(duration / time.Hour)
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	minutes := int(duration / time.Minute)
	if minutes <= 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func writeSourceDefaultOperation(root string, args []string, startedAt time.Time, source model.Source, includeLocal bool) error {
	finishedAt := time.Now().UTC()
	sum, err := fileSHA256(filepath.Join(root, "ledger.json"))
	if err != nil {
		return err
	}
	input := map[string]string{}
	command := "source default"
	if source.System == "" {
		command = "source default --clear"
	} else {
		input["source"] = ledger.SourceSpec(source)
	}
	operation := model.Operation{
		ID:         ledger.NewOperationID(command, startedAt),
		Command:    command,
		Args:       append([]string(nil), args...),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Actor:      ledger.LocalActor(gitConfig("user.name"), gitConfig("user.email"), includeLocal),
		Input:      input,
		Output: model.OperationOutput{
			Ledger:  root,
			Updated: 1,
		},
		Changes: []model.ObjectChange{
			{
				Type:   "updated",
				Object: "ledger",
				Path:   "ledger.json",
				SHA256: sum,
			},
		},
	}
	return (ledger.Writer{Root: root}).WriteOperation(operation)
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- checksum path comes from a source manifest object reference.
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func addLedgerMetadataChange(diff *ledger.Diff, root string, existed bool) error {
	sum, err := fileSHA256(filepath.Join(root, "ledger.json"))
	if err != nil {
		return err
	}
	changeType := "created"
	if existed {
		changeType = "updated"
	}
	if existed {
		diff.Updated++
	} else {
		diff.Created++
	}
	diff.Changes = append(diff.Changes, model.ObjectChange{
		Type:   changeType,
		Object: "ledger",
		Path:   "ledger.json",
		SHA256: sum,
	})
	return nil
}
