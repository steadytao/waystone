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

func (w Writer) WriteForgeImport(imported model.GitHubImport) error {
	return w.WriteGitHubImport(imported)
}

func (w Writer) DiffForgeImport(imported model.GitHubImport) (Diff, error) {
	return w.DiffGitHubImport(imported)
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

func (w Writer) WriteLocalIssueComment(issue model.Issue, comment model.Comment) error {
	if err := validateLocalIssueForWrite(w.Root, issue); err != nil {
		return err
	}
	if comment.Source.System != "waystone" {
		return fmt.Errorf("local comments must use waystone sources")
	}
	if comment.ID == "" {
		return fmt.Errorf("comment ID must not be empty")
	}
	if comment.IssueNumber != issue.Number {
		return fmt.Errorf("comment issue number does not match issue")
	}
	if err := w.writeLedgerMetadata(); err != nil {
		return err
	}
	for _, target := range localIssueCommentTargets(issue, comment) {
		if err := w.writeTarget(target); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) WriteLocalIssueEvent(issue model.Issue, event model.IssueEvent) error {
	if err := validateLocalIssueForWrite(w.Root, issue); err != nil {
		return err
	}
	if event.Source.System != "waystone" {
		return fmt.Errorf("local issue events must use waystone sources")
	}
	if event.ID == "" {
		return fmt.Errorf("issue event ID must not be empty")
	}
	if event.IssueNumber != issue.Number {
		return fmt.Errorf("issue event number does not match issue")
	}
	if err := w.writeLedgerMetadata(); err != nil {
		return err
	}
	for _, target := range localIssueEventTargets(issue, event) {
		if err := w.writeTarget(target); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) WriteLocalLabel(label model.Label) error {
	if w.Root == "" {
		return fmt.Errorf("ledger root must not be empty")
	}
	if label.Source.System != "waystone" {
		return fmt.Errorf("local labels must use waystone sources")
	}
	if label.ID == "" {
		return fmt.Errorf("label ID must not be empty")
	}
	if label.Slug == "" {
		return fmt.Errorf("label slug must not be empty")
	}
	if label.Name == "" {
		return fmt.Errorf("label name must not be empty")
	}
	if err := w.writeLedgerMetadata(); err != nil {
		return err
	}
	for _, target := range localLabelTargets(label) {
		if err := w.writeTarget(target); err != nil {
			return err
		}
	}
	return nil
}

func validateLocalIssueForWrite(root string, issue model.Issue) error {
	if root == "" {
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
	return writeJSONUnderRoot(w.Root, "ledger.json", ledger)
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
	return writeJSONUnderRoot(w.Root, "ledger.json", current)
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

func localIssueCommentTargets(issue model.Issue, comment model.Comment) []writeTarget {
	manifestSource := issue.Source
	issueTarget := localIssueTarget(issue)
	embeddedSource := cleanEmbeddedSource(manifestSource)
	comment.Provenance.Source = embeddedSource
	comment.Source = embeddedSource
	commentTarget := writeTarget{
		relative: filepath.Join(sourceScopedPath(comment.Source), "comments", idFile(comment.ID)),
		object:   "comment",
		number:   comment.IssueNumber,
		id:       comment.ID,
		value:    comment,
	}
	return localIssueMutationTargets(manifestSource, issueTarget, commentTarget)
}

func localIssueEventTargets(issue model.Issue, event model.IssueEvent) []writeTarget {
	manifestSource := issue.Source
	issueTarget := localIssueTarget(issue)
	embeddedSource := cleanEmbeddedSource(manifestSource)
	event.Provenance.Source = embeddedSource
	event.Source = embeddedSource
	eventTarget := writeTarget{
		relative: filepath.Join(sourceScopedPath(event.Source), "events", idFile(event.ID)),
		object:   "issue_event",
		number:   event.IssueNumber,
		id:       event.ID,
		value:    event,
	}
	return localIssueMutationTargets(manifestSource, issueTarget, eventTarget)
}

func localLabelTargets(label model.Label) []writeTarget {
	manifestSource := label.Source
	embeddedSource := cleanEmbeddedSource(manifestSource)
	label.Provenance.Source = embeddedSource
	label.Source = embeddedSource
	labelTarget := writeTarget{
		relative: filepath.Join(sourceScopedPath(label.Source), "labels", label.ID+".json"),
		object:   "label",
		id:       label.ID,
		value:    label,
	}
	source := manifestSource
	source.Objects = append([]model.SourceObjectRef(nil), manifestSource.Objects...)
	source.Operations = append([]model.SourceOperationRef(nil), manifestSource.Operations...)
	ref, err := sourceObjectRef(labelTarget)
	if err == nil {
		source.Objects = upsertSourceObjectRef(source.Objects, ref)
	}
	return []writeTarget{
		{relative: sourceManifestPath(source), object: "source", id: source.System + ":" + source.Owner + "/" + source.Repo, value: source, source: true},
		labelTarget,
	}
}

func localIssueTarget(issue model.Issue) writeTarget {
	embeddedSource := cleanEmbeddedSource(issue.Source)
	issue.Provenance.Source = embeddedSource
	issue.Source = embeddedSource
	return writeTarget{
		relative: filepath.Join(sourceScopedPath(issue.Source), "issues", numberedFile(issue.Number)),
		object:   "issue",
		number:   issue.Number,
		id:       issue.ID,
		value:    issue,
	}
}

func localIssueMutationTargets(source model.Source, targets ...writeTarget) []writeTarget {
	source.Objects = append([]model.SourceObjectRef(nil), source.Objects...)
	source.Operations = append([]model.SourceOperationRef(nil), source.Operations...)
	for _, target := range targets {
		ref, err := sourceObjectRef(target)
		if err == nil {
			source.Objects = upsertSourceObjectRef(source.Objects, ref)
		}
	}
	result := []writeTarget{
		{relative: sourceManifestPath(source), object: "source", id: source.System + ":" + source.Owner + "/" + source.Repo, value: source, source: true},
	}
	return append(result, targets...)
}

func cleanEmbeddedSource(source model.Source) model.Source {
	source.Objects = nil
	source.Operations = nil
	source.Signature = nil
	return source
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
	relative := sourceManifestPath(source)
	path, err := safeRootedFilePath(w.Root, relative)
	if err != nil {
		return err
	}
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
			return w.writeSourceManifest(relative, current)
		}
	}
	current.Operations = append(current.Operations, ref)
	return w.writeSourceManifest(relative, current)
}

func (w Writer) writeTarget(target writeTarget) error {
	if target.source {
		source, ok := target.value.(model.Source)
		if !ok {
			return fmt.Errorf("source target has unexpected value type %T", target.value)
		}
		return w.writeSourceManifest(target.relative, source)
	}
	return writeJSONUnderRoot(w.Root, target.relative, target.value)
}

func (w Writer) writeSourceManifest(relative string, source model.Source) error {
	signature, err := w.sourceSignature(source)
	if err != nil {
		return err
	}
	source.Signature = signature
	return writeJSONUnderRoot(w.Root, relative, source)
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

func writeJSONUnderRoot(root, relative string, value any) error {
	path, err := safeRootedWritePath(root, relative)
	if err != nil {
		return err
	}
	return writeJSONFile(path, value)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeJSONFile(path, value)
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := rejectSymlinkReplaceTarget(path); err != nil {
		return err
	}
	file, err := createTempFileForReplace(path)
	if err != nil {
		return err
	}
	tempPath := file.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func rejectSymlinkReplaceTarget(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("ledger path %s is a symlink", path)
	}
	if info.IsDir() {
		return fmt.Errorf("ledger path %s is a directory", path)
	}
	return nil
}

func createTempFileForReplace(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("ledger path %s is a symlink", dir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("ledger path %s is not a directory", dir)
	}
	return os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*") // #nosec G304 -- path is constructed inside a checked ledger-owned directory.
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
