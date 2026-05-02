// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

func (c *Client) repository(ctx context.Context, owner, repo string) (ghRepository, error) {
	var repository ghRepository
	err := c.get(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo), nil, &repository)
	return repository, err
}

func attachProvenance(imported *model.GitHubImport) {
	if imported == nil {
		return
	}
	provenance := model.Provenance{
		ImportID: sourceID(imported.Source),
		Source:   imported.Source,
	}
	for i := range imported.Issues {
		imported.Issues[i].Provenance = provenance
	}
	for i := range imported.Comments {
		imported.Comments[i].Provenance = provenance
	}
	for i := range imported.PullRequests {
		imported.PullRequests[i].Provenance = provenance
	}
	for i := range imported.ReviewComments {
		imported.ReviewComments[i].Provenance = provenance
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

func sourceID(source model.Source) string {
	system := strings.ToLower(source.System)
	return fmt.Sprintf("%s:%s/%s", system, source.Owner, source.Repo)
}

func (c *Client) issues(ctx context.Context, owner, repo string) ([]model.Issue, []int, []int, error) {
	var out []model.Issue
	var pullNumbers []int
	var commentNumbers []int
	err := paginate(ctx, c, fmt.Sprintf("/repos/%s/%s/issues", owner, repo), url.Values{
		"state":     {"all"},
		"sort":      {"created"},
		"direction": {"asc"},
	}, func(items []ghIssue) error {
		for _, item := range items {
			commentNumbers = append(commentNumbers, item.Number)
			if item.PullRequest != nil {
				pullNumbers = append(pullNumbers, item.Number)
				continue
			}
			out = append(out, convertIssue(item))
		}
		return nil
	})
	return out, pullNumbers, commentNumbers, err
}

func (c *Client) issueComments(ctx context.Context, owner, repo string, number int) ([]model.Comment, error) {
	var out []model.Comment
	err := paginate(ctx, c, fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number), nil, func(items []ghComment) error {
		for _, item := range items {
			out = append(out, convertComment(number, item))
		}
		return nil
	})
	return out, err
}

func (c *Client) pullRequest(ctx context.Context, owner, repo string, number int) (model.PullRequest, error) {
	var pr ghPullRequest
	err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number), nil, &pr)
	return convertPullRequest(pr), err
}

func (c *Client) reviewComments(ctx context.Context, owner, repo string, number int) ([]model.ReviewComment, error) {
	var out []model.ReviewComment
	err := paginate(ctx, c, fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number), nil, func(items []ghReviewComment) error {
		for _, item := range items {
			out = append(out, convertReviewComment(number, item))
		}
		return nil
	})
	return out, err
}

func (c *Client) labels(ctx context.Context, owner, repo string) ([]model.Label, error) {
	var out []model.Label
	err := paginate(ctx, c, fmt.Sprintf("/repos/%s/%s/labels", owner, repo), nil, func(items []ghLabel) error {
		for _, item := range items {
			out = append(out, convertLabel(item))
		}
		return nil
	})
	return out, err
}

func (c *Client) milestones(ctx context.Context, owner, repo string) ([]model.Milestone, error) {
	var out []model.Milestone
	err := paginate(ctx, c, fmt.Sprintf("/repos/%s/%s/milestones", owner, repo), url.Values{"state": {"all"}}, func(items []ghMilestone) error {
		for _, item := range items {
			out = append(out, convertMilestone(item))
		}
		return nil
	})
	return out, err
}

func (c *Client) releases(ctx context.Context, owner, repo string) ([]model.Release, error) {
	var out []model.Release
	err := paginate(ctx, c, fmt.Sprintf("/repos/%s/%s/releases", owner, repo), nil, func(items []ghRelease) error {
		for _, item := range items {
			out = append(out, convertRelease(item))
		}
		return nil
	})
	return out, err
}

func convertIssue(item ghIssue) model.Issue {
	labels := make([]string, 0, len(item.Labels))
	for _, label := range item.Labels {
		labels = append(labels, label.Name)
	}
	var milestone string
	if item.Milestone != nil {
		milestone = item.Milestone.Title
	}
	return model.Issue{
		ID:          stableID("github", "issue", strconv.FormatInt(item.ID, 10)),
		SourceID:    item.ID,
		Number:      item.Number,
		Title:       item.Title,
		Body:        item.Body,
		State:       item.State,
		Author:      convertAuthor(item.User),
		Labels:      labels,
		Milestone:   milestone,
		Comments:    item.Comments,
		OriginalURL: item.HTMLURL,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		ClosedAt:    valueTime(item.ClosedAt),
	}
}

func convertComment(number int, item ghComment) model.Comment {
	return model.Comment{
		ID:          stableID("github", "issue_comment", strconv.FormatInt(item.ID, 10)),
		SourceID:    item.ID,
		IssueNumber: number,
		Author:      convertAuthor(item.User),
		Body:        item.Body,
		OriginalURL: item.HTMLURL,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func convertPullRequest(item ghPullRequest) model.PullRequest {
	return model.PullRequest{
		ID:          stableID("github", "pull_request", strconv.FormatInt(item.ID, 10)),
		SourceID:    item.ID,
		Number:      item.Number,
		Title:       item.Title,
		Body:        item.Body,
		State:       item.State,
		Author:      convertAuthor(item.User),
		BaseRef:     item.Base.Ref,
		HeadRef:     item.Head.Ref,
		Merged:      item.Merged,
		OriginalURL: item.HTMLURL,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		ClosedAt:    valueTime(item.ClosedAt),
		MergedAt:    valueTime(item.MergedAt),
	}
}

func convertReviewComment(number int, item ghReviewComment) model.ReviewComment {
	return model.ReviewComment{
		ID:                stableID("github", "review_comment", strconv.FormatInt(item.ID, 10)),
		SourceID:          item.ID,
		PullRequestNumber: number,
		Author:            convertAuthor(item.User),
		Body:              item.Body,
		Path:              item.Path,
		Position:          item.Position,
		Line:              item.Line,
		OriginalURL:       item.HTMLURL,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func convertLabel(item ghLabel) model.Label {
	return model.Label{
		ID:          stableID("github", "label", strconv.FormatInt(item.ID, 10)),
		SourceID:    item.ID,
		Name:        item.Name,
		Color:       item.Color,
		Description: item.Description,
	}
}

func convertMilestone(item ghMilestone) model.Milestone {
	return model.Milestone{
		ID:          stableID("github", "milestone", strconv.FormatInt(item.ID, 10)),
		SourceID:    item.ID,
		Number:      item.Number,
		Title:       item.Title,
		Description: item.Description,
		State:       item.State,
		OriginalURL: item.HTMLURL,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		ClosedAt:    valueTime(item.ClosedAt),
		DueOn:       valueTime(item.DueOn),
	}
}

func convertRelease(item ghRelease) model.Release {
	return model.Release{
		ID:          stableID("github", "release", strconv.FormatInt(item.ID, 10)),
		SourceID:    item.ID,
		TagName:     item.TagName,
		Name:        item.Name,
		Body:        item.Body,
		Author:      convertAuthor(item.Author),
		Draft:       item.Draft,
		Prerelease:  item.Prerelease,
		OriginalURL: item.HTMLURL,
		CreatedAt:   item.CreatedAt,
		PublishedAt: valueTime(item.PublishedAt),
	}
}

func convertAuthor(user ghUser) model.Author {
	return model.Author{
		ID:        user.ID,
		Login:     user.Login,
		URL:       user.HTMLURL,
		AvatarURL: user.AvatarURL,
	}
}

func valueTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
