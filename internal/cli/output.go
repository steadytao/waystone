// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/steadytao/waystone/internal/github"
	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func writeField(w io.Writer, label string, value any) {
	fmt.Fprintf(w, "%-14s %v\n", label, value)
}

func writeIndentedField(w io.Writer, label string, value any) {
	fmt.Fprintf(w, "  %-16s %v\n", label, value)
}

func writeIssueComments(w io.Writer, number int, comments []model.Comment) {
	fmt.Fprintln(w)
	writeField(w, "Issue", fmt.Sprintf("#%d", number))
	writeField(w, "Comments", len(comments))
	for _, comment := range comments {
		fmt.Fprintln(w)
		writeIndentedField(w, "Source", ledger.SourceSpec(comment.Source))
		writeIndentedField(w, "Author", comment.Author.Login)
		writeIndentedField(w, "Created", comment.CreatedAt.Format(time.RFC3339))
		if comment.OriginalURL != "" {
			writeIndentedField(w, "URL", comment.OriginalURL)
		}
		if comment.Body != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, comment.Body)
		}
	}
}

func writePullRequestComments(w io.Writer, number int, comments []model.ReviewComment) {
	fmt.Fprintln(w)
	writeField(w, "Pull request", fmt.Sprintf("#%d", number))
	writeField(w, "Review comments", len(comments))
	for _, comment := range comments {
		fmt.Fprintln(w)
		writeIndentedField(w, "Source", ledger.SourceSpec(comment.Source))
		writeIndentedField(w, "Author", comment.Author.Login)
		writeIndentedField(w, "Path", comment.Path)
		writeIndentedField(w, "Line", comment.Line)
		writeIndentedField(w, "URL", comment.OriginalURL)
		if comment.Body != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, comment.Body)
		}
	}
}

type timelineEvent struct {
	Time   time.Time `json:"time"`
	Type   string    `json:"type"`
	Author string    `json:"author,omitempty"`
	Title  string    `json:"title,omitempty"`
	Body   string    `json:"body,omitempty"`
	URL    string    `json:"url,omitempty"`
	Path   string    `json:"path,omitempty"`
	Line   int       `json:"line,omitempty"`
}

func issueTimeline(issue model.Issue, comments []model.Comment, issueEvents []model.IssueEvent) []timelineEvent {
	events := []timelineEvent{
		{
			Time:   issue.CreatedAt,
			Type:   "issue.opened",
			Author: issue.Author.Login,
			Title:  issue.Title,
			Body:   issue.Body,
			URL:    issue.OriginalURL,
		},
	}
	for _, comment := range comments {
		events = append(events, timelineEvent{
			Time:   comment.CreatedAt,
			Type:   "issue.comment",
			Author: comment.Author.Login,
			Body:   comment.Body,
			URL:    comment.OriginalURL,
		})
	}
	for _, event := range issueEvents {
		events = append(events, timelineEvent{
			Time:   event.CreatedAt,
			Type:   event.Type,
			Author: event.Author.Login,
		})
	}
	if !issue.ClosedAt.IsZero() && !hasTimelineEvent(issueEvents, "issue.closed") {
		events = append(events, timelineEvent{
			Time:   issue.ClosedAt,
			Type:   "issue.closed",
			Author: issue.Author.Login,
			URL:    issue.OriginalURL,
		})
	}
	return sortTimeline(events)
}

func hasTimelineEvent(events []model.IssueEvent, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func pullRequestTimeline(pr model.PullRequest, conversationComments []model.Comment, reviewComments []model.ReviewComment) []timelineEvent {
	events := []timelineEvent{
		{
			Time:   pr.CreatedAt,
			Type:   "pull_request.opened",
			Author: pr.Author.Login,
			Title:  pr.Title,
			Body:   pr.Body,
			URL:    pr.OriginalURL,
		},
	}
	for _, comment := range conversationComments {
		events = append(events, timelineEvent{
			Time:   comment.CreatedAt,
			Type:   "pull_request.comment",
			Author: comment.Author.Login,
			Body:   comment.Body,
			URL:    comment.OriginalURL,
		})
	}
	for _, comment := range reviewComments {
		events = append(events, timelineEvent{
			Time:   comment.CreatedAt,
			Type:   "pull_request.review_comment",
			Author: comment.Author.Login,
			Body:   comment.Body,
			URL:    comment.OriginalURL,
			Path:   comment.Path,
			Line:   comment.Line,
		})
	}
	if !pr.MergedAt.IsZero() {
		events = append(events, timelineEvent{
			Time:   pr.MergedAt,
			Type:   "pull_request.merged",
			Author: pr.Author.Login,
			URL:    pr.OriginalURL,
		})
	} else if !pr.ClosedAt.IsZero() {
		events = append(events, timelineEvent{
			Time:   pr.ClosedAt,
			Type:   "pull_request.closed",
			Author: pr.Author.Login,
			URL:    pr.OriginalURL,
		})
	}
	return sortTimeline(events)
}

func sortTimeline(events []timelineEvent) []timelineEvent {
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time.Equal(events[j].Time) {
			if events[i].Type == events[j].Type {
				return events[i].URL < events[j].URL
			}
			return events[i].Type < events[j].Type
		}
		return events[i].Time.Before(events[j].Time)
	})
	return events
}

