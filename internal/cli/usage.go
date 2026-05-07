// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
)

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone audit list [flags]")
	fmt.Fprintln(w, "  waystone audit show [flags] <audit>")
	fmt.Fprintln(w, "  waystone github auth login [flags]")
	fmt.Fprintln(w, "  waystone github auth logout [flags]")
	fmt.Fprintln(w, "  waystone github audit [flags] owner/repo")
	fmt.Fprintln(w, "  waystone github import [flags] owner/repo")
	fmt.Fprintln(w, "  waystone github refresh [flags] owner/repo")
	fmt.Fprintln(w, "  waystone identity init [flags]")
	fmt.Fprintln(w, "  waystone identity list [flags]")
	fmt.Fprintln(w, "  waystone identity show [flags]")
	fmt.Fprintln(w, "  waystone identity status [flags]")
	fmt.Fprintln(w, "  waystone identity trust [flags] <identity-id>")
	fmt.Fprintln(w, "  waystone identity untrust [flags] <identity-id>")
	fmt.Fprintln(w, "  waystone issue list [flags]")
	fmt.Fprintln(w, "  waystone issue search [flags] <text>")
	fmt.Fprintln(w, "  waystone issue show [flags] <number>")
	fmt.Fprintln(w, "  waystone issue comments [flags] <number>")
	fmt.Fprintln(w, "  waystone issue timeline [flags] <number>")
	fmt.Fprintln(w, "  waystone label list [flags]")
	fmt.Fprintln(w, "  waystone milestone list [flags]")
	fmt.Fprintln(w, "  waystone ledger export [flags]")
	fmt.Fprintln(w, "  waystone ledger doctor [flags]")
	fmt.Fprintln(w, "  waystone ledger diff --source <source> --since <operation>")
	fmt.Fprintln(w, "  waystone ledger import [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger inspect [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger summary [flags]")
	fmt.Fprintln(w, "  waystone ledger status [flags]")
	fmt.Fprintln(w, "  waystone ledger history [flags]")
	fmt.Fprintln(w, "  waystone ledger show-operation [flags] <operation-id>")
	fmt.Fprintln(w, "  waystone ledger verify [flags]")
	fmt.Fprintln(w, "  waystone pr list [flags]")
	fmt.Fprintln(w, "  waystone pr search [flags] <text>")
	fmt.Fprintln(w, "  waystone pr show [flags] <number>")
	fmt.Fprintln(w, "  waystone pr comments [flags] <number>")
	fmt.Fprintln(w, "  waystone pr timeline [flags] <number>")
	fmt.Fprintln(w, "  waystone source default [flags] [source]")
	fmt.Fprintln(w, "  waystone source inspect [flags] <source>")
	fmt.Fprintln(w, "  waystone source list [flags]")
	fmt.Fprintln(w, "  waystone source refresh [flags]")
	fmt.Fprintln(w, "  waystone source show [flags] <source>")
	fmt.Fprintln(w, "  waystone source status [flags]")
	fmt.Fprintln(w, "  waystone version [flags]")
}

func printAuditUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone audit list [flags]")
	fmt.Fprintln(w, "  waystone audit show [flags] <audit>")
}

func printGitHubUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone github auth login [flags]")
	fmt.Fprintln(w, "  waystone github auth logout [flags]")
	fmt.Fprintln(w, "  waystone github audit [flags] owner/repo")
	fmt.Fprintln(w, "  waystone github import [flags] owner/repo")
	fmt.Fprintln(w, "  waystone github refresh [flags] owner/repo")
}

func printGitHubAuthUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone github auth login [flags]")
	fmt.Fprintln(w, "  waystone github auth logout [flags]")
}

func printIssueUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone issue create --source owner/repo --title <title> [flags]")
	fmt.Fprintln(w, "  waystone issue edit --source owner/repo --issue <number> [flags]")
	fmt.Fprintln(w, "  waystone issue comment --source owner/repo --issue <number> [flags]")
	fmt.Fprintln(w, "  waystone issue close --source owner/repo --issue <number> [flags]")
	fmt.Fprintln(w, "  waystone issue reopen --source owner/repo --issue <number> [flags]")
	fmt.Fprintln(w, "  waystone issue list [flags]")
	fmt.Fprintln(w, "  waystone issue search [flags] <text>")
	fmt.Fprintln(w, "  waystone issue show [flags] <number>")
	fmt.Fprintln(w, "  waystone issue comments [flags] <number>")
	fmt.Fprintln(w, "  waystone issue timeline [flags] <number>")
}

func printIdentityUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone identity init [flags]")
	fmt.Fprintln(w, "  waystone identity list [flags]")
	fmt.Fprintln(w, "  waystone identity show [flags]")
	fmt.Fprintln(w, "  waystone identity status [flags]")
	fmt.Fprintln(w, "  waystone identity trust [flags] <identity-id>")
	fmt.Fprintln(w, "  waystone identity untrust [flags] <identity-id>")
}

func printLabelUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone label list [flags]")
}

func printMilestoneUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone milestone list [flags]")
}

func printLedgerUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone ledger doctor [flags]")
	fmt.Fprintln(w, "  waystone ledger diff --source <source> --since <operation>")
	fmt.Fprintln(w, "  waystone ledger export [flags]")
	fmt.Fprintln(w, "  waystone ledger import [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger inspect [flags] <archive>")
	fmt.Fprintln(w, "  waystone ledger summary [flags]")
	fmt.Fprintln(w, "  waystone ledger status [flags]")
	fmt.Fprintln(w, "  waystone ledger history [flags]")
	fmt.Fprintln(w, "  waystone ledger show-operation [flags] <operation-id>")
	fmt.Fprintln(w, "  waystone ledger verify [flags]")
}

func printSourceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone source default [flags] [source]")
	fmt.Fprintln(w, "  waystone source inspect [flags] <source>")
	fmt.Fprintln(w, "  waystone source list [flags]")
	fmt.Fprintln(w, "  waystone source refresh [flags]")
	fmt.Fprintln(w, "  waystone source show [flags] <source>")
	fmt.Fprintln(w, "  waystone source status [flags]")
}

func printPullRequestUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone pr list [flags]")
	fmt.Fprintln(w, "  waystone pr search [flags] <text>")
	fmt.Fprintln(w, "  waystone pr show [flags] <number>")
	fmt.Fprintln(w, "  waystone pr comments [flags] <number>")
	fmt.Fprintln(w, "  waystone pr timeline [flags] <number>")
}
