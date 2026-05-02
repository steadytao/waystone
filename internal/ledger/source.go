// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"fmt"
	"strings"

	"github.com/steadytao/waystone/internal/model"
)

func ParseSourceSpec(spec string) (model.Source, error) {
	system, rest, ok := strings.Cut(spec, ":")
	if !ok {
		parts := strings.Split(spec, "/")
		if len(parts) != 3 {
			return model.Source{}, fmt.Errorf("source must be system:owner/repo or system/owner/repo, got %q", spec)
		}
		return model.Source{System: parts[0], Owner: parts[1], Repo: parts[2]}, nil
	}
	owner, repo, ok := strings.Cut(rest, "/")
	if !ok || system == "" || owner == "" || repo == "" {
		return model.Source{}, fmt.Errorf("source must be system:owner/repo or system/owner/repo, got %q", spec)
	}
	return model.Source{System: system, Owner: owner, Repo: repo}, nil
}

func SourceSpec(source model.Source) string {
	return source.System + ":" + source.Owner + "/" + source.Repo
}