func writeTimeline(w io.Writer, kind string, number int, source string, events []timelineEvent) {
	writeField(w, kind, fmt.Sprintf("#%d", number))
	writeField(w, "Source", source)
	writeField(w, "Events", len(events))
	for _, event := range events {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s  %s", event.Time.Format(time.RFC3339), event.Type)
		if event.Author != "" {
			fmt.Fprintf(w, "  %s", event.Author)
		}
		fmt.Fprintln(w)
		if event.Title != "" {
			writeIndentedField(w, "Title", event.Title)
		}
		if event.Path != "" {
			writeIndentedField(w, "Path", event.Path)
		}
		if event.Line > 0 {
			writeIndentedField(w, "Line", event.Line)
		}
		if event.URL != "" {
			writeIndentedField(w, "URL", event.URL)
		}
		if event.Body != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, event.Body)
		}
	}
}

func writeJSONOutput(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printImportProgress(w io.Writer, progress github.Progress, verbose bool) {
	if progress.Detail && !verbose {
		return
	}
	if progress.Detail {
		fmt.Fprintf(w, "  - %s...\n", progress.Message)
		return
	}
	fmt.Fprintf(w, "- %s...\n", progress.Message)
}

func writeGitHubAudit(w io.Writer, audit model.GitHubAudit, verbose bool) {
	writeField(w, "Repository", audit.Repository.FullName)
	writeField(w, "URL", audit.Repository.URL)
	writeField(w, "Default branch", audit.Repository.DefaultBranch)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Portable")
	for _, item := range audit.Portable {
		fmt.Fprintf(w, "- %s\n", item)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Needs migration plan")
	if len(audit.NeedsMigrationPlan) == 0 {
		fmt.Fprintln(w, "- none detected in this audit slice")
	} else {
		for _, item := range audit.NeedsMigrationPlan {
			fmt.Fprintf(w, "- %s\n", item)
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Evidence")
	if len(audit.Workflows) == 0 && len(audit.Actions) == 0 {
		fmt.Fprintln(w, "- no workflow evidence found")
	} else {
		for _, workflow := range audit.Workflows {
			fmt.Fprintf(w, "- workflow %s\n", workflow.Path)
		}
		writeActionSummary(w, audit.Actions)
		if verbose {
			for _, action := range audit.Actions {
				fmt.Fprintf(w, "- action %s (%s, %s)\n", action.Value, action.Kind, action.Workflow)
			}
		}
	}
	writePresence(w, "Dependabot", audit.Dependabot)
	writePresence(w, "CodeQL", audit.CodeQL)
	writePresence(w, "Issue templates", audit.IssueTemplates)
	writePresence(w, "Pull request template", audit.PullRequestTemplate)
	writePresence(w, "CODEOWNERS", audit.Codeowners)
	writeBranchProtection(w, audit.BranchProtection)
	writeAuditCount(w, "Repository secrets", audit.Secrets)
	writeAuditCount(w, "Repository variables", audit.Variables)
	writeAuditCount(w, "Environments", audit.Environments)
	writePresence(w, "GitHub Pages", audit.Pages)
	if audit.ReleaseAssets.Releases > 0 || audit.ReleaseAssets.Assets > 0 {
		fmt.Fprintf(w, "- Release assets %d across %d releases\n", audit.ReleaseAssets.Assets, audit.ReleaseAssets.Releases)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Limitations")
	for _, limitation := range audit.Limitations {
		fmt.Fprintf(w, "- %s\n", limitation)
	}
}

func writeActionSummary(w io.Writer, actions []model.GitHubActionUse) {
	if len(actions) == 0 {
		return
	}
	counts := map[string]int{}
	for _, action := range actions {
		counts[action.Kind]++
	}
	fmt.Fprintln(w, "- Actions")
	for _, kind := range []string{"remote", "local", "reusable_workflow"} {
		if counts[kind] > 0 {
			fmt.Fprintf(w, "  - %s %d\n", kind, counts[kind])
		}
	}
}

func writePresence(w io.Writer, label string, presence model.GitHubAuditPresence) {
	if presence.Inaccessible {
		fmt.Fprintf(w, "- %s inaccessible status=%d\n", label, presence.InaccessibleStatusCode)
		return
	}
	if !presence.Present {
		return
	}
	for _, path := range presence.Paths {
		fmt.Fprintf(w, "- %s %s\n", label, path)
	}
}

func writeBranchProtection(w io.Writer, protection model.GitHubBranchProtection) {
	switch {
	case protection.Present:
		fmt.Fprintf(w, "- Branch protection required_checks=%d required_reviews=%t approvals=%d code_owner_reviews=%t admin_enforcement=%t\n",
			protection.RequiredStatusChecks,
			protection.RequiredReviews,
			protection.RequiredApprovals,
			protection.CodeOwnerReviews,
			protection.AdminEnforcement,
		)
	case protection.Inaccessible:
		fmt.Fprintf(w, "- Branch protection inaccessible status=%d\n", protection.InaccessibleStatusCode)
	}
}

func writeAuditCount(w io.Writer, label string, count model.GitHubAuditCount) {
	if count.Inaccessible {
		fmt.Fprintf(w, "- %s inaccessible status=%d\n", label, count.InaccessibleStatusCode)
		return
	}
	if !count.Accessible || count.Count == 0 {
		return
	}
	fmt.Fprintf(w, "- %s %d\n", label, count.Count)
}
