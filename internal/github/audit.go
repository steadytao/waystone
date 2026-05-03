// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

var actionUsePattern = regexp.MustCompile(`(?m)^\s*(?:-\s*)?uses:\s*['"]?([^'"\s]+)`)

func (c *Client) AuditRepository(ctx context.Context, owner, repo string) (model.GitHubAudit, error) {
	c.report("Fetching repository metadata")
	repository, err := c.repository(ctx, owner, repo)
	if err != nil {
		return model.GitHubAudit{}, err
	}
	source := model.Source{
		System: "github",
		Owner:  owner,
		Repo:   repo,
		URL:    repository.HTMLURL,
	}
	audit := model.GitHubAudit{
		Repository: model.GitHubAuditRepository{
			ID:            repository.ID,
			FullName:      repository.FullName,
			Description:   repository.Description,
			URL:           repository.HTMLURL,
			DefaultBranch: repository.DefaultBranch,
			CreatedAt:     repository.CreatedAt,
			UpdatedAt:     repository.UpdatedAt,
		},
		Source:      source,
		GeneratedAt: time.Now().UTC(),
		Portable: []string{
			"issues",
			"pull requests",
			"comments",
			"labels",
			"milestones",
			"releases",
		},
	}

	c.report("Inspecting workflow files")
	workflows, actions, err := c.auditWorkflows(ctx, owner, repo, repository.DefaultBranch)
	if err != nil {
		return model.GitHubAudit{}, err
	}
	audit.Workflows = workflows
	audit.Actions = actions
	if len(workflows) > 0 {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "GitHub Actions workflows")
		for _, workflow := range workflows {
			audit.Evidence = append(audit.Evidence, model.GitHubAuditEvidenceItem{Type: "workflow", Path: workflow.Path, Name: workflow.Name})
		}
	}

	audit.Dependabot, err = c.auditPresence(ctx, owner, repo, repository.DefaultBranch, ".github/dependabot.yml")
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.Dependabot.Present {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "Dependabot")
	}
	audit.CodeQL, err = c.auditAnyPresence(ctx, owner, repo, repository.DefaultBranch, ".github/codeql.yml", ".github/codeql/codeql-config.yml", ".github/workflows/codeql.yml")
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.CodeQL.Present {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "CodeQL")
	}
	audit.IssueTemplates, err = c.auditDirectoryPresence(ctx, owner, repo, repository.DefaultBranch, ".github/ISSUE_TEMPLATE")
	if err != nil {
		return model.GitHubAudit{}, err
	}
	audit.PullRequestTemplate, err = c.auditAnyPresence(ctx, owner, repo, repository.DefaultBranch, ".github/PULL_REQUEST_TEMPLATE.md", "PULL_REQUEST_TEMPLATE.md")
	if err != nil {
		return model.GitHubAudit{}, err
	}
	audit.Codeowners, err = c.auditAnyPresence(ctx, owner, repo, repository.DefaultBranch, "CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS")
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.Codeowners.Present {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "CODEOWNERS")
	}
	audit.BranchProtection, err = c.auditBranchProtection(ctx, owner, repo, repository.DefaultBranch)
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.BranchProtection.Present {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "branch protection")
	}
	audit.Secrets, err = c.auditCount(ctx, fmt.Sprintf("/repos/%s/%s/actions/secrets", owner, repo))
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.Secrets.Accessible && audit.Secrets.Count > 0 {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "repository secrets")
	}
	audit.Variables, err = c.auditCount(ctx, fmt.Sprintf("/repos/%s/%s/actions/variables", owner, repo))
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.Variables.Accessible && audit.Variables.Count > 0 {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "repository variables")
	}
	audit.Environments, err = c.auditCount(ctx, fmt.Sprintf("/repos/%s/%s/environments", owner, repo))
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.Environments.Accessible && audit.Environments.Count > 0 {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "environments")
	}
	audit.Pages, err = c.auditPages(ctx, owner, repo)
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.Pages.Present {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "GitHub Pages")
	}
	audit.ReleaseAssets, err = c.auditReleaseAssets(ctx, owner, repo)
	if err != nil {
		return model.GitHubAudit{}, err
	}
	if audit.ReleaseAssets.Assets > 0 {
		audit.NeedsMigrationPlan = append(audit.NeedsMigrationPlan, "release assets")
	}
	audit.Limitations = append(audit.Limitations,
		"does not inspect secret values",
		"does not execute workflows or repository code",
		"does not yet inspect webhooks or packages",
	)
	return audit, nil
}

