// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/steadytao/waystone/internal/model"
)

type Reader struct {
	Root string
}

type Summary struct {
	Ledger         model.Ledger
	Projects       []model.Project
	Sources        []model.Source
	Issues         int
	Comments       int
	PullRequests   int
	ReviewComments int
	Labels         int
	Milestones     int
	Releases       int
}

func (r Reader) Project() (model.Project, error) {
	projects, err := r.Projects()
	if err != nil {
		return model.Project{}, err
	}
	if len(projects) != 1 {
		return model.Project{}, fmt.Errorf("ledger has %d projects; use source-scoped command", len(projects))
	}
	return projects[0], nil
}

func (r Reader) Ledger() (model.Ledger, error) {
	var ledger model.Ledger
	if err := r.readJSON("ledger.json", &ledger); err != nil {
		return model.Ledger{}, err
	}
	return ledger, nil
}

func (r Reader) Projects() ([]model.Project, error) {
	projects, err := readTreeJSON[model.Project](filepath.Join(r.Root, "projects"))
	if err != nil {
		return nil, err
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
	return projects, nil
}

func (r Reader) SourceProject(source model.Source) (model.Project, error) {
	var project model.Project
	if err := r.readJSON(filepath.Join("projects", source.System, source.Owner, source.Repo+".json"), &project); err != nil {
		return model.Project{}, err
	}
	return project, nil
}

func (r Reader) Summary() (Summary, error) {
	var summary Summary
	ledger, err := r.Ledger()
	if err != nil {
		return Summary{}, err
	}
	summary.Ledger = ledger
	projects, err := r.Projects()
	if err != nil {
		return Summary{}, err
	}
	summary.Projects = projects

	sources, err := r.Sources()
	if err != nil {
		return Summary{}, err
	}
	summary.Sources = sources

	summary.Issues, err = r.countObjectFiles("issues")
	if err != nil {
		return Summary{}, err
	}
	summary.Comments, err = r.countObjectFiles("comments")
	if err != nil {
		return Summary{}, err
	}
	summary.PullRequests, err = r.countObjectFiles("pull_requests")
	if err != nil {
		return Summary{}, err
	}
	summary.ReviewComments, err = r.countObjectFiles("reviews")
	if err != nil {
		return Summary{}, err
	}
	summary.Labels, err = r.countObjectFiles("labels")
	if err != nil {
		return Summary{}, err
	}
	summary.Milestones, err = r.countObjectFiles("milestones")
	if err != nil {
		return Summary{}, err
	}
	summary.Releases, err = r.countObjectFiles("releases")
	if err != nil {
		return Summary{}, err
	}
	return summary, nil
}

func (r Reader) Sources() ([]model.Source, error) {
	sources, err := readTreeJSON[model.Source](filepath.Join(r.Root, "imports"))
	if err != nil {
		return nil, err
	}
	sort.Slice(sources, func(i, j int) bool {
		return lessSource(sources[i], sources[j])
	})
	return sources, nil
}

func (r Reader) Source(source model.Source) (model.Source, error) {
	var current model.Source
	if err := r.readJSON(sourceManifestPath(source), &current); err != nil {
		return model.Source{}, err
	}
	return current, nil
}

func (r Reader) Issues() ([]model.Issue, error) {
	issues, err := readObjectTreeJSON[model.Issue](r.Root, "issues")
	if err != nil {
		return nil, err
	}
	sort.Slice(issues, func(i, j int) bool {
		return lessIssue(issues[i], issues[j])
	})
	return issues, nil
}

func (r Reader) SourceIssues(source model.Source) ([]model.Issue, error) {
	issues, err := readDirJSON[model.Issue](filepath.Join(r.Root, sourceScopedPath(source), "issues"))
	if err != nil {
		return nil, err
	}
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Number < issues[j].Number
	})
	return issues, nil
}

func (r Reader) Issue(number int) (model.Issue, error) {
	issues, err := r.Issues()
	if err != nil {
		return model.Issue{}, err
	}
	var matches []model.Issue
	for _, issue := range issues {
		if issue.Number == number {
			matches = append(matches, issue)
		}
	}
	if len(matches) == 0 {
		return model.Issue{}, fmt.Errorf("issue %d not found", number)
	}
	if len(matches) > 1 {
		return model.Issue{}, fmt.Errorf("issue %d exists in multiple sources", number)
	}
	return matches[0], nil
}

