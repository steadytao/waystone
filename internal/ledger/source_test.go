// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import "testing"

func TestParseSourceSpecAllowsWaystoneNamespace(t *testing.T) {
	source, err := ParseSourceSpec("waystone:example/project")
	if err != nil {
		t.Fatalf("ParseSourceSpec returned error: %v", err)
	}
	if source.System != "waystone" || source.Owner != "example" || source.Repo != "project" {
		t.Fatalf("source = %#v, want waystone:example/project", source)
	}
}