func (c *Client) auditWorkflows(ctx context.Context, owner, repo, ref string) ([]model.GitHubWorkflow, []model.GitHubActionUse, error) {
	var entries []ghContent
	ok, err := c.getOptional(ctx, fmt.Sprintf("/repos/%s/%s/contents/.github/workflows", owner, repo), refQuery(ref), &entries)
	if err != nil || !ok {
		return nil, nil, err
	}
	var workflows []model.GitHubWorkflow
	var actions []model.GitHubActionUse
	for _, entry := range entries {
		if entry.Type != "file" || !isWorkflowFile(entry.Name) {
			continue
		}
		content, err := c.content(ctx, owner, repo, entry.Path, ref)
		if err != nil {
			return nil, nil, err
		}
		uses := parseActionUses(entry.Path, string(content))
		workflows = append(workflows, model.GitHubWorkflow{Name: entry.Name, Path: entry.Path, Actions: len(uses)})
		actions = append(actions, uses...)
	}
	sort.Slice(workflows, func(i, j int) bool { return workflows[i].Path < workflows[j].Path })
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Workflow == actions[j].Workflow {
			return actions[i].Value < actions[j].Value
		}
		return actions[i].Workflow < actions[j].Workflow
	})
	return workflows, actions, nil
}

func (c *Client) content(ctx context.Context, owner, repo, filePath, ref string) ([]byte, error) {
	var item ghContent
	ok, err := c.getOptional(ctx, fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, filePath), refQuery(ref), &item)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("github content %s not found", filePath)
	}
	return decodeContent(item)
}

func (c *Client) auditPresence(ctx context.Context, owner, repo, ref, filePath string) (model.GitHubAuditPresence, error) {
	var item ghContent
	ok, err := c.getOptional(ctx, fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, filePath), refQuery(ref), &item)
	if err != nil {
		return model.GitHubAuditPresence{}, err
	}
	if !ok {
		return model.GitHubAuditPresence{}, nil
	}
	return model.GitHubAuditPresence{Present: true, Paths: []string{filePath}}, nil
}

func (c *Client) auditAnyPresence(ctx context.Context, owner, repo, ref string, paths ...string) (model.GitHubAuditPresence, error) {
	var found []string
	for _, filePath := range paths {
		presence, err := c.auditPresence(ctx, owner, repo, ref, filePath)
		if err != nil {
			return model.GitHubAuditPresence{}, err
		}
		if presence.Present {
			found = append(found, presence.Paths...)
		}
	}
	return model.GitHubAuditPresence{Present: len(found) > 0, Paths: found}, nil
}

func (c *Client) auditDirectoryPresence(ctx context.Context, owner, repo, ref, dirPath string) (model.GitHubAuditPresence, error) {
	var entries []ghContent
	ok, err := c.getOptional(ctx, fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, dirPath), refQuery(ref), &entries)
	if err != nil {
		return model.GitHubAuditPresence{}, err
	}
	if !ok {
		return model.GitHubAuditPresence{}, nil
	}
	var paths []string
	for _, entry := range entries {
		if entry.Type == "file" {
			paths = append(paths, entry.Path)
		}
	}
	return model.GitHubAuditPresence{Present: len(paths) > 0, Paths: paths}, nil
}

