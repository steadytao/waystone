// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import "time"

type ghUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

type ghRepository struct {
	ID            int64     `json:"id"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description"`
	HTMLURL       string    `json:"html_url"`
	DefaultBranch string    `json:"default_branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ghContent struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

type ghIssue struct {
	ID          int64        `json:"id"`
	Number      int          `json:"number"`
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	State       string       `json:"state"`
	User        ghUser       `json:"user"`
	Labels      []ghLabel    `json:"labels"`
	Milestone   *ghMilestone `json:"milestone"`
	Comments    int          `json:"comments"`
	HTMLURL     string       `json:"html_url"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	ClosedAt    *time.Time   `json:"closed_at"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
}

type ghComment struct {
	ID        int64     `json:"id"`
	User      ghUser    `json:"user"`
	Body      string    `json:"body"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ghPullRequest struct {
	ID        int64      `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	User      ghUser     `json:"user"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	MergedAt  *time.Time `json:"merged_at"`
	Merged    bool       `json:"merged"`
	Base      struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

type ghReviewComment struct {
	ID        int64     `json:"id"`
	User      ghUser    `json:"user"`
	Body      string    `json:"body"`
	HTMLURL   string    `json:"html_url"`
	Path      string    `json:"path"`
	Position  int       `json:"position"`
	Line      int       `json:"line"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ghLabel struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type ghMilestone struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	HTMLURL     string     `json:"html_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	DueOn       *time.Time `json:"due_on"`
}

type ghRelease struct {
	ID          int64      `json:"id"`
	TagName     string     `json:"tag_name"`
	Name        string     `json:"name"`
	Body        string     `json:"body"`
	Author      ghUser     `json:"author"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	HTMLURL     string     `json:"html_url"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at"`
	Assets      []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"assets"`
}

type ghBranchProtection struct {
	RequiredStatusChecks *struct {
		Contexts []string `json:"contexts"`
		Checks   []struct {
			Context string `json:"context"`
		} `json:"checks"`
	} `json:"required_status_checks"`
	RequiredPullRequestReviews *struct {
		RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
		RequireCodeOwnerReviews      bool `json:"require_code_owner_reviews"`
	} `json:"required_pull_request_reviews"`
	EnforceAdmins *struct {
		Enabled bool `json:"enabled"`
	} `json:"enforce_admins"`
}

type ghCountEnvelope struct {
	TotalCount int `json:"total_count"`
}

type ghPages struct {
	HTMLURL string `json:"html_url"`
}
