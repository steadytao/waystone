// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

type Writer struct {
	Root string
}

const maxFilenameSlugLength = 80

type writeTarget struct {
	relative string
	object   string
	number   int
	id       string
	value    any
	source   bool
}

func (w Writer) WriteGitHubImport(imported model.GitHubImport) error {
	if w.Root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}

	if err := w.writeLedgerMetadata(); err != nil {
		return err
	}

	for _, target := range gitHubImportTargets(imported) {
		if err := w.writeTarget(target); err != nil {
			return err
		}
	}

	return nil
}

func (w Writer) WriteGitHubAudit(audit model.GitHubAudit) error {
	if w.Root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}
	if audit.ID == "" {
		return fmt.Errorf("audit ID must not be empty")
	}
	if err := w.writeLedgerMetadata(); err != nil {
		return err
	}
	for _, target := range gitHubAuditTargets(audit) {
		if err := w.writeTarget(target); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) WriteLocalIssue(issue model.Issue) error {
	if w.Root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}
	if issue.Source.System != "waystone" {
		return fmt.Errorf("local issues must use waystone sources")
	}
	if issue.Number <= 0 {
		return fmt.Errorf("issue number must be positive")
	}
	if issue.ID == "" {
		return fmt.Errorf("issue ID must not be empty")
	}
	if err := w.writeLedgerMetadata(); err != nil {
		return err
	}
	for _, target := range localIssueTargets(issue) {
		if err := w.writeTarget(target); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) writeLedgerMetadata() error {
	now := time.Now().UTC()
	ledger := model.Ledger{
		Version:   "waystone.ledger.v1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	path := filepath.Join(w.Root, "ledger.json")
	var current model.Ledger
	if err := readJSONFile(path, &current); err == nil {
		ledger.CreatedAt = current.CreatedAt
		ledger.DefaultSource = current.DefaultSource
	}
	return writeJSON(path, ledger)
}

func (w Writer) SetDefaultSource(source model.Source) error {
	reader := Reader(w)
	current, err := reader.Ledger()
	if err != nil {
		return err
	}
	if source.System != "" {
		if _, err := reader.Source(source); err != nil {
			return err
		}
	}
	if source.System == "" {
		current.DefaultSource = nil
	} else {
		current.DefaultSource = &source
	}
	current.UpdatedAt = time.Now().UTC()
	return writeJSON(filepath.Join(w.Root, "ledger.json"), current)
}

func gitHubImportTargets(imported model.GitHubImport) []writeTarget {
	targets := gitHubImportObjectTargets(imported)
	source := imported.Source
	source.Objects = sourceObjectRefs(targets)
	return append([]writeTarget{
		{relative: sourceManifestPath(imported.Source), object: "source", id: imported.Source.System + ":" + imported.Source.Owner + "/" + imported.Source.Repo, value: source, source: true},
	}, targets...)
}

func gitHubImportObjectTargets(imported model.GitHubImport) []writeTarget {
	prefix := sourceScopedPath(imported.Source)
	targets := []writeTarget{
		{relative: filepath.Join("projects", imported.Source.System, imported.Source.Owner, imported.Source.Repo+".json"), object: "project", id: imported.Project.ID, value: imported.Project},
	}
	for _, item := range imported.Issues {
		targets = append(targets, writeTarget{relative: filepath.Join(prefix, "issues", numberedFile(item.Number)), object: "issue", number: item.Number, id: item.ID, value: item})
	}
	for _, item := range imported.Comments {
		targets = append(targets, writeTarget{relative: filepath.Join(prefix, "comments", idFile(item.ID)), object: "comment", id: item.ID, value: item})
	}
	for _, item := range imported.PullRequests {
		targets = append(targets, writeTarget{relative: filepath.Join(prefix, "pull_requests", numberedFile(item.Number)), object: "pull_request", number: item.Number, id: item.ID, value: item})
	}
	for _, item := range imported.ReviewComments {
		targets = append(targets, writeTarget{relative: filepath.Join(prefix, "reviews", idFile(item.ID)), object: "review_comment", number: item.PullRequestNumber, id: item.ID, value: item})
	}
	for _, item := range imported.Releases {
		targets = append(targets, writeTarget{relative: filepath.Join(prefix, "releases", idFile(item.ID)), object: "release", id: item.ID, value: item})
	}
	for _, item := range imported.Labels {
		targets = append(targets, writeTarget{relative: filepath.Join(prefix, "labels", namedFile(item.Name)), object: "label", id: item.ID, value: item})
	}
	for _, item := range imported.Milestones {
		targets = append(targets, writeTarget{relative: filepath.Join(prefix, "milestones", numberedFile(item.Number)), object: "milestone", number: item.Number, id: item.ID, value: item})
	}
	return targets
}

func gitHubAuditTarget(audit model.GitHubAudit) writeTarget {
	audit.Source.Objects = nil
	audit.Source.Operations = nil
	return writeTarget{
		relative: filepath.Join(sourceScopedPath(audit.Source), "audits", idFile(audit.ID)),
		object:   "audit",
		id:       audit.ID,
		value:    audit,
	}
}

func gitHubAuditTargets(audit model.GitHubAudit) []writeTarget {
	auditTarget := gitHubAuditTarget(audit)
	source := audit.Source
	ref, err := sourceObjectRef(auditTarget)
	if err == nil {
		source.Objects = upsertSourceObjectRef(source.Objects, ref)
	}
	return []writeTarget{
		{relative: sourceManifestPath(source), object: "source", id: source.System + ":" + source.Owner + "/" + source.Repo, value: source, source: true},
		auditTarget,
	}
}

func localIssueTargets(issue model.Issue) []writeTarget {
	manifestSource := issue.Source
	embeddedSource := manifestSource
	embeddedSource.Objects = nil
	embeddedSource.Operations = nil
	issue.Provenance.Source = embeddedSource
	issue.Source = embeddedSource
	issueTarget := writeTarget{
		relative: filepath.Join(sourceScopedPath(issue.Source), "issues", numberedFile(issue.Number)),
		object:   "issue",
		number:   issue.Number,
		id:       issue.ID,
		value:    issue,
	}
	source := manifestSource
	source.Objects = append([]model.SourceObjectRef(nil), manifestSource.Objects...)
	source.Operations = append([]model.SourceOperationRef(nil), manifestSource.Operations...)
	ref, err := sourceObjectRef(issueTarget)
	if err == nil {
		source.Objects = upsertSourceObjectRef(source.Objects, ref)
	}
	return []writeTarget{
		{relative: sourceManifestPath(source), object: "source", id: source.System + ":" + source.Owner + "/" + source.Repo, value: source, source: true},
		issueTarget,
	}
}

func sourceObjectRefs(targets []writeTarget) []model.SourceObjectRef {
	objects := make([]model.SourceObjectRef, 0, len(targets))
	for _, target := range targets {
		data, err := canonicalJSON(target.value)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(data)
		objects = append(objects, model.SourceObjectRef{
			Object: target.object,
			Number: target.number,
			ID:     target.id,
			Path:   filepath.ToSlash(target.relative),
			SHA256: fmt.Sprintf("%x", sum[:]),
		})
	}
	return objects
}

func (w Writer) RecordSourceOperation(source model.Source, operation model.Operation) error {
	var current model.Source
	path := filepath.Join(w.Root, sourceManifestPath(source))
	if err := readJSONFile(path, &current); err != nil {
		return err
	}
	ref := model.SourceOperationRef{
		ID:        operation.ID,
		Command:   operation.Command,
		Path:      OperationPath(operation.ID),
		StartedAt: operation.StartedAt,
	}
	for i, existing := range current.Operations {
		if existing.ID == ref.ID {
			current.Operations[i] = ref
			return w.writeSourceManifest(path, current)
		}
	}
	current.Operations = append(current.Operations, ref)
	return w.writeSourceManifest(path, current)
}

func (w Writer) writeTarget(target writeTarget) error {
	path := filepath.Join(w.Root, target.relative)
	if target.source {
		source, ok := target.value.(model.Source)
		if !ok {
			return fmt.Errorf("source target has unexpected value type %T", target.value)
		}
		return w.writeSourceManifest(path, source)
	}
	return writeJSON(path, target.value)
}

func (w Writer) writeSourceManifest(path string, source model.Source) error {
	signature, err := w.sourceSignature(source)
	if err != nil {
		return err
	}
	source.Signature = signature
	return writeJSON(path, source)
}

func (w Writer) sourceSignature(source model.Source) (*model.OperationSignature, error) {
	identity, err := DefaultIdentity(w.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	privateKey, err := defaultPrivateKey(w.Root)
	if err != nil {
		return nil, err
	}
	data, err := sourceSigningBytes(source)
	if err != nil {
		return nil, err
	}
	return w.signPayload(identity, privateKey, data)
}

func sourceSigningBytes(source model.Source) ([]byte, error) {
	value := struct {
		System     string                     `json:"system"`
		Owner      string                     `json:"owner"`
		Repo       string                     `json:"repo"`
		URL        string                     `json:"url"`
		Objects    []model.SourceObjectRef    `json:"objects,omitempty"`
		Operations []model.SourceOperationRef `json:"operations,omitempty"`
	}{
		System:     source.System,
		Owner:      source.Owner,
		Repo:       source.Repo,
		URL:        source.URL,
		Objects:    source.Objects,
		Operations: source.Operations,
	}
	return canonicalJSON(value)
}

func sourceObjectRef(target writeTarget) (model.SourceObjectRef, error) {
	data, err := canonicalJSON(target.value)
	if err != nil {
		return model.SourceObjectRef{}, err
	}
	sum := sha256.Sum256(data)
	return model.SourceObjectRef{
		Object: target.object,
		Number: target.number,
		ID:     target.id,
		Path:   filepath.ToSlash(target.relative),
		SHA256: fmt.Sprintf("%x", sum[:]),
	}, nil
}

func upsertSourceObjectRef(refs []model.SourceObjectRef, ref model.SourceObjectRef) []model.SourceObjectRef {
	for i, existing := range refs {
		if existing.Path == ref.Path {
			refs[i] = ref
			return refs
		}
	}
	return append(refs, ref)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.Create(path) // #nosec G304 -- path is constructed inside the selected ledger root.
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func sourceScopedPath(source model.Source) string {
	return filepath.Join("objects", source.System, source.Owner, source.Repo)
}

func SourceScopedPath(source model.Source) string {
	return filepath.ToSlash(sourceScopedPath(source))
}

func sourceManifestPath(source model.Source) string {
	return filepath.Join("imports", source.System, sourceFile(source))
}

func sourceFile(source model.Source) string {
	return namedFile(source.Owner + "-" + source.Repo)
}

func SourcePath(source model.Source) string {
	return filepath.ToSlash(sourceManifestPath(source))
}

func numberedFile(number int) string {
	return fmt.Sprintf("%06d.json", number)
}

func idFile(id string) string {
	return namedFile(id)
}

func namedFile(value string) string {
	hash := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%s-%x.json", boundedSafeName(value), hash[:6])
}

func boundedSafeName(value string) string {
	name := safeName(value)
	if len(name) <= maxFilenameSlugLength {
		return name
	}
	name = strings.Trim(name[:maxFilenameSlugLength], "-._")
	if name == "" {
		return "unnamed"
	}
	return name
}

func safeName(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		return "unnamed"
	}
	return name
}