func (c *Client) auditBranchProtection(ctx context.Context, owner, repo, branch string) (model.GitHubBranchProtection, error) {
	if branch == "" {
		return model.GitHubBranchProtection{}, nil
	}
	var protection ghBranchProtection
	ok, status, err := c.getMaybe(ctx, fmt.Sprintf("/repos/%s/%s/branches/%s/protection", owner, repo, branch), nil, &protection)
	if err != nil {
		return model.GitHubBranchProtection{}, err
	}
	if !ok {
		if status == 0 || status == http.StatusNotFound {
			return model.GitHubBranchProtection{}, nil
		}
		return model.GitHubBranchProtection{Inaccessible: true, InaccessibleStatusCode: status}, nil
	}
	audit := model.GitHubBranchProtection{Present: true}
	if protection.RequiredStatusChecks != nil {
		audit.RequiredStatusChecks = len(protection.RequiredStatusChecks.Contexts) + len(protection.RequiredStatusChecks.Checks)
	}
	if protection.RequiredPullRequestReviews != nil {
		audit.RequiredReviews = true
		audit.RequiredApprovals = protection.RequiredPullRequestReviews.RequiredApprovingReviewCount
		audit.CodeOwnerReviews = protection.RequiredPullRequestReviews.RequireCodeOwnerReviews
	}
	if protection.EnforceAdmins != nil {
		audit.AdminEnforcement = protection.EnforceAdmins.Enabled
	}
	return audit, nil
}

func (c *Client) auditCount(ctx context.Context, path string) (model.GitHubAuditCount, error) {
	var count ghCountEnvelope
	ok, _, err := c.getMaybe(ctx, path, nil, &count)
	if err != nil {
		return model.GitHubAuditCount{}, err
	}
	if !ok {
		return model.GitHubAuditCount{}, nil
	}
	return model.GitHubAuditCount{Accessible: true, Count: count.TotalCount}, nil
}

func (c *Client) auditPages(ctx context.Context, owner, repo string) (model.GitHubAuditPresence, error) {
	var pages ghPages
	ok, _, err := c.getMaybe(ctx, fmt.Sprintf("/repos/%s/%s/pages", owner, repo), nil, &pages)
	if err != nil {
		return model.GitHubAuditPresence{}, err
	}
	if !ok {
		return model.GitHubAuditPresence{}, nil
	}
	return model.GitHubAuditPresence{Present: true, Paths: []string{pages.HTMLURL}}, nil
}

func (c *Client) auditReleaseAssets(ctx context.Context, owner, repo string) (model.GitHubReleaseAssets, error) {
	var releases int
	var assets int
	err := paginate[ghRelease](ctx, c, fmt.Sprintf("/repos/%s/%s/releases", owner, repo), nil, func(items []ghRelease) error {
		releases += len(items)
		for _, release := range items {
			assets += len(release.Assets)
		}
		return nil
	})
	if err != nil {
		return model.GitHubReleaseAssets{}, err
	}
	return model.GitHubReleaseAssets{Releases: releases, Assets: assets}, nil
}

func parseActionUses(workflow, content string) []model.GitHubActionUse {
	var uses []model.GitHubActionUse
	seen := map[string]bool{}
	matches := actionUsePattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		value := strings.TrimSpace(match[1])
		if seen[value] {
			continue
		}
		seen[value] = true
		uses = append(uses, model.GitHubActionUse{
			Workflow: workflow,
			Value:    value,
			Kind:     actionKind(value),
		})
	}
	return uses
}

func actionKind(value string) string {
	switch {
	case strings.HasPrefix(value, "./"):
		return "local"
	case strings.Contains(value, ".github/workflows/"):
		return "reusable_workflow"
	default:
		return "remote"
	}
}

func isWorkflowFile(name string) bool {
	return strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")
}

func refQuery(ref string) url.Values {
	if ref == "" {
		return nil
	}
	return url.Values{"ref": {ref}}
}
