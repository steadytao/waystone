// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/steadytao/waystone/internal/model"
)

var sourceComponentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ParseSourceSpec(spec string) (model.Source, error) {
	system, rest, ok := strings.Cut(spec, ":")
	if !ok {
		parts := strings.Split(spec, "/")
		if len(parts) != 3 {
			return model.Source{}, fmt.Errorf("source must be system:owner/repo or system/owner/repo, got %q", spec)
		}
		return newSource(parts[0], parts[1], parts[2], spec)
	}
	owner, repo, ok := strings.Cut(rest, "/")
	if !ok || system == "" || owner == "" || repo == "" {
		return model.Source{}, fmt.Errorf("source must be system:owner/repo or system/owner/repo, got %q", spec)
	}
	return newSource(system, owner, repo, spec)
}

func SourceSpec(source model.Source) string {
	return source.System + ":" + source.Owner + "/" + source.Repo
}

func newSource(system, owner, repo, spec string) (model.Source, error) {
	if !validSourceComponent(system) || !validSourceComponent(owner) || !validSourceComponent(repo) {
		return model.Source{}, fmt.Errorf("source contains unsafe component, got %q", spec)
	}
	return model.Source{System: system, Owner: owner, Repo: repo}, nil
}

func validSourceComponent(value string) bool {
	return sourceComponentPattern.MatchString(value) && value != "." && value != ".."
}
