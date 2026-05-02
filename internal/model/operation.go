// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type Operation struct {
	ID                string            `json:"id"`
	PreviousOperation string            `json:"previous_operation,omitempty"`
	OperationHash     string            `json:"operation_hash,omitempty"`
	Command           string            `json:"command"`
	Args              []string          `json:"args,omitempty"`
	StartedAt         time.Time         `json:"started_at"`
	FinishedAt        time.Time         `json:"finished_at"`
	Actor             OperationActor    `json:"actor"`
	Auth              OperationAuth     `json:"auth,omitempty"`
	Input             map[string]string `json:"input,omitempty"`
	Output            OperationOutput   `json:"output"`
	Changes           []ObjectChange    `json:"changes,omitempty"`
}

type OperationActor struct {
	Source       string `json:"source"`
	User         string `json:"user,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
	GitUserName  string `json:"git_user_name,omitempty"`
	GitUserEmail string `json:"git_user_email,omitempty"`
}

type OperationAuth struct {
	Provider string `json:"provider,omitempty"`
	Mode     string `json:"mode,omitempty"`
	Login    string `json:"login,omitempty"`
}

type OperationOutput struct {
	Ledger    string        `json:"ledger"`
	Created   int           `json:"created"`
	Updated   int           `json:"updated"`
	Deleted   int           `json:"deleted"`
	Unchanged int           `json:"unchanged"`
	Summary   RecordSummary `json:"summary"`
}

type RecordSummary struct {
	Issues         int `json:"issues"`
	Comments       int `json:"comments"`
	PullRequests   int `json:"pull_requests"`
	ReviewComments int `json:"review_comments"`
	Labels         int `json:"labels"`
	Milestones     int `json:"milestones"`
	Releases       int `json:"releases"`
}

type ObjectChange struct {
	Type   string `json:"type"`
	Object string `json:"object"`
	Number int    `json:"number,omitempty"`
	ID     string `json:"id,omitempty"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}
