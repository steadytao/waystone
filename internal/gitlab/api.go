// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import "time"

type glProject struct {
	ID                int64     `json:"id"`
	PathWithNamespace string    `json:"path_with_namespace"`
	Description       string    `json:"description"`
	WebURL            string    `json:"web_url"`
	CreatedAt         time.Time `json:"created_at"`
	LastActivityAt    time.Time `json:"last_activity_at"`
}

type glAuthor struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	WebURL    string `json:"web_url"`
	AvatarURL string `json:"avatar_url"`
}

type glIssue struct {
	ID             int64        `json:"id"`
	IID            int          `json:"iid"`
	Title          string       `json:"title"`
	Description    string       `json:"description"`
	State          string       `json:"state"`
	Author         glAuthor     `json:"author"`
	Labels         []string     `json:"labels"`
	Milestone      *glMilestone `json:"milestone"`
	UserNotesCount int          `json:"user_notes_count"`
	WebURL         string       `json:"web_url"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	ClosedAt       *time.Time   `json:"closed_at"`
	Confidential   bool         `json:"confidential"`
}

type glNote struct {
	ID        int64     `json:"id"`
	Author    glAuthor  `json:"author"`
	Body      string    `json:"body"`
	System    bool      `json:"system"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type glMergeRequest struct {
	ID           int64      `json:"id"`
	IID          int        `json:"iid"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	Author       glAuthor   `json:"author"`
	WebURL       string     `json:"web_url"`
	SourceBranch string     `json:"source_branch"`
	TargetBranch string     `json:"target_branch"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at"`
	MergedAt     *time.Time `json:"merged_at"`
}

type glLabel struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type glMilestone struct {
	ID          int64     `json:"id"`
	IID         int       `json:"iid"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	WebURL      string    `json:"web_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DueDate     string    `json:"due_date"`
}

type glRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Author      glAuthor  `json:"author"`
	CreatedAt   time.Time `json:"created_at"`
	ReleasedAt  time.Time `json:"released_at"`
	Links       struct {
		Self string `json:"self"`
	} `json:"_links"`
}
