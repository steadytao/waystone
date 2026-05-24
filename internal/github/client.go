// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	progress    func(Progress)
	concurrency int
}

type Progress struct {
	Detail  bool
	Message string
}

func NewClient(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		token:       token,
		client:      &http.Client{Timeout: timeout},
		concurrency: 6,
	}
}

func (c *Client) WithProgress(progress func(Progress)) *Client {
	c.progress = progress
	return c
}

func (c *Client) WithConcurrency(concurrency int) *Client {
	if concurrency < 1 {
		concurrency = 1
	}
	c.concurrency = concurrency
	return c
}

func (c *Client) ImportRepository(ctx context.Context, owner, repo string) (model.GitHubImport, error) {
	c.report("Fetching repository metadata")
	repository, err := c.repository(ctx, owner, repo)
	if err != nil {
		return model.GitHubImport{}, err
	}

	c.report("Fetching issues and pull request references")
	issues, pullNumbers, commentNumbers, err := c.issues(ctx, owner, repo)
	if err != nil {
		return model.GitHubImport{}, err
	}
	pullNumberSet := make(map[int]bool, len(pullNumbers))
	for _, number := range pullNumbers {
		pullNumberSet[number] = true
	}

	c.report("Fetching issue and pull request conversation comments")
	commentResults, err := concurrentMap(ctx, c.concurrency, commentNumbers, func(ctx context.Context, number int) ([]model.Comment, error) {
		c.reportDetail(fmt.Sprintf("Fetching conversation comments for #%d", number))
		parentObject := "issue"
		if pullNumberSet[number] {
			parentObject = "pull_request"
		}
		issueComments, err := c.issueComments(ctx, owner, repo, number, parentObject)
		if err != nil {
			return nil, err
		}
		c.reportDetail(fmt.Sprintf("Fetched conversation comments for #%d (%d)", number, len(issueComments)))
		return issueComments, nil
	})
	if err != nil {
		return model.GitHubImport{}, err
	}
	var comments []model.Comment
	for _, result := range commentResults {
		comments = append(comments, result...)
	}

	c.report("Fetching pull request details and review comments")
	type pullResult struct {
		pullRequest    model.PullRequest
		reviewComments []model.ReviewComment
	}
	pullResults, err := concurrentMap(ctx, c.concurrency, pullNumbers, func(ctx context.Context, number int) (pullResult, error) {
		c.reportDetail(fmt.Sprintf("Fetching pull request #%d", number))
		pr, err := c.pullRequest(ctx, owner, repo, number)
		if err != nil {
			return pullResult{}, err
		}
		c.reportDetail(fmt.Sprintf("Fetched pull request #%d", number))

		c.reportDetail(fmt.Sprintf("Fetching review comments for #%d", number))
		comments, err := c.reviewComments(ctx, owner, repo, number)
		if err != nil {
			return pullResult{}, err
		}
		c.reportDetail(fmt.Sprintf("Fetched review comments for #%d (%d)", number, len(comments)))
		return pullResult{pullRequest: pr, reviewComments: comments}, nil
	})
	if err != nil {
		return model.GitHubImport{}, err
	}
	var pullRequests []model.PullRequest
	var reviewComments []model.ReviewComment
	for _, result := range pullResults {
		pullRequests = append(pullRequests, result.pullRequest)
		reviewComments = append(reviewComments, result.reviewComments...)
	}

	c.report("Fetching labels")
	labels, err := c.labels(ctx, owner, repo)
	if err != nil {
		return model.GitHubImport{}, err
	}

	c.report("Fetching milestones")
	milestones, err := c.milestones(ctx, owner, repo)
	if err != nil {
		return model.GitHubImport{}, err
	}

	c.report("Fetching releases")
	releases, err := c.releases(ctx, owner, repo)
	if err != nil {
		return model.GitHubImport{}, err
	}

	imported := model.GitHubImport{
		Project: model.Project{
			ID:          stableID("github", "repo", strconv.FormatInt(repository.ID, 10)),
			Name:        repository.FullName,
			Description: repository.Description,
			URL:         repository.HTMLURL,
			CreatedAt:   repository.CreatedAt,
			UpdatedAt:   repository.UpdatedAt,
		},
		Source: model.Source{
			System: "github",
			Owner:  owner,
			Repo:   repo,
			URL:    repository.HTMLURL,
		},
		ImportedAt:     time.Now().UTC(),
		Issues:         issues,
		Comments:       comments,
		PullRequests:   pullRequests,
		ReviewComments: reviewComments,
		Labels:         labels,
		Milestones:     milestones,
		Releases:       releases,
	}
	attachProvenance(&imported)
	return imported, nil
}

func (c *Client) AuthenticatedUser(ctx context.Context) (string, error) {
	if c.token == "" {
		return "", nil
	}
	var user ghUser
	if err := c.get(ctx, "/user", nil, &user); err != nil {
		return "", err
	}
	return user.Login, nil
}

func (c *Client) ConfirmRepository(ctx context.Context, owner, repo string) error {
	var repository ghRepository
	return c.get(ctx, "/repos/"+owner+"/"+repo, nil, &repository)
}

func (c *Client) report(message string) {
	if c.progress != nil {
		c.progress(Progress{Message: message})
	}
}

func (c *Client) reportDetail(message string) {
	if c.progress != nil {
		c.progress(Progress{Detail: true, Message: message})
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
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "waystone")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github request failed for %s: %s", path, githubErrorMessage(resp))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) getOptional(ctx context.Context, path string, query url.Values, out any) (bool, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "waystone")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("github request failed for %s: %s", path, githubErrorMessage(resp))
	}

	return true, json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) getMaybe(ctx context.Context, path string, query url.Values, out any) (bool, int, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "waystone")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, resp.StatusCode, json.NewDecoder(resp.Body).Decode(out)
	case http.StatusNotFound, http.StatusForbidden, http.StatusUnauthorized:
		_, _ = io.Copy(io.Discard, resp.Body)
		return false, resp.StatusCode, nil
	default:
		return false, resp.StatusCode, fmt.Errorf("github request failed for %s: %s", path, githubErrorMessage(resp))
	}
}

func githubErrorMessage(resp *http.Response) string {
	var githubErr struct {
		Message          string `json:"message"`
		DocumentationURL string `json:"documentation_url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&githubErr)

	message := githubErr.Message
	if message == "" {
		message = resp.Status
	}
	if acceptedScopes := resp.Header.Get("X-Accepted-OAuth-Scopes"); acceptedScopes != "" {
		message += "; accepted OAuth scopes: " + acceptedScopes
	}
	if tokenScopes := resp.Header.Get("X-OAuth-Scopes"); tokenScopes != "" {
		message += "; token OAuth scopes: " + tokenScopes
	}
	if githubErr.DocumentationURL != "" {
		message += "; documentation: " + githubErr.DocumentationURL
	}
	return message
}

func decodeContent(item ghContent) ([]byte, error) {
	if item.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported GitHub content encoding %q for %s", item.Encoding, item.Path)
	}
	content := strings.ReplaceAll(item.Content, "\n", "")
	return base64.StdEncoding.DecodeString(content)
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
