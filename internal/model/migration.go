// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type MigrationPlan struct {
	Version     string                `json:"version"`
	CreatedAt   time.Time             `json:"created_at"`
	ToolVersion string                `json:"tool_version"`
	From        string                `json:"from"`
	To          string                `json:"to"`
	Strategy    MigrationPlanStrategy `json:"strategy"`
	Records     []MigrationPlanRecord `json:"records"`
	Warnings    []string              `json:"warnings,omitempty"`
}

type MigrationPlanStrategy struct {
	Numbering         string `json:"numbering_strategy"`
	AuthorMapping     string `json:"author_mapping_strategy"`
	LabelMapping      string `json:"label_mapping_strategy"`
	MilestoneMapping  string `json:"milestone_mapping_strategy"`
	StateMapping      string `json:"state_mapping_strategy"`
	ChangeProposal    string `json:"change_proposal_strategy"`
	Timestamp         string `json:"timestamp_strategy"`
	Collision         string `json:"collision_strategy"`
	Attachment        string `json:"attachment_strategy"`
	Visibility        string `json:"visibility_strategy"`
	Comment           string `json:"comment_strategy"`
	UnsupportedRecord string `json:"unsupported_record_strategy"`
	TargetWrite       string `json:"target_write_strategy"`
}

type MigrationPlanRecord struct {
	Object            string   `json:"object"`
	SourceID          string   `json:"source_id"`
	SourceNumber      int      `json:"source_number,omitempty"`
	SourceURL         string   `json:"source_url,omitempty"`
	WaystoneID        string   `json:"waystone_id"`
	TargetSource      string   `json:"target_source"`
	TargetKey         string   `json:"target_key"`
	NumberingStrategy string   `json:"numbering_strategy"`
	UnsupportedFields []string `json:"unsupported_fields,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
}