func (r Reader) SourceIssue(source model.Source, number int) (model.Issue, error) {
	var issue model.Issue
	if err := r.readJSON(filepath.Join(sourceScopedPath(source), "issues", numberedFile(number)), &issue); err != nil {
		return model.Issue{}, err
	}
	return issue, nil
}

func (r Reader) Comments(issueNumber int) ([]model.Comment, error) {
	comments, err := readObjectTreeJSON[model.Comment](r.Root, "comments")
	if err != nil {
		return nil, err
	}
	var filtered []model.Comment
	for _, comment := range comments {
		if comment.IssueNumber == issueNumber {
			filtered = append(filtered, comment)
		}
	}
	return sortComments(filtered), nil
}

func (r Reader) SourceComments(source model.Source, issueNumber int) ([]model.Comment, error) {
	comments, err := readDirJSON[model.Comment](filepath.Join(r.Root, sourceScopedPath(source), "comments"))
	if err != nil {
		return nil, err
	}
	return filterComments(comments, issueNumber), nil
}

func (r Reader) SourceIssueEvents(source model.Source, issueNumber int) ([]model.IssueEvent, error) {
	events, err := readDirJSON[model.IssueEvent](filepath.Join(r.Root, sourceScopedPath(source), "events"))
	if err != nil {
		return nil, err
	}
	return filterIssueEvents(events, issueNumber), nil
}

func (r Reader) PullRequests() ([]model.PullRequest, error) {
	prs, err := readObjectTreeJSON[model.PullRequest](r.Root, "pull_requests")
	if err != nil {
		return nil, err
	}
	sort.Slice(prs, func(i, j int) bool {
		return lessPullRequest(prs[i], prs[j])
	})
	return prs, nil
}

func (r Reader) SourcePullRequests(source model.Source) ([]model.PullRequest, error) {
	prs, err := readDirJSON[model.PullRequest](filepath.Join(r.Root, sourceScopedPath(source), "pull_requests"))
	if err != nil {
		return nil, err
	}
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].Number < prs[j].Number
	})
	return prs, nil
}

func (r Reader) PullRequest(number int) (model.PullRequest, error) {
	prs, err := r.PullRequests()
	if err != nil {
		return model.PullRequest{}, err
	}
	var matches []model.PullRequest
	for _, pr := range prs {
		if pr.Number == number {
			matches = append(matches, pr)
		}
	}
	if len(matches) == 0 {
		return model.PullRequest{}, fmt.Errorf("pull request %d not found", number)
	}
	if len(matches) > 1 {
		return model.PullRequest{}, fmt.Errorf("pull request %d exists in multiple sources", number)
	}
	return matches[0], nil
}

func (r Reader) SourcePullRequest(source model.Source, number int) (model.PullRequest, error) {
	var pr model.PullRequest
	if err := r.readJSON(filepath.Join(sourceScopedPath(source), "pull_requests", numberedFile(number)), &pr); err != nil {
		return model.PullRequest{}, err
	}
	return pr, nil
}

func (r Reader) ReviewComments(pullRequestNumber int) ([]model.ReviewComment, error) {
	comments, err := readObjectTreeJSON[model.ReviewComment](r.Root, "reviews")
	if err != nil {
		return nil, err
	}
	var filtered []model.ReviewComment
	for _, comment := range comments {
		if comment.PullRequestNumber == pullRequestNumber {
			filtered = append(filtered, comment)
		}
	}
	return sortReviewComments(filtered), nil
}

func (r Reader) SourceReviewComments(source model.Source, pullRequestNumber int) ([]model.ReviewComment, error) {
	comments, err := readDirJSON[model.ReviewComment](filepath.Join(r.Root, sourceScopedPath(source), "reviews"))
	if err != nil {
		return nil, err
	}
	return filterReviewComments(comments, pullRequestNumber), nil
}

