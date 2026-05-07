// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type Identity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Algorithm string    `json:"algorithm"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}

type GitHubImport struct {
	Project        Project         `json:"project"`
	Source         Source          `json:"source"`
	ImportedAt     time.Time       `json:"imported_at"`
	Issues         []Issue         `json:"issues"`
	Comments       []Comment       `json:"comments"`
	PullRequests   []PullRequest   `json:"pull_requests"`
	ReviewComments []ReviewComment `json:"review_comments"`
	Labels         []Label         `json:"labels"`
	Milestones     []Milestone     `json:"milestones"`
	Releases       []Release       `json:"releases"`
}

type GitHubAudit struct {
	ID                  string                    `json:"id"`
	Repository          GitHubAuditRepository     `json:"repository"`
	Source              Source                    `json:"source"`
	GeneratedAt         time.Time                 `json:"generated_at"`
	Portable            []string                  `json:"portable"`
	NeedsMigrationPlan  []string                  `json:"needs_migration_plan"`
	Unsupported         []string                  `json:"unsupported,omitempty"`
	Warnings            []string                  `json:"warnings,omitempty"`
	Limitations         []string                  `json:"limitations,omitempty"`
	Workflows           []GitHubWorkflow          `json:"workflows,omitempty"`
	Actions             []GitHubActionUse         `json:"actions,omitempty"`
	Dependabot          GitHubAuditPresence       `json:"dependabot"`
	CodeQL              GitHubAuditPresence       `json:"codeql"`
	IssueTemplates      GitHubAuditPresence       `json:"issue_templates"`
	PullRequestTemplate GitHubAuditPresence       `json:"pull_request_template"`
	Codeowners          GitHubAuditPresence       `json:"codeowners"`
	BranchProtection    GitHubBranchProtection    `json:"branch_protection"`
	Secrets             GitHubAuditCount          `json:"secrets"`
	Variables           GitHubAuditCount          `json:"variables"`
	Environments        GitHubAuditCount          `json:"environments"`
	Pages               GitHubAuditPresence       `json:"pages"`
	ReleaseAssets       GitHubReleaseAssets       `json:"release_assets"`
	Evidence            []GitHubAuditEvidenceItem `json:"evidence,omitempty"`
}

type GitHubAuditRepository struct {
	ID            int64     `json:"id"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description,omitempty"`
	URL           string    `json:"url"`
	DefaultBranch string    `json:"default_branch,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type GitHubWorkflow struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Actions int    `json:"actions"`
}

type GitHubActionUse struct {
	Workflow string `json:"workflow"`
	Value    string `json:"value"`
	Kind     string `json:"kind"`
}

type GitHubAuditPresence struct {
	Present                bool     `json:"present"`
	Paths                  []string `json:"paths,omitempty"`
	Inaccessible           bool     `json:"inaccessible,omitempty"`
	InaccessibleStatusCode int      `json:"inaccessible_status_code,omitempty"`
}

type GitHubAuditEvidenceItem struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
}

type GitHubBranchProtection struct {
	Present                bool `json:"present"`
	RequiredStatusChecks   int  `json:"required_status_checks"`
	RequiredReviews        bool `json:"required_reviews"`
	RequiredApprovals      int  `json:"required_approvals,omitempty"`
	CodeOwnerReviews       bool `json:"code_owner_reviews"`
	AdminEnforcement       bool `json:"admin_enforcement"`
	Inaccessible           bool `json:"inaccessible,omitempty"`
	InaccessibleStatusCode int  `json:"inaccessible_status_code,omitempty"`
}

type GitHubAuditCount struct {
	Accessible             bool `json:"accessible"`
	Count                  int  `json:"count"`
	Inaccessible           bool `json:"inaccessible,omitempty"`
	InaccessibleStatusCode int  `json:"inaccessible_status_code,omitempty"`
}

type GitHubReleaseAssets struct {
	Releases int `json:"releases"`
	Assets   int `json:"assets"`
}

type Ledger struct {
	Version       string    `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DefaultSource *Source   `json:"default_source,omitempty"`
}

type Source struct {
	System     string               `json:"system"`
	Owner      string               `json:"owner"`
	Repo       string               `json:"repo"`
	URL        string               `json:"url"`
	Objects    []SourceObjectRef    `json:"objects,omitempty"`
	Operations []SourceOperationRef `json:"operations,omitempty"`
	Signature  *OperationSignature  `json:"signature,omitempty"`
}

type SourceObjectRef struct {
	Object string `json:"object"`
	Number int    `json:"number,omitempty"`
	ID     string `json:"id,omitempty"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type SourceOperationRef struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Path      string    `json:"path"`
	SHA256    string    `json:"sha256,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

type Provenance struct {
	ImportID string `json:"import_id"`
	Source   Source `json:"source"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Author struct {
	ID        int64  `json:"id,omitempty"`
	Login     string `json:"login,omitempty"`
	Name      string `json:"name,omitempty"`
	URL       string `json:"url,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type Issue struct {
	Provenance
	ID          string    `json:"id"`
	SourceID    int64     `json:"source_id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body,omitempty"`
	State       string    `json:"state"`
	Author      Author    `json:"author"`
	Labels      []string  `json:"labels,omitempty"`
	Milestone   string    `json:"milestone,omitempty"`
	Comments    int       `json:"comments"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
}

type Comment struct {
	Provenance
	ID          string    `json:"id"`
	SourceID    int64     `json:"source_id"`
	IssueNumber int       `json:"issue_number"`
	Author      Author    `json:"author"`
	Body        string    `json:"body,omitempty"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type IssueEvent struct {
	Provenance
	ID          string    `json:"id"`
	IssueNumber int       `json:"issue_number"`
	Type        string    `json:"type"`
	Author      Author    `json:"author"`
	Title       string    `json:"title,omitempty"`
	Body        string    `json:"body,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type PullRequest struct {
	Provenance
	ID          string    `json:"id"`
	SourceID    int64     `json:"source_id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body,omitempty"`
	State       string    `json:"state"`
	Author      Author    `json:"author"`
	BaseRef     string    `json:"base_ref"`
	HeadRef     string    `json:"head_ref"`
	Merged      bool      `json:"merged"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
	MergedAt    time.Time `json:"merged_at,omitempty"`
}

type ReviewComment struct {
	Provenance
	ID                string    `json:"id"`
	SourceID          int64     `json:"source_id"`
	PullRequestNumber int       `json:"pull_request_number"`
	Author            Author    `json:"author"`
	Body              string    `json:"body,omitempty"`
	Path              string    `json:"path,omitempty"`
	Position          int       `json:"position,omitempty"`
	Line              int       `json:"line,omitempty"`
	OriginalURL       string    `json:"original_url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Label struct {
	Provenance
	ID          string `json:"id"`
	SourceID    int64  `json:"source_id"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

type Milestone struct {
	Provenance
	ID          string    `json:"id"`
	SourceID    int64     `json:"source_id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	State       string    `json:"state"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
	DueOn       time.Time `json:"due_on,omitempty"`
}

type Release struct {
	Provenance
	ID          string    `json:"id"`
	SourceID    int64     `json:"source_id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name,omitempty"`
	Body        string    `json:"body,omitempty"`
	Author      Author    `json:"author"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}
