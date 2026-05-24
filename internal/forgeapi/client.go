// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

// ForgeAPI exists for both Gitea and Forgejo

package forgeapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	system      string
	token       string
	client      *http.Client
	concurrency int
}

func NewClient(baseURL, system, token string, timeout time.Duration, concurrency int) *Client {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		system:      system,
		token:       token,
		client:      &http.Client{Timeout: timeout},
		concurrency: concurrency,
	}
}

func (c *Client) ImportRepository(ctx context.Context, owner, repo string) (model.GitHubImport, error) {
	repository, err := get[fjRepository](ctx, c, "/repos/"+owner+"/"+repo, nil)
	if err != nil {
		return model.GitHubImport{}, err
	}
	issues, err := c.issues(ctx, owner, repo, repository.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	issueComments, err := c.issueComments(ctx, owner, repo, repository.ID, issues)
	if err != nil {
		return model.GitHubImport{}, err
	}
	pullRequests, err := c.pullRequests(ctx, owner, repo, repository.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	pullRequestComments, err := c.pullRequestComments(ctx, owner, repo, repository.ID, pullRequests)
	if err != nil {
		return model.GitHubImport{}, err
	}
	labels, err := c.labels(ctx, owner, repo, repository.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	milestones, err := c.milestones(ctx, owner, repo, repository.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	releases, err := c.releases(ctx, owner, repo, repository.ID)
	if err != nil {
		return model.GitHubImport{}, err
	}
	imported := model.GitHubImport{
		Project: model.Project{
			ID:          stableID(c.system, "repo", strconv.FormatInt(repository.ID, 10)),
			Name:        repository.FullName,
			Description: repository.Description,
			URL:         repository.HTMLURL,
			CreatedAt:   repository.CreatedAt,
			UpdatedAt:   repository.UpdatedAt,
		},
		Source: model.Source{
			System: c.system,
			Owner:  owner,
			Repo:   repo,
			URL:    repository.HTMLURL,
		},
		ImportedAt:   time.Now().UTC(),
		Issues:       issues,
		Comments:     append(issueComments, pullRequestComments...),
		PullRequests: pullRequests,
		Labels:       labels,
		Milestones:   milestones,
		Releases:     releases,
	}
	attachProvenance(&imported)
	return imported, nil
}

func (c *Client) issues(ctx context.Context, owner, repo string, repositoryID int64) ([]model.Issue, error) {
	values := url.Values{"state": {"all"}, "type": {"issues"}, "limit": {"50"}}
	items, err := getAll[fjIssue](ctx, c, "/repos/"+owner+"/"+repo+"/issues", values)
	if err != nil {
		return nil, err
	}
	issues := make([]model.Issue, 0, len(items))
	for _, item := range items {
		issue := model.Issue{
			ID:          stableID(c.system, "issue", strconv.FormatInt(repositoryID, 10), strconv.Itoa(item.Number)),
			SourceID:    item.ID,
			Number:      item.Number,
			Title:       item.Title,
			Body:        item.Body,
			State:       item.State,
			Author:      author(item.User),
			Labels:      labelNames(item.Labels),
			Comments:    item.Comments,
			Milestone:   milestoneTitle(item.Milestone),
			OriginalURL: item.HTMLURL,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
		if issue.State == "" {
			issue.State = "open"
		}
		if item.ClosedAt != nil {
			issue.ClosedAt = *item.ClosedAt
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

func (c *Client) issueComments(ctx context.Context, owner, repo string, repositoryID int64, issues []model.Issue) ([]model.Comment, error) {
	results, err := concurrentMap(ctx, c.concurrency, issues, func(ctx context.Context, issue model.Issue) ([]model.Comment, error) {
		return c.comments(ctx, owner, repo, repositoryID, issue.Number, "issue_comment", "issue")
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

func (c *Client) pullRequests(ctx context.Context, owner, repo string, repositoryID int64) ([]model.PullRequest, error) {
	values := url.Values{"state": {"all"}, "limit": {"50"}}
	items, err := getAllOptional[fjPullRequest](ctx, c, "/repos/"+owner+"/"+repo+"/pulls", values)
	if err != nil {
		return nil, err
	}
	pullRequests := make([]model.PullRequest, 0, len(items))
	for _, item := range items {
		pr := model.PullRequest{
			ID:          stableID(c.system, "pull_request", strconv.FormatInt(repositoryID, 10), strconv.Itoa(item.Number)),
			SourceID:    item.ID,
			Number:      item.Number,
			Title:       item.Title,
			Body:        item.Body,
			State:       item.State,
			Author:      author(item.User),
			BaseRef:     item.Base.Ref,
			HeadRef:     item.Head.Ref,
			Merged:      item.Merged,
			OriginalURL: item.HTMLURL,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
		if pr.State == "" {
			pr.State = "open"
		}
		if item.ClosedAt != nil {
			pr.ClosedAt = *item.ClosedAt
		}
		if item.MergedAt != nil {
			pr.MergedAt = *item.MergedAt
		}
		pullRequests = append(pullRequests, pr)
	}
	return pullRequests, nil
}

func (c *Client) pullRequestComments(ctx context.Context, owner, repo string, repositoryID int64, pullRequests []model.PullRequest) ([]model.Comment, error) {
	results, err := concurrentMap(ctx, c.concurrency, pullRequests, func(ctx context.Context, pr model.PullRequest) ([]model.Comment, error) {
		return c.comments(ctx, owner, repo, repositoryID, pr.Number, "pull_request_comment", "pull_request")
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

func (c *Client) comments(ctx context.Context, owner, repo string, repositoryID int64, number int, kind, parentObject string) ([]model.Comment, error) {
	items, err := getAll[fjComment](ctx, c, fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number), url.Values{"limit": {"50"}})
	if err != nil {
		return nil, err
	}
	comments := make([]model.Comment, 0, len(items))
	for _, item := range items {
		comments = append(comments, model.Comment{
			ID:           stableID(c.system, kind, strconv.FormatInt(repositoryID, 10), strconv.Itoa(number), strconv.FormatInt(item.ID, 10)),
			SourceID:     item.ID,
			IssueNumber:  number,
			ParentObject: parentObject,
			Author:       author(item.User),
			Body:         item.Body,
			OriginalURL:  item.HTMLURL,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}
	return comments, nil
}

func (c *Client) labels(ctx context.Context, owner, repo string, repositoryID int64) ([]model.Label, error) {
	items, err := getAll[fjLabel](ctx, c, "/repos/"+owner+"/"+repo+"/labels", url.Values{"limit": {"50"}})
	if err != nil {
		return nil, err
	}
	labels := make([]model.Label, 0, len(items))
	for _, item := range items {
		labels = append(labels, model.Label{
			ID:          stableID(c.system, "label", strconv.FormatInt(repositoryID, 10), strconv.FormatInt(item.ID, 10)),
			SourceID:    item.ID,
			Name:        item.Name,
			Color:       strings.TrimPrefix(item.Color, "#"),
			Description: item.Description,
		})
	}
	return labels, nil
}

func (c *Client) milestones(ctx context.Context, owner, repo string, repositoryID int64) ([]model.Milestone, error) {
	items, err := getAll[fjMilestone](ctx, c, "/repos/"+owner+"/"+repo+"/milestones", url.Values{"state": {"all"}, "limit": {"50"}})
	if err != nil {
		return nil, err
	}
	milestones := make([]model.Milestone, 0, len(items))
	for _, item := range items {
		number := int(item.ID)
		milestones = append(milestones, model.Milestone{
			ID:          stableID(c.system, "milestone", strconv.FormatInt(repositoryID, 10), strconv.FormatInt(item.ID, 10)),
			SourceID:    item.ID,
			Number:      number,
			Title:       item.Title,
			Description: item.Description,
			State:       item.State,
			OriginalURL: item.HTMLURL,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return milestones, nil
}

func (c *Client) releases(ctx context.Context, owner, repo string, repositoryID int64) ([]model.Release, error) {
	items, err := getAll[fjRelease](ctx, c, "/repos/"+owner+"/"+repo+"/releases", url.Values{"limit": {"50"}})
	if err != nil {
		return nil, err
	}
	releases := make([]model.Release, 0, len(items))
	for _, item := range items {
		releases = append(releases, model.Release{
			ID:          stableID(c.system, "release", strconv.FormatInt(repositoryID, 10), strconv.FormatInt(item.ID, 10)),
			SourceID:    item.ID,
			TagName:     item.TagName,
			Name:        item.Name,
			Body:        item.Body,
			Author:      author(item.Author),
			OriginalURL: item.HTMLURL,
			CreatedAt:   item.CreatedAt,
			PublishedAt: item.PublishedAt,
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
	return getAllWithMissingPolicy[T](ctx, client, path, query, false)
}

func getAllOptional[T any](ctx context.Context, client *Client, path string, query url.Values) ([]T, error) {
	return getAllWithMissingPolicy[T](ctx, client, path, query, true)
}

func getAllWithMissingPolicy[T any](ctx context.Context, client *Client, path string, query url.Values, missingIsEmpty bool) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("page", "1")
	query.Set("limit", "50")
	var values []T
	for {
		var page []T
		ok, err := client.getMaybe(ctx, path, query, &page)
		if err != nil {
			return nil, err
		}
		if !ok && missingIsEmpty {
			return nil, nil
		}
		if !ok {
			return nil, fmt.Errorf("%s request failed for %s: not found", client.system, path)
		}
		values = append(values, page...)
		if len(page) < 50 {
			return values, nil
		}
		currentPage, err := strconv.Atoi(query.Get("page"))
		if err != nil {
			return nil, err
		}
		query.Set("page", strconv.Itoa(currentPage+1))
	}
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "waystone")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed for %s: %s", c.system, path, forgeAPITransportErrorMessage(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s request failed for %s: %s", c.system, path, forgeAPIErrorMessage(resp))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) getMaybe(ctx context.Context, path string, query url.Values, out any) (bool, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "waystone")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("%s request failed for %s: %s", c.system, path, forgeAPITransportErrorMessage(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("%s request failed for %s: %s", c.system, path, forgeAPIErrorMessage(resp))
	}
	return true, json.NewDecoder(resp.Body).Decode(out)
}

func forgeAPIErrorMessage(resp *http.Response) string {
	var forgeErr struct {
		Message string `json:"message"`
		URL     string `json:"url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&forgeErr)
	message := forgeErr.Message
	if message == "" {
		message = resp.Status
	}
	if resp.StatusCode == http.StatusTooManyRequests && !strings.Contains(strings.ToLower(message), "rate") {
		message = "rate limited: " + message
	}
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		message += "; retry after: " + retryAfter
	}
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		message += "; rate limit remaining: " + remaining
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		message += "; rate limit reset: " + reset
	}
	if forgeErr.URL != "" {
		message += "; documentation: " + forgeErr.URL
	}
	return message
}

func forgeAPITransportErrorMessage(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "request timed out"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "request timed out"
	}
	return err.Error()
}

func attachProvenance(imported *model.GitHubImport) {
	provenance := model.Provenance{ImportID: imported.Source.System + ":" + imported.Source.Owner + "/" + imported.Source.Repo, Source: imported.Source}
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

func author(value fjUser) model.Author {
	return model.Author{ID: value.ID, Login: value.Login, Name: value.FullName, URL: value.HTMLURL, AvatarURL: value.AvatarURL}
}

func labelNames(labels []fjLabel) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		names = append(names, label.Name)
	}
	return names
}

func milestoneTitle(milestone *fjMilestone) string {
	if milestone == nil {
		return ""
	}
	return milestone.Title
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
