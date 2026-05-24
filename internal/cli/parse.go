// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func gitConfig(key string) string {
	var out []byte
	var err error
	switch key {
	case "user.name":
		out, err = exec.Command("git", "config", "--get", "user.name").Output()
	case "user.email":
		out, err = exec.Command("git", "config", "--get", "user.email").Output()
	default:
		return ""
	}
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseRepo(value string) (string, string, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be owner/repo, got %q", value)
	}
	if _, err := ledger.ParseSourceSpec("github:" + value); err != nil {
		return "", "", fmt.Errorf("repository must be safe owner/repo, got %q", value)
	}
	return parts[0], parts[1], nil
}

func parseNumber(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("number must be a positive integer, got %q", value)
	}
	return number, nil
}

func parseOptionalSource(value string) (model.Source, bool, error) {
	if value == "" {
		return model.Source{}, false, nil
	}
	source, err := ledger.ParseSourceSpec(value)
	if err != nil {
		return model.Source{}, false, err
	}
	return source, true, nil
}

func resolveOptionalSource(reader ledger.Reader, value string) (model.Source, bool, error) {
	source, ok, err := parseOptionalSource(value)
	if err != nil || ok {
		return source, ok, err
	}
	current, err := reader.Ledger()
	if err != nil {
		return model.Source{}, false, nil
	}
	if current.DefaultSource == nil {
		return model.Source{}, false, nil
	}
	return *current.DefaultSource, true, nil
}

func resolveRefreshSources(reader ledger.Reader, requested []string) ([]model.Source, error) {
	if len(requested) == 0 {
		return reader.Sources()
	}
	var sources []model.Source
	seen := map[string]bool{}
	for _, value := range requested {
		source, err := ledger.ParseSourceSpec(value)
		if err != nil {
			return nil, err
		}
		source, err = reader.Source(source)
		if err != nil {
			return nil, err
		}
		spec := ledger.SourceSpec(source)
		if seen[spec] {
			continue
		}
		seen[spec] = true
		sources = append(sources, source)
	}
	return sources, nil
}

type valueListFlag []string

func (f *valueListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *valueListFlag) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			*f = append(*f, item)
		}
	}
	return nil
}

type searchFieldsFlag []string

func (f *searchFieldsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *searchFieldsFlag) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			*f = append(*f, item)
		}
	}
	return nil
}

func issueSearchFields() map[string]func(model.Issue) string {
	return map[string]func(model.Issue) string{
		"title":       func(issue model.Issue) string { return issue.Title },
		"description": func(issue model.Issue) string { return issue.Body },
		"body":        func(issue model.Issue) string { return issue.Body },
		"author":      func(issue model.Issue) string { return issue.Author.Login },
		"state":       func(issue model.Issue) string { return issue.State },
		"label":       func(issue model.Issue) string { return strings.Join(issue.Labels, " ") },
		"labels":      func(issue model.Issue) string { return strings.Join(issue.Labels, " ") },
		"milestone":   func(issue model.Issue) string { return issue.Milestone },
		"url":         func(issue model.Issue) string { return issue.OriginalURL },
	}
}

func pullRequestSearchFields() map[string]func(model.PullRequest) string {
	return map[string]func(model.PullRequest) string{
		"title":       func(pr model.PullRequest) string { return pr.Title },
		"description": func(pr model.PullRequest) string { return pr.Body },
		"body":        func(pr model.PullRequest) string { return pr.Body },
		"author":      func(pr model.PullRequest) string { return pr.Author.Login },
		"state":       func(pr model.PullRequest) string { return pr.State },
		"branch":      func(pr model.PullRequest) string { return pr.BaseRef + " " + pr.HeadRef },
		"base":        func(pr model.PullRequest) string { return pr.BaseRef },
		"head":        func(pr model.PullRequest) string { return pr.HeadRef },
		"url":         func(pr model.PullRequest) string { return pr.OriginalURL },
	}
}

func normalizeSearchFields[T any](fields []string, allowed map[string]func(T) string) ([]string, error) {
	if len(fields) == 0 {
		return []string{"title", "description"}, nil
	}
	var normalized []string
	seen := map[string]bool{}
	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if field == "all" {
			normalized = normalized[:0]
			for allowedField := range allowed {
				normalized = append(normalized, allowedField)
			}
			sort.Strings(normalized)
			return normalized, nil
		}
		if _, ok := allowed[field]; !ok {
			return nil, fmt.Errorf("unsupported search field %q", field)
		}
		if !seen[field] {
			seen[field] = true
			normalized = append(normalized, field)
		}
	}
	return normalized, nil
}

func searchPullRequests(prs []model.PullRequest, query string, fields []string) []model.PullRequest {
	query = strings.ToLower(query)
	searchable := pullRequestSearchFields()
	var matches []model.PullRequest
	for _, pr := range prs {
		text := searchableText(pr, fields, searchable)
		if strings.Contains(text, query) {
			matches = append(matches, pr)
		}
	}
	return matches
}

func matchingPullRequestField(pr model.PullRequest, query string, fields []string) string {
	return matchingField(pr, query, fields, pullRequestSearchFields())
}

func matchingField[T any](value T, query string, fields []string, allowed map[string]func(T) string) string {
	query = strings.ToLower(query)
	for _, field := range fields {
		if read, ok := allowed[field]; ok && strings.Contains(strings.ToLower(read(value)), query) {
			if field == "body" {
				return "description"
			}
			return field
		}
	}
	return ""
}

func searchableText[T any](value T, fields []string, allowed map[string]func(T) string) string {
	var parts []string
	for _, field := range fields {
		if read, ok := allowed[field]; ok {
			parts = append(parts, read(value))
		}
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

func duplicateIssueNumbers(issues []model.Issue) []int {
	counts := map[int]int{}
	for _, issue := range issues {
		counts[issue.Number]++
	}
	return duplicateNumbers(counts)
}

func duplicatePullRequestNumbers(prs []model.PullRequest) []int {
	counts := map[int]int{}
	for _, pr := range prs {
		counts[pr.Number]++
	}
	return duplicateNumbers(counts)
}

func duplicateNumbers(counts map[int]int) []int {
	var duplicates []int
	for number, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, number)
		}
	}
	sort.Ints(duplicates)
	return duplicates
}

func normalizeImportArgs(args []string) ([]string, error) {
	var flags []string
	var repo string
	boolFlags := map[string]bool{
		"--plain-file-store": true,
		"-plain-file-store":  true,
		"--verbose":          true,
		"-verbose":           true,
		"--v":                true,
		"-v":                 true,
		"--local":            true,
		"-local":             true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if strings.Contains(arg, "=") {
				continue
			}
			if boolFlags[arg] {
				continue
			}
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags = append(flags, args[i])
			continue
		}
		if repo != "" {
			return nil, fmt.Errorf("unexpected extra argument %q", arg)
		}
		repo = arg
	}

	if repo == "" {
		return flags, nil
	}
	return append(flags, repo), nil
}

func normalizeSingleValueCommandArgs(args []string, valueName string, boolFlags map[string]bool) ([]string, error) {
	var flags []string
	var value string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if strings.Contains(arg, "=") || boolFlags[arg] {
				continue
			}
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags = append(flags, args[i])
			continue
		}
		if value != "" {
			return nil, fmt.Errorf("unexpected extra %s argument %q", valueName, arg)
		}
		value = arg
	}
	if value == "" {
		return flags, nil
	}
	return append(flags, value), nil
}