func (r Reader) Labels() ([]model.Label, error) {
	labels, err := readObjectTreeJSON[model.Label](r.Root, "labels")
	if err != nil {
		return nil, err
	}
	sort.Slice(labels, func(i, j int) bool {
		if cmp := compareSource(labels[i].Source, labels[j].Source); cmp != 0 {
			return cmp < 0
		}
		return labels[i].Name < labels[j].Name
	})
	return labels, nil
}

func (r Reader) SourceLabels(source model.Source) ([]model.Label, error) {
	labels, err := readDirJSON[model.Label](filepath.Join(r.Root, sourceScopedPath(source), "labels"))
	if err != nil {
		return nil, err
	}
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})
	return labels, nil
}

func (r Reader) SourceLabelByID(source model.Source, id string) (model.Label, error) {
	labels, err := r.SourceLabels(source)
	if err != nil {
		return model.Label{}, err
	}
	for _, label := range labels {
		if label.ID == id {
			return label, nil
		}
	}
	return model.Label{}, fmt.Errorf("label %q not found", id)
}

func (r Reader) SourceLabelBySlug(source model.Source, slug string) (model.Label, error) {
	labels, err := r.SourceLabels(source)
	if err != nil {
		return model.Label{}, err
	}
	for _, label := range labels {
		if effectiveLabelSlug(label) == slug {
			return label, nil
		}
	}
	for _, label := range labels {
		if strings.EqualFold(effectiveLabelSlug(label), slug) {
			return label, nil
		}
	}
	return model.Label{}, fmt.Errorf("label slug %q not found", slug)
}

func effectiveLabelSlug(label model.Label) string {
	if label.Slug != "" {
		return label.Slug
	}
	name := strings.ToLower(strings.TrimSpace(label.Name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func (r Reader) Milestones() ([]model.Milestone, error) {
	milestones, err := readObjectTreeJSON[model.Milestone](r.Root, "milestones")
	if err != nil {
		return nil, err
	}
	sort.Slice(milestones, func(i, j int) bool {
		if cmp := compareSource(milestones[i].Source, milestones[j].Source); cmp != 0 {
			return cmp < 0
		}
		if milestones[i].Number == milestones[j].Number {
			return milestones[i].ID < milestones[j].ID
		}
		return milestones[i].Number < milestones[j].Number
	})
	return milestones, nil
}

func (r Reader) SourceMilestones(source model.Source) ([]model.Milestone, error) {
	milestones, err := readDirJSON[model.Milestone](filepath.Join(r.Root, sourceScopedPath(source), "milestones"))
	if err != nil {
		return nil, err
	}
	sort.Slice(milestones, func(i, j int) bool {
		return milestones[i].Number < milestones[j].Number
	})
	return milestones, nil
}

func (r Reader) SourceReleases(source model.Source) ([]model.Release, error) {
	releases, err := readDirJSON[model.Release](filepath.Join(r.Root, sourceScopedPath(source), "releases"))
	if err != nil {
		return nil, err
	}
	sort.Slice(releases, func(i, j int) bool {
		left := releases[i].PublishedAt
		if left.IsZero() {
			left = releases[i].CreatedAt
		}
		right := releases[j].PublishedAt
		if right.IsZero() {
			right = releases[j].CreatedAt
		}
		if left.Equal(right) {
			return releases[i].ID < releases[j].ID
		}
		return left.Before(right)
	})
	return releases, nil
}

func (r Reader) Audits() ([]model.GitHubAudit, error) {
	audits, err := readObjectTreeJSON[model.GitHubAudit](r.Root, "audits")
	if err != nil {
		return nil, err
	}
	sortAudits(audits)
	return audits, nil
}

func (r Reader) SourceAudits(source model.Source) ([]model.GitHubAudit, error) {
	audits, err := readDirJSON[model.GitHubAudit](filepath.Join(r.Root, sourceScopedPath(source), "audits"))
	if err != nil {
		return nil, err
	}
	sortAudits(audits)
	return audits, nil
}

func (r Reader) Audit(id string) (model.GitHubAudit, error) {
	audits, err := r.Audits()
	if err != nil {
		return model.GitHubAudit{}, err
	}
	for _, audit := range audits {
		if audit.ID == id || strings.TrimSuffix(namedFile(audit.ID), ".json") == id {
			return audit, nil
		}
	}
	return model.GitHubAudit{}, fmt.Errorf("audit %q not found", id)
}

func readDirJSON[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var values []T
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var item T
		if err := readJSONFile(filepath.Join(dir, entry.Name()), &item); err != nil {
			return nil, err
		}
		values = append(values, item)
	}
	return values, nil
}

func readTreeJSON[T any](root string) ([]T, error) {
	var values []T
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		var item T
		if err := readJSONFile(path, &item); err != nil {
			return err
		}
		values = append(values, item)
		return nil
	}); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return values, nil
}

