// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

type Client struct {
	baseURL     string
	token       string
	client      *http.Client
	concurrency int
}

func NewClient(baseURL, token string, timeout time.Duration, concurrency int) *Client {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		token:       token,
		client:      &http.Client{Timeout: timeout},
		concurrency: concurrency,
	}
}

func (c *Client) ImportProject(ctx context.Context, owner, repo string) (model.GitHubImport, error) {
	projectPath := owner + "/" + repo
	encodedProject := url.PathEscape(projectPath)
	project, err := get[glProject](ctx, c, "/projects/"+encodedProject, nil)
	if err != nil {
		return model.GitHubImport{}, err
	}
	issues, err := c.issues(ctx, encodedProject, project.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	issueNotes, err := c.issueNotes(ctx, encodedProject, project.ID, issues)
	if err != nil {
		return model.GitHubImport{}, err
	}
	mergeRequests, err := c.mergeRequests(ctx, encodedProject, project.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	mergeRequestNotes, err := c.mergeRequestNotes(ctx, encodedProject, project.ID, mergeRequests)
	if err != nil {
		return model.GitHubImport{}, err
	}
	labels, err := c.labels(ctx, encodedProject, project.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	milestones, err := c.milestones(ctx, encodedProject, project.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	releases, err := c.releases(ctx, encodedProject, project.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	imported := model.GitHubImport{
		Project: model.Project{
			ID:          stableID("gitlab", "project", strconv.FormatInt(project.ID, 10)),
			Name:        project.PathWithNamespace,
			Description: project.Description,
			URL:         project.WebURL,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.LastActivityAt,
		},
		Source: model.Source{
			System: "gitlab",
			Owner:  owner,
			Repo:   repo,
			URL:    project.WebURL,
		},
		ImportedAt:   time.Now().UTC(),
		Issues:       issues,
		Comments:     append(issueNotes, mergeRequestNotes...),
		PullRequests: mergeRequests,
		Labels:       labels,
		Milestones:   milestones,
		Releases:     releases,
	}
	attachProvenance(&imported)
	return imported, nil
}

func (c *Client) issues(ctx context.Context, project string, projectID int64) ([]model.Issue, error) {
	values := url.Values{"scope": {"all"}, "per_page": {"100"}}
	items, err := getAll[glIssue](ctx, c, "/projects/"+project+"/issues", values)
	if err != nil {
		return nil, err
	}
	issues := make([]model.Issue, 0, len(items))
	for _, item := range items {
		issue := model.Issue{
			ID:          stableID("gitlab", "issue", strconv.FormatInt(projectID, 10), strconv.Itoa(item.IID)),
			SourceID:    item.ID,
			Number:      item.IID,
			Title:       item.Title,
			Body:        item.Description,
			State:       gitLabIssueState(item.State),
			Author:      author(item.Author),
			Labels:      append([]string(nil), item.Labels...),
			Comments:    item.UserNotesCount,
			OriginalURL: item.WebURL,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
		if item.Milestone != nil {
			issue.Milestone = item.Milestone.Title
		}
		if item.ClosedAt != nil {
			issue.ClosedAt = *item.ClosedAt
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

func (c *Client) issueNotes(ctx context.Context, project string, projectID int64, issues []model.Issue) ([]model.Comment, error) {
	results, err := concurrentMap(ctx, c.concurrency, issues, func(ctx context.Context, issue model.Issue) ([]model.Comment, error) {
		return c.issueNoteComments(ctx, project, projectID, issue)
	})
	if err != nil {
		return nil, err
	}
	var comments []model.Comment
	for _, result := range results {
		comments = append(comments, result...)
	}
	return comments, nil
}

func (c *Client) issueNoteComments(ctx context.Context, project string, projectID int64, issue model.Issue) ([]model.Comment, error) {
	notes, err := getAll[glNote](ctx, c, fmt.Sprintf("/projects/%s/issues/%d/notes", project, issue.Number), url.Values{"per_page": {"100"}})
	if err != nil {
		return nil, err
	}
	comments := make([]model.Comment, 0, len(notes))
	for _, note := range notes {
		comments = append(comments, model.Comment{
			ID:           stableID("gitlab", "issue_note", strconv.FormatInt(projectID, 10), strconv.Itoa(issue.Number), strconv.FormatInt(note.ID, 10)),
			SourceID:     note.ID,
			IssueNumber:  issue.Number,
			ParentObject: "issue",
			Author:       author(note.Author),
			Body:         note.Body,
			CreatedAt:    note.CreatedAt,
			UpdatedAt:    note.UpdatedAt,
		})
	}
	return comments, nil
}

func (c *Client) mergeRequests(ctx context.Context, project string, projectID int64) ([]model.PullRequest, error) {
	values := url.Values{"scope": {"all"}, "per_page": {"100"}}
	items, err := getAll[glMergeRequest](ctx, c, "/projects/"+project+"/merge_requests", values)
	if err != nil {
		return nil, err
	}
	mergeRequests := make([]model.PullRequest, 0, len(items))
	for _, item := range items {
		mr := model.PullRequest{
			ID:          stableID("gitlab", "merge_request", strconv.FormatInt(projectID, 10), strconv.Itoa(item.IID)),
			SourceID:    item.ID,
			Number:      item.IID,
			Title:       item.Title,
			Body:        item.Description,
			State:       gitLabMergeRequestState(item.State),
			Author:      author(item.Author),
			BaseRef:     item.TargetBranch,
			HeadRef:     item.SourceBranch,
			Merged:      item.State == "merged",
			OriginalURL: item.WebURL,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
		if item.ClosedAt != nil {
			mr.ClosedAt = *item.ClosedAt
		}
		if item.MergedAt != nil {
			mr.MergedAt = *item.MergedAt
		}
		mergeRequests = append(mergeRequests, mr)
	}
	return mergeRequests, nil
}

func (c *Client) mergeRequestNotes(ctx context.Context, project string, projectID int64, mergeRequests []model.PullRequest) ([]model.Comment, error) {
	results, err := concurrentMap(ctx, c.concurrency, mergeRequests, func(ctx context.Context, mr model.PullRequest) ([]model.Comment, error) {
		return c.mergeRequestNoteComments(ctx, project, projectID, mr)
	})
	if err != nil {
		return nil, err
	}
	var comments []model.Comment
	for _, result := range results {
		comments = append(comments, result...)
	}
	return comments, nil
}

func (c *Client) mergeRequestNoteComments(ctx context.Context, project string, projectID int64, mr model.PullRequest) ([]model.Comment, error) {
	notes, err := getAll[glNote](ctx, c, fmt.Sprintf("/projects/%s/merge_requests/%d/notes", project, mr.Number), url.Values{"per_page": {"100"}})
	if err != nil {
		return nil, err
	}
	comments := make([]model.Comment, 0, len(notes))
	for _, note := range notes {
		comments = append(comments, model.Comment{
			ID:           stableID("gitlab", "merge_request_note", strconv.FormatInt(projectID, 10), strconv.Itoa(mr.Number), strconv.FormatInt(note.ID, 10)),
			SourceID:     note.ID,
			IssueNumber:  mr.Number,
			ParentObject: "pull_request",
			Author:       author(note.Author),
			Body:         note.Body,
			CreatedAt:    note.CreatedAt,
			UpdatedAt:    note.UpdatedAt,
		})
	}
	return comments, nil
}

func (c *Client) labels(ctx context.Context, project string, projectID int64) ([]model.Label, error) {
	items, err := getAll[glLabel](ctx, c, "/projects/"+project+"/labels", url.Values{"per_page": {"100"}})
	if err != nil {
		return nil, err
	}
	labels := make([]model.Label, 0, len(items))
	for _, item := range items {
		labels = append(labels, model.Label{
			ID:          stableID("gitlab", "label", strconv.FormatInt(projectID, 10), strconv.FormatInt(item.ID, 10)),
			SourceID:    item.ID,
			Name:        item.Name,
			Color:       strings.TrimPrefix(item.Color, "#"),
			Description: item.Description,
		})
	}
	return labels, nil
}

func (c *Client) milestones(ctx context.Context, project string, projectID int64) ([]model.Milestone, error) {
	items, err := getAll[glMilestone](ctx, c, "/projects/"+project+"/milestones", url.Values{"per_page": {"100"}})
	if err != nil {
		return nil, err
	}
	milestones := make([]model.Milestone, 0, len(items))
	for _, item := range items {
		milestones = append(milestones, model.Milestone{
			ID:          stableID("gitlab", "milestone", strconv.FormatInt(projectID, 10), strconv.Itoa(item.IID)),
			SourceID:    item.ID,
			Number:      item.IID,
			Title:       item.Title,
			Description: item.Description,
			State:       gitLabMilestoneState(item.State),
			OriginalURL: item.WebURL,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return milestones, nil
}

func (c *Client) releases(ctx context.Context, project string, projectID int64) ([]model.Release, error) {
	items, err := getAll[glRelease](ctx, c, "/projects/"+project+"/releases", url.Values{"per_page": {"100"}})
	if err != nil {
		return nil, err
	}
	releases := make([]model.Release, 0, len(items))
	for _, item := range items {
		releases = append(releases, model.Release{
			ID:          stableID("gitlab", "release", strconv.FormatInt(projectID, 10), item.TagName),
			TagName:     item.TagName,
			Name:        item.Name,
			Body:        item.Description,
			Author:      author(item.Author),
			OriginalURL: item.Links.Self,
			CreatedAt:   item.CreatedAt,
			PublishedAt: item.ReleasedAt,
		})
	}
	return releases, nil
}

func get[T any](ctx context.Context, client *Client, path string, query url.Values) (T, error) {
	var value T
	if err := client.get(ctx, path, query, &value); err != nil {
		return value, err
	}
	return value, nil
}

func getAll[T any](ctx context.Context, client *Client, path string, query url.Values) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("page", "1")
	var values []T
	for {
		var page []T
		next, err := client.getPage(ctx, path, query, &page)
		if err != nil {
			return nil, err
		}
		values = append(values, page...)
		if next == "" {
			return values, nil
		}
		query.Set("page", next)
	}
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	_, err := c.getPage(ctx, path, query, out)
	return err
}

func (c *Client) getPage(ctx context.Context, path string, query url.Values, out any) (string, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "waystone")
	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var gitLabErr struct {
			Message          any    `json:"message"`
			Error            any    `json:"error"`
			ErrorDescription string `json:"error_description"`
			Scope            string `json:"scope"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&gitLabErr)
		message := fmt.Sprint(gitLabErr.Message)
		if message == "" || message == "<nil>" {
			message = fmt.Sprint(gitLabErr.Error)
		}
		if message == "" || message == "<nil>" {
			message = resp.Status
		}
		if gitLabErr.ErrorDescription != "" {
			message += ": " + gitLabErr.ErrorDescription
		}
		if gitLabErr.Scope != "" {
			message += " (required scope: " + gitLabErr.Scope + ")"
		}
		return "", fmt.Errorf("gitlab request failed for %s: %s", path, message)
	}
	return resp.Header.Get("X-Next-Page"), json.NewDecoder(resp.Body).Decode(out)
}

func attachProvenance(imported *model.GitHubImport) {
	provenance := model.Provenance{ImportID: "gitlab:" + imported.Source.Owner + "/" + imported.Source.Repo, Source: imported.Source}
	for i := range imported.Issues {
		imported.Issues[i].Provenance = provenance
	}
	for i := range imported.Comments {
		imported.Comments[i].Provenance = provenance
	}
	for i := range imported.PullRequests {
		imported.PullRequests[i].Provenance = provenance
	}
	for i := range imported.Labels {
		imported.Labels[i].Provenance = provenance
	}
	for i := range imported.Milestones {
		imported.Milestones[i].Provenance = provenance
	}
	for i := range imported.Releases {
		imported.Releases[i].Provenance = provenance
	}
}

func author(value glAuthor) model.Author {
	return model.Author{ID: value.ID, Login: value.Username, Name: value.Name, URL: value.WebURL, AvatarURL: value.AvatarURL}
}

func gitLabIssueState(state string) string {
	if state == "closed" {
		return "closed"
	}
	return "open"
}

func gitLabMergeRequestState(state string) string {
	switch state {
	case "closed", "merged":
		return "closed"
	default:
		return "open"
	}
}

func gitLabMilestoneState(state string) string {
	if state == "closed" {
		return "closed"
	}
	return "open"
}

func stableID(parts ...string) string {
	return strings.Join(parts, ":")
}

func concurrentMap[T any, R any](ctx context.Context, concurrency int, values []T, fn func(context.Context, T) (R, error)) ([]R, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(values) && len(values) > 0 {
		concurrency = len(values)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]R, len(values))
	jobs := make(chan int)
	errs := make(chan error, 1)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				result, err := fn(ctx, values[index])
				if err != nil {
					select {
					case errs <- err:
						cancel()
					default:
					}
					continue
				}
				results[index] = result
			}
		}()
	}

	for i := range values {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			select {
			case err := <-errs:
				return nil, err
			default:
				return nil, ctx.Err()
			}
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()

	select {
	case err := <-errs:
		return nil, err
	default:
		return results, nil
	}
}