func readObjectTreeJSON[T any](root, objectDir string) ([]T, error) {
	var values []T
	base := filepath.Join(root, "objects")
	if _, err := os.Stat(base); err != nil {
		return nil, err
	}
	if err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || filepath.Base(filepath.Dir(path)) != objectDir {
			return nil
		}
		var item T
		if err := readJSONFile(path, &item); err != nil {
			return err
		}
		values = append(values, item)
		return nil
	}); err != nil {
		return nil, err
	}
	return values, nil
}

func filterComments(comments []model.Comment, issueNumber int) []model.Comment {
	var filtered []model.Comment
	for _, comment := range comments {
		if comment.IssueNumber == issueNumber {
			filtered = append(filtered, comment)
		}
	}
	return sortComments(filtered)
}

func sortComments(comments []model.Comment) []model.Comment {
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
			return comments[i].ID < comments[j].ID
		}
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})
	return comments
}

func filterIssueEvents(events []model.IssueEvent, issueNumber int) []model.IssueEvent {
	var filtered []model.IssueEvent
	for _, event := range events {
		if event.IssueNumber == issueNumber {
			filtered = append(filtered, event)
		}
	}
	return sortIssueEvents(filtered)
}

func sortIssueEvents(events []model.IssueEvent) []model.IssueEvent {
	sort.Slice(events, func(i, j int) bool {
		if events[i].CreatedAt.Equal(events[j].CreatedAt) {
			return events[i].ID < events[j].ID
		}
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})
	return events
}

func filterReviewComments(comments []model.ReviewComment, pullRequestNumber int) []model.ReviewComment {
	var filtered []model.ReviewComment
	for _, comment := range comments {
		if comment.PullRequestNumber == pullRequestNumber {
			filtered = append(filtered, comment)
		}
	}
	return sortReviewComments(filtered)
}

func sortReviewComments(comments []model.ReviewComment) []model.ReviewComment {
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
			return comments[i].ID < comments[j].ID
		}
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})
	return comments
}

func sortAudits(audits []model.GitHubAudit) {
	sort.Slice(audits, func(i, j int) bool {
		if audits[i].GeneratedAt.Equal(audits[j].GeneratedAt) {
			return audits[i].ID < audits[j].ID
		}
		return audits[i].GeneratedAt.Before(audits[j].GeneratedAt)
	})
}

func lessSource(a, b model.Source) bool {
	if cmp := compareSource(a, b); cmp != 0 {
		return cmp < 0
	}
	return a.URL < b.URL
}

func lessIssue(a, b model.Issue) bool {
	if cmp := compareSource(a.Source, b.Source); cmp != 0 {
		return cmp < 0
	}
	if a.Number != b.Number {
		return a.Number < b.Number
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	return a.ID < b.ID
}

func lessPullRequest(a, b model.PullRequest) bool {
	if cmp := compareSource(a.Source, b.Source); cmp != 0 {
		return cmp < 0
	}
	if a.Number != b.Number {
		return a.Number < b.Number
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	return a.ID < b.ID
}

func compareSource(a, b model.Source) int {
	return strings.Compare(SourceSpec(a), SourceSpec(b))
}

func (r Reader) readJSON(relative string, out any) error {
	return readJSONFile(filepath.Join(r.Root, relative), out)
}

func (r Reader) countObjectFiles(objectDir string) (int, error) {
	items, err := readObjectTreeJSON[json.RawMessage](r.Root, objectDir)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func readJSONFile(path string, out any) error {
	data, err := os.ReadFile(path) // #nosec G304 -- readers only access files under the configured ledger root.
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
	}
	return nil
}
