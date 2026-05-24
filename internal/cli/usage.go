// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"strings"
)

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone <command> [<args>]")
	fmt.Fprintln(w, "  waystone help <command> [subcommand]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Import and audit commands")
	fmt.Fprintln(w, "  waystone forgejo import [--out <dir>] [--api-base <url>] <owner/repo>")
	fmt.Fprintln(w, "  waystone gitea import [--out <dir>] [--api-base <url>] <owner/repo>")
	fmt.Fprintln(w, "  waystone github audit [--ledger <dir>] [--api-base <url>] <owner/repo>")
	fmt.Fprintln(w, "  waystone github import [--out <dir>] [--v] <owner/repo>")
	fmt.Fprintln(w, "  waystone github refresh [--out <dir>] [--v] <owner/repo>")
	fmt.Fprintln(w, "  waystone gitlab import [--out <dir>] [--api-base <url>] <group/project>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Browse commands")
	fmt.Fprintln(w, "  waystone issue list [--source <source>] [--state <state>]")
	fmt.Fprintln(w, "  waystone issue search [--source <source>] [--field <field>] <text>")
	fmt.Fprintln(w, "  waystone issue show [--source <source>] <number>")
	fmt.Fprintln(w, "  waystone issue comments [--source <source>] <number>")
	fmt.Fprintln(w, "  waystone issue timeline [--source <source>] <number>")
	fmt.Fprintln(w, "  waystone label list [--source <source>]")
	fmt.Fprintln(w, "  waystone milestone list [--source <source>]")
	fmt.Fprintln(w, "  waystone pr list [--source <source>]")
	fmt.Fprintln(w, "  waystone pr search [--source <source>] [--field <field>] <text>")
	fmt.Fprintln(w, "  waystone pr show [--source <source>] <number>")
	fmt.Fprintln(w, "  waystone pr comments [--source <source>] <number>")
	fmt.Fprintln(w, "  waystone pr timeline [--source <source>] <number>")
	fmt.Fprintln(w, "  waystone source list [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone source show [--ledger <dir>] <source>")
	fmt.Fprintln(w, "  waystone source status [--ledger <dir>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Local authoring commands")
	fmt.Fprintln(w, "  waystone issue create --source <owner/repo> --title <title> [--body <body> | --body-file <file>]")
	fmt.Fprintln(w, "  waystone issue edit --source <owner/repo> --issue <number> [--title <title>] [--body <body> | --body-file <file>]")
	fmt.Fprintln(w, "  waystone issue comment --source <owner/repo> --issue <number> [--body <body> | --body-file <file>]")
	fmt.Fprintln(w, "  waystone issue close --source <owner/repo> --issue <number>")
	fmt.Fprintln(w, "  waystone issue reopen --source <owner/repo> --issue <number>")
	fmt.Fprintln(w, "  waystone issue label add --source <owner/repo> --issue <number> <label>")
	fmt.Fprintln(w, "  waystone issue label remove --source <owner/repo> --issue <number> <label>")
	fmt.Fprintln(w, "  waystone label create --source <owner/repo> --slug <slug> --name <name> [--color <hex>] [--description <text>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Ledger commands")
	fmt.Fprintln(w, "  waystone ledger export [--ledger <dir>] [--source <source>] [--out <file>]")
	fmt.Fprintln(w, "  waystone ledger doctor [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone ledger diff --source <source> --since <operation>")
	fmt.Fprintln(w, "  waystone ledger import [--ledger <dir>] [--unsafe] <archive>")
	fmt.Fprintln(w, "  waystone ledger inspect <archive>")
	fmt.Fprintln(w, "  waystone ledger summary [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone ledger status [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone ledger history [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone ledger show-operation [--ledger <dir>] <operation-id>")
	fmt.Fprintln(w, "  waystone ledger verify [--ledger <dir>] [--strict] [--signatures]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Migration commands")
	fmt.Fprintln(w, "  waystone migrate inspect [--allow-unknown] <plan>")
	fmt.Fprintln(w, "  waystone migrate loss-report --from <source> [--from <source>] --to <source> --json")
	fmt.Fprintln(w, "  waystone migrate plan --from <source> [--from <source>] --to <source> --out <file> [--numbering-strategy <strategy>] [--strategy-file <file>]")
	fmt.Fprintln(w, "  waystone migrate report --from <source> [--from <source>] --to <source> [--numbering-strategy <strategy>] [--json]")
	fmt.Fprintln(w, "  waystone migrate verify <plan>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Identity and source management")
	fmt.Fprintln(w, "  waystone github auth login [--client-id <id>] [--plain-file-store]")
	fmt.Fprintln(w, "  waystone github auth logout [--plain-file-store]")
	fmt.Fprintln(w, "  waystone identity init [--ledger <dir>] [--name <name>]")
	fmt.Fprintln(w, "  waystone identity list [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone identity show [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone identity status [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone identity trust [--ledger <dir>] <identity-id>")
	fmt.Fprintln(w, "  waystone identity untrust [--ledger <dir>] <identity-id>")
	fmt.Fprintln(w, "  waystone source default [--ledger <dir>] [source]")
	fmt.Fprintln(w, "  waystone source inspect [--ledger <dir>] <source>")
	fmt.Fprintln(w, "  waystone source refresh [--ledger <dir>] [--source <source>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Other commands")
	fmt.Fprintln(w, "  waystone audit list [--ledger <dir>]")
	fmt.Fprintln(w, "  waystone audit show [--ledger <dir>] <audit>")
	fmt.Fprintln(w, "  waystone version [--json]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "See 'waystone help <command>' for more information on a command.")
}

func runHelp(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	key := strings.Join(args, " ")
	if help, ok := commandHelp[key]; ok {
		fmt.Fprint(stdout, help)
		return nil
	}
	if help, ok := commandHelp[args[0]]; ok {
		fmt.Fprint(stdout, help)
		return nil
	}
	return fmt.Errorf("unknown help topic %q", key)
}

var commandHelp = map[string]string{
	"audit": `Usage:
  waystone audit list [--ledger <dir>] [--source <source>] [--json]
  waystone audit show [--ledger <dir>] [--json] [--verbose | --v] <audit>

Show locally stored GitHub exit-readiness audit records.
`,
	"audit list": `Usage:
  waystone audit list [--ledger <dir>] [--source <source>] [--json]

List stored GitHub exit-readiness audits.

Options:
  --ledger <dir>      Waystone ledger directory, default .waystone
  --source <source>   source to inspect, e.g. github:owner/repo
  --json              write JSON output
`,
	"audit show": `Usage:
  waystone audit show [--ledger <dir>] [--json] [--verbose | --v] <audit>

Show one stored GitHub exit-readiness audit.

Options:
  --ledger <dir>      Waystone ledger directory, default .waystone
  --json              write JSON output
  --verbose, --v      show detailed audit evidence
`,
	"forgejo": `Usage:
  waystone forgejo import [--out <dir>] [--token-env <name>] [--api-base <url>] [--concurrency <n>] [--timeout <duration>] [--local] <owner/repo>

Import read-only Forgejo project history into a forgejo: source.

Options:
  --out <dir>          ledger output directory, default .waystone
  --token-env <name>   environment variable containing a Forgejo token
  --api-base <url>     Forgejo API base URL, default https://codeberg.org/api/v1
  --concurrency <n>    maximum concurrent detail requests
  --timeout <duration> request timeout
  --local              include local OS user and hostname in operation records
`,
	"forgejo import": `Usage:
  waystone forgejo import [--out <dir>] [--token-env <name>] [--api-base <url>] [--concurrency <n>] [--timeout <duration>] [--local] <owner/repo>

Import read-only Forgejo project history into a forgejo: source.

Options:
  --out <dir>          ledger output directory, default .waystone
  --token-env <name>   environment variable containing a Forgejo token
  --api-base <url>     Forgejo API base URL, default https://codeberg.org/api/v1
  --concurrency <n>    maximum concurrent detail requests
  --timeout <duration> request timeout
  --local              include local OS user and hostname in operation records
`,
	"gitea": `Usage:
  waystone gitea import [--out <dir>] [--token-env <name>] [--api-base <url>] [--concurrency <n>] [--timeout <duration>] [--local] <owner/repo>

Import read-only Gitea project history into a gitea: source.

Options:
  --out <dir>          ledger output directory, default .waystone
  --token-env <name>   environment variable containing a Gitea token
  --api-base <url>     Gitea API base URL, default https://gitea.com/api/v1
  --concurrency <n>    maximum concurrent detail requests
  --timeout <duration> request timeout
  --local              include local OS user and hostname in operation records
`,
	"gitea import": `Usage:
  waystone gitea import [--out <dir>] [--token-env <name>] [--api-base <url>] [--concurrency <n>] [--timeout <duration>] [--local] <owner/repo>

Import read-only Gitea project history into a gitea: source.

Options:
  --out <dir>          ledger output directory, default .waystone
  --token-env <name>   environment variable containing a Gitea token
  --api-base <url>     Gitea API base URL, default https://gitea.com/api/v1
  --concurrency <n>    maximum concurrent detail requests
  --timeout <duration> request timeout
  --local              include local OS user and hostname in operation records
`,
	"github": `Usage:
  waystone github auth login [--client-id <id>] [--scope <scope>] [--timeout <duration>] [--plain-file-store]
  waystone github auth logout [--plain-file-store]
  waystone github audit [--ledger <dir>] [--api-base <url>] [--json] [--verbose | --v] [--no-write] <owner/repo>
  waystone github import [--out <dir>] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--concurrency <n>] [--plain-file-store] [--verbose | --v] [--local] <owner/repo>
  waystone github refresh [--out <dir>] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--concurrency <n>] [--plain-file-store] [--verbose | --v] [--local] <owner/repo>

Authenticate with GitHub, audit migration surfaces and import GitHub project history.
`,
	"github auth": `Usage:
  waystone github auth login [--client-id <id>] [--scope <scope>] [--timeout <duration>] [--plain-file-store]
  waystone github auth logout [--plain-file-store]

Manage GitHub device-flow authentication.
`,
	"github auth login": `Usage:
  waystone github auth login [--client-id <id>] [--scope <scope>] [--timeout <duration>] [--plain-file-store]

Start GitHub OAuth device-flow authentication.

Options:
  --client-id <id>        GitHub OAuth app client ID
  --scope <scope>         GitHub OAuth scope
  --timeout <duration>    login timeout, default 15m
  --plain-file-store      store token in a plaintext local file instead of the OS credential store
`,
	"github auth logout": `Usage:
  waystone github auth logout [--plain-file-store]

Delete the stored GitHub token.

Options:
  --plain-file-store      delete token from the plaintext local file instead of the OS credential store
`,
	"github import": `Usage:
  waystone github import [--out <dir>] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--concurrency <n>] [--plain-file-store] [--verbose | --v] [--local] <owner/repo>
  waystone github refresh [--out <dir>] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--concurrency <n>] [--plain-file-store] [--verbose | --v] [--local] <owner/repo>

Import or refresh read-only GitHub project history.

Options:
  --out <dir>              ledger output directory, default .waystone
  --token-env <name>       environment variable containing a GitHub token
  --api-base <url>         GitHub API base URL
  --timeout <duration>     request timeout
  --concurrency <n>        maximum concurrent detail requests
  --plain-file-store       read stored token from a plaintext local file
  --verbose, --v           show detailed import progress
  --local                  include local OS user and hostname in operation records
`,
	"github refresh": `Usage:
  waystone github refresh [--out <dir>] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--concurrency <n>] [--plain-file-store] [--verbose | --v] [--local] <owner/repo>

Refresh an existing GitHub source.

Uses the same import pipeline as github import.

Options:
  --out <dir>              ledger output directory, default .waystone
  --token-env <name>       environment variable containing a GitHub token
  --api-base <url>         GitHub API base URL
  --timeout <duration>     request timeout
  --concurrency <n>        maximum concurrent detail requests
  --plain-file-store       read stored token from a plaintext local file
  --verbose, --v           show detailed import progress
  --local                  include local OS user and hostname in operation records
`,
	"github audit": `Usage:
  waystone github audit [--ledger <dir>] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--plain-file-store] [--json] [--verbose | --v] [--no-write] [--local] <owner/repo>

Audit GitHub migration surfaces without mutating the remote repository.

Options:
  --ledger <dir>           Waystone ledger directory
  --token-env <name>       environment variable containing a GitHub token
  --api-base <url>         GitHub API base URL
  --timeout <duration>     request timeout
  --plain-file-store       read stored token from a plaintext local file
  --json                   write JSON output
  --verbose, --v           show detailed audit evidence
  --no-write               print the audit without writing it to the ledger
  --local                  include local OS user and hostname in operation records
`,
	"gitlab": `Usage:
  waystone gitlab import [--out <dir>] [--token-env <name>] [--api-base <url>] [--concurrency <n>] [--timeout <duration>] [--local] <group/project>

Import read-only GitLab project history into a gitlab: source.

Options:
  --out <dir>          ledger output directory, default .waystone
  --token-env <name>   environment variable containing a GitLab token
  --api-base <url>     GitLab API base URL
  --concurrency <n>    maximum concurrent GitLab detail requests
  --timeout <duration> request timeout
  --local              include local OS user and hostname in operation records
`,
	"gitlab import": `Usage:
  waystone gitlab import [--out <dir>] [--token-env <name>] [--api-base <url>] [--concurrency <n>] [--timeout <duration>] [--local] <group/project>

Import read-only GitLab project history into a gitlab: source.

Options:
  --out <dir>          ledger output directory, default .waystone
  --token-env <name>   environment variable containing a GitLab token
  --api-base <url>     GitLab API base URL
  --concurrency <n>    maximum concurrent GitLab detail requests
  --timeout <duration> request timeout
  --local              include local OS user and hostname in operation records
`,
	"identity": `Usage:
  waystone identity init [--ledger <dir>] [--name <name>]
  waystone identity list [--ledger <dir>] [--json]
  waystone identity show [--ledger <dir>] [--json]
  waystone identity status [--ledger <dir>] [--json]
  waystone identity trust [--ledger <dir>] <identity-id>
  waystone identity untrust [--ledger <dir>] <identity-id>

Manage local Waystone signing identities and trust policy.
`,
	"identity init": `Usage:
  waystone identity init [--ledger <dir>] [--name <name>]

Create a local signing identity for operation and source signatures.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --name <name>        identity display name
`,
	"identity list": `Usage:
  waystone identity list [--ledger <dir>] [--json]

List local signing identities and trust state.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --json               write JSON output
`,
	"identity show": `Usage:
  waystone identity show [--ledger <dir>] [--json]

Show the default local signing identity.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --json               write JSON output
`,
	"identity status": `Usage:
  waystone identity status [--ledger <dir>] [--json]

Show identity trust counts.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --json               write JSON output
`,
	"identity trust": `Usage:
  waystone identity trust [--ledger <dir>] <identity-id>

Mark an identity as trusted for signature verification.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
`,
	"identity untrust": `Usage:
  waystone identity untrust [--ledger <dir>] <identity-id>

Remove an identity from the local trust policy.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
`,
	"issue": `Usage:
  waystone issue create --source <owner/repo> --title <title> [--body <body> | --body-file <file>]
  waystone issue edit --source <owner/repo> --issue <number> [--title <title>] [--body <body> | --body-file <file>]
  waystone issue comment --source <owner/repo> --issue <number> [--body <body> | --body-file <file>]
  waystone issue close --source <owner/repo> --issue <number>
  waystone issue reopen --source <owner/repo> --issue <number>
  waystone issue label add --source <owner/repo> --issue <number> <label>
  waystone issue label remove --source <owner/repo> --issue <number> <label>
  waystone issue list [--source <source>] [--state <state>]
  waystone issue search [--source <source>] [--field <field>] <text>
  waystone issue show [--source <source>] [--with-comments] [--json] <number>
  waystone issue comments [--source <source>] [--json] <number>
  waystone issue timeline [--source <source>] [--json] <number>

Browse imported issues and write local issue history under waystone: sources.
`,
	"issue create": `Usage:
  waystone issue create [--ledger <dir>] --source <owner/repo> --title <title> [--body <body> | --body-file <file>] [--local]

Create a local issue under a waystone: source.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --title <title>         issue title
  --body <body>           issue body
  --body-file <file>      file containing issue body
  --local                 include local OS user and hostname in operation records
`,
	"issue edit": `Usage:
  waystone issue edit [--ledger <dir>] --source <owner/repo> --issue <number> [--title <title>] [--body <body> | --body-file <file>] [--local]

Edit a local issue under a waystone: source.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --issue <number>        issue number
  --title <title>         replacement issue title
  --body <body>           replacement issue body
  --body-file <file>      file containing replacement issue body
  --local                 include local OS user and hostname in operation records
`,
	"issue comment": `Usage:
  waystone issue comment [--ledger <dir>] --source <owner/repo> --issue <number> [--body <body> | --body-file <file>] [--local]

Add a local issue comment.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --issue <number>        issue number
  --body <body>           comment body
  --body-file <file>      file containing comment body
  --local                 include local OS user and hostname in operation records
`,
	"issue close": `Usage:
  waystone issue close --source <owner/repo> --issue <number> [--local]

Close a local issue.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --issue <number>        issue number
  --local                 include local OS user and hostname in operation records
`,
	"issue reopen": `Usage:
  waystone issue reopen --source <owner/repo> --issue <number> [--local]

Reopen a local issue.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --issue <number>        issue number
  --local                 include local OS user and hostname in operation records
`,
	"issue label": `Usage:
  waystone issue label add --source <owner/repo> --issue <number> <label>
  waystone issue label remove --source <owner/repo> --issue <number> <label>

Apply or remove local labels on local issues.
`,
	"issue label add": `Usage:
  waystone issue label add --source <owner/repo> --issue <number> <label> [--local]

Apply a local label to a local issue.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --issue <number>        issue number
  --local                 include local OS user and hostname in operation records
`,
	"issue label remove": `Usage:
  waystone issue label remove --source <owner/repo> --issue <number> <label> [--local]

Remove a local label from a local issue.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --issue <number>        issue number
  --local                 include local OS user and hostname in operation records
`,
	"issue list": `Usage:
  waystone issue list [--ledger <dir>] [--source <source>] [--state <state>]

List issues from the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --state <state>      issue state: open, closed or all
`,
	"issue search": `Usage:
  waystone issue search [--ledger <dir>] [--source <source>] [--state <state>] [--field <field>] <text>

Search issues from the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --state <state>      issue state: open, closed or all
  --field <field>      field to search: title, description, author, state, label, milestone, url or all
  --json               write JSON output
`,
	"issue show": `Usage:
  waystone issue show [--ledger <dir>] [--source <source>] [--with-comments] [--json] <number>

Show one issue from the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --with-comments      include issue comments
  --json               write JSON output
`,
	"issue comments": `Usage:
  waystone issue comments [--ledger <dir>] [--source <source>] [--json] <number>

Show issue comments.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --json               write JSON output
`,
	"issue timeline": `Usage:
  waystone issue timeline [--ledger <dir>] [--source <source>] [--json] <number>

Show an issue timeline.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --json               write JSON output
`,
	"label": `Usage:
  waystone label list [--ledger <dir>] [--source <source>] [--json]
  waystone label create --source <owner/repo> --slug <slug> --name <name> [--color <hex>] [--description <text>] [--local]

Browse imported labels and create local labels under waystone: sources.
`,
	"label list": `Usage:
  waystone label list [--ledger <dir>] [--source <source>] [--json]

List labels from the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --json               write JSON output
`,
	"label create": `Usage:
  waystone label create --source <owner/repo> --slug <slug> --name <name> [--color <hex>] [--description <text>] [--local]

Create a local label under a waystone: source.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <owner/repo>   local source; bare owner/repo means waystone:owner/repo
  --slug <slug>           stable label slug
  --name <name>           label display name
  --color <hex>           six-character label colour
  --description <text>    label description
  --local                 include local OS user and hostname in operation records
`,
	"ledger": `Usage:
  waystone ledger doctor [--ledger <dir>] [--stale-after <duration>] [--json]
  waystone ledger diff --source <source> --since <operation> [--include-verified] [--json]
  waystone ledger export [--ledger <dir>] [--source <source>] [--out <file>] [--format archive|json] [--compact]
  waystone ledger import [--ledger <dir>] [--unsafe] <archive>
  waystone ledger inspect [--json] <archive>
  waystone ledger summary [--ledger <dir>] [--json]
  waystone ledger status [--ledger <dir>] [--json]
  waystone ledger history [--ledger <dir>] [--json]
  waystone ledger show-operation [--ledger <dir>] [--json] <operation-id>
  waystone ledger verify [--ledger <dir>] [--strict] [--signatures] [--json]

Inspect, verify, export and import Waystone ledgers.
`,
	"ledger doctor": `Usage:
  waystone ledger doctor [--ledger <dir>] [--stale-after <duration>] [--json]

Find practical ledger issues.

Options:
  --ledger <dir>              Waystone ledger directory, default .waystone
  --stale-after <duration>    warn when a source has not been refreshed for this duration, or 0 to disable
  --json                      write JSON output
`,
	"ledger diff": `Usage:
  waystone ledger diff --source <source> --since <operation> [--include-verified] [--json]

Show source changes since an operation.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <source>       source to diff, e.g. github:owner/repo
  --since <operation>     operation ID to diff after
  --include-verified      include verification-only changes
  --json                  write JSON output
`,
	"ledger export": `Usage:
  waystone ledger export [--ledger <dir>] [--source <source>] [--out <file>] [--format <format>] [--compact]

Export a ledger archive or JSON snapshot.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    export only one source, e.g. github:owner/repo
  --out <file>         export path, default waystone-ledger
  --format <format>    export format: archive or json
  --compact            write compact JSON when --format=json
`,
	"ledger import": `Usage:
  waystone ledger import [--ledger <dir>] [--unsafe] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--plain-file-store] <archive>

Import a Waystone ledger archive.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --unsafe                skip remote source confirmation before importing
  --token-env <name>      environment variable containing a GitHub token
  --api-base <url>        GitHub API base URL
  --timeout <duration>    request timeout
  --plain-file-store      read stored token from a plaintext local file
`,
	"ledger inspect": `Usage:
  waystone ledger inspect [--json] <archive>

Inspect a Waystone ledger archive.

Options:
  --json                  write JSON output
`,
	"ledger summary": `Usage:
  waystone ledger summary [--ledger <dir>] [--json]

Summarise ledger record counts.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --json                  write JSON output
`,
	"ledger status": `Usage:
  waystone ledger status [--ledger <dir>] [--json]

Show ledger health and latest operation state.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --json                  write JSON output
`,
	"ledger history": `Usage:
  waystone ledger history [--ledger <dir>] [--json]

List ledger operation records.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --json                  write JSON output
`,
	"ledger show-operation": `Usage:
  waystone ledger show-operation [--ledger <dir>] [--json] <operation-id>

Show one operation record.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --json                  write JSON output
`,
	"ledger verify": `Usage:
  waystone ledger verify [--ledger <dir>] [--strict | --operations] [--signatures] [--json] [--local]

Verify ledger files, hashes and optional signatures.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --strict                strictly verify operation chain and recorded file hashes
  --operations            alias for --strict
  --signatures            verify operation record signatures
  --json                  write JSON output
  --local                 include local OS user and hostname in operation records
`,
	"migrate": `Usage:
  waystone migrate inspect [--allow-unknown] <plan>
  waystone migrate loss-report --from <source> [--from <source>] --to <source> --json
  waystone migrate plan --from <source> [--from <source>] --to <source> --out <file> [--numbering-strategy <strategy>] [--strategy-file <file>]
  waystone migrate report --from <source> [--from <source>] --to <source> [--numbering-strategy <strategy>] [--json]
  waystone migrate verify <plan>

Report migration shape and write read-only migration plans.
`,
	"migrate inspect": `Usage:
  waystone migrate inspect [--allow-unknown] <plan>

Inspect a saved migration plan.

Options:
  --allow-unknown     inspect an unknown plan version without treating it as an error
`,
	"migrate loss-report": `Usage:
  waystone migrate loss-report [--ledger <dir>] --from <source> [--from <source>] --to <source> --json

Write a structured migration loss report from one or more source namespaces.

Options:
  --ledger <dir>      Waystone ledger directory
  --from <source>     source to report from; repeatable or comma-separated
  --to <source>       target source to report to
  --json              write JSON output
`,
	"migrate report": `Usage:
  waystone migrate report [--ledger <dir>] --from <source> [--from <source>] --to <source> [--numbering-strategy <strategy>] [--json]

Generate a read-only migration report from one or more source namespaces.

Options:
  --ledger <dir>                    Waystone ledger directory
  --from <source>                   source to report from; repeatable or comma-separated
  --to <source>                     target source to report to
  --numbering-strategy <strategy>   numbering strategy, currently preserve-source-numbering
  --json                            write JSON output
`,
	"migrate plan": `Usage:
  waystone migrate plan [--ledger <dir>] --from <source> [--from <source>] --to <source> --out <file> [--numbering-strategy <strategy>] [--strategy-file <file>]

Write a read-only JSON migration plan.

Options:
  --ledger <dir>                    Waystone ledger directory
  --from <source>                   source to plan from; repeatable or comma-separated
  --to <source>                     target source to plan to
  --out <file>                      migration plan output path
  --numbering-strategy <strategy>   numbering strategy, currently preserve-source-numbering
  --strategy-file <file>            migration strategy JSON file accepting only safe read-only defaults
`,
	"migrate verify": `Usage:
  waystone migrate verify <plan>

Verify a saved migration plan artefact.
`,
	"milestone": `Usage:
  waystone milestone list [--ledger <dir>] [--source <source>] [--json]

Browse imported milestones.
`,
	"milestone list": `Usage:
  waystone milestone list [--ledger <dir>] [--source <source>] [--json]

List milestones from the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --json               write JSON output
`,
	"pr": `Usage:
  waystone pr list [--source <source>]
  waystone pr search [--source <source>] [--field <field>] <text>
  waystone pr show [--source <source>] [--with-comments] [--json] <number>
  waystone pr comments [--source <source>] [--json] <number>
  waystone pr timeline [--source <source>] [--json] <number>

Browse imported pull requests, merge requests and review records.
`,
	"pr list": `Usage:
  waystone pr list [--ledger <dir>] [--source <source>]

List pull requests and merge requests from the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
`,
	"pr search": `Usage:
  waystone pr search [--ledger <dir>] [--source <source>] [--field <field>] <text>

Search pull requests and merge requests from the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --field <field>      field to search: title, description, author, state, branch, url or all
  --json               write JSON output
`,
	"pr show": `Usage:
  waystone pr show [--ledger <dir>] [--source <source>] [--with-comments] [--json] <number>

Show one pull request or merge request.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --with-comments      include review comments
  --json               write JSON output
`,
	"pr comments": `Usage:
  waystone pr comments [--ledger <dir>] [--source <source>] [--json] <number>

Show pull request or merge request comments.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --json               write JSON output
`,
	"pr timeline": `Usage:
  waystone pr timeline [--ledger <dir>] [--source <source>] [--json] <number>

Show a pull request or merge request timeline.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --source <source>    filter by source, e.g. github:owner/repo
  --json               write JSON output
`,
	"source": `Usage:
  waystone source default [--ledger <dir>] [--clear] [source]
  waystone source inspect [--ledger <dir>] [--stale-after <duration>] [--json] <source>
  waystone source list [--ledger <dir>] [--json]
  waystone source refresh [--ledger <dir>] [--source <source>] [--sources <sources>] [--v]
  waystone source show [--ledger <dir>] [--json] <source>
  waystone source status [--ledger <dir>] [--stale-after <duration>] [--json]

Manage and inspect ledger source manifests.
`,
	"source default": `Usage:
  waystone source default [--ledger <dir>] [--clear] [--local] [source]

Show, set or clear the default source.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --clear              clear the default source
  --local              include local OS user and hostname in operation records
`,
	"source inspect": `Usage:
  waystone source inspect [--ledger <dir>] [--stale-after <duration>] [--json] <source>

Inspect one source manifest and referenced objects.

Options:
  --ledger <dir>              Waystone ledger directory, default .waystone
  --stale-after <duration>    mark source stale after this duration, or 0 to disable
  --json                      write JSON output
`,
	"source list": `Usage:
  waystone source list [--ledger <dir>] [--json]

List source manifests in the ledger.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --json               write JSON output
`,
	"source refresh": `Usage:
  waystone source refresh [--ledger <dir>] [--source <source>] [--sources <sources>] [--token-env <name>] [--api-base <url>] [--timeout <duration>] [--concurrency <n>] [--plain-file-store] [--verbose | --v] [--local]

Refresh source manifests from their remote forge.

Options:
  --ledger <dir>          Waystone ledger directory, default .waystone
  --source <source>       source to refresh; repeatable or comma-separated
  --sources <sources>     sources to refresh as a comma-separated list
  --token-env <name>      environment variable containing a GitHub token
  --api-base <url>        GitHub API base URL
  --timeout <duration>    request timeout
  --concurrency <n>       maximum concurrent GitHub detail requests
  --plain-file-store      read stored token from a plaintext local file
  --verbose, --v          show detailed import progress
  --local                 include local OS user and hostname in operation records
`,
	"source show": `Usage:
  waystone source show [--ledger <dir>] [--json] <source>

Show one source manifest summary.

Options:
  --ledger <dir>       Waystone ledger directory, default .waystone
  --json               write JSON output
`,
	"source status": `Usage:
  waystone source status [--ledger <dir>] [--stale-after <duration>] [--json]

Show refresh age and stale status for each source.

Options:
  --ledger <dir>              Waystone ledger directory, default .waystone
  --stale-after <duration>    mark sources stale after this duration, or 0 to disable
  --json                      write JSON output
`,
	"version": `Usage:
  waystone version [--json]

Print the Waystone version.
`,
}

func printAuditUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone audit list [--ledger <dir>] [--source <source>] [--json]")
	fmt.Fprintln(w, "  waystone audit show [--ledger <dir>] [--json] [--verbose | --v] <audit>")
}

func printGitHubUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone github auth login [--client-id <id>] [--scope <scope>] [--timeout <duration>] [--plain-file-store]")
	fmt.Fprintln(w, "  waystone github auth logout [--plain-file-store]")
	fmt.Fprintln(w, "  waystone github audit [--ledger <dir>] [--api-base <url>] <owner/repo>")
	fmt.Fprintln(w, "  waystone github import [--out <dir>] [--v] <owner/repo>")
	fmt.Fprintln(w, "  waystone github refresh [--out <dir>] [--v] <owner/repo>")
}

func printForgejoUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone forgejo import [--out <dir>] [--api-base <url>] <owner/repo>")
}

func printGiteaUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone gitea import [--out <dir>] [--api-base <url>] <owner/repo>")
}

func printGitHubAuthUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone github auth login [--client-id <id>] [--scope <scope>] [--timeout <duration>] [--plain-file-store]")
	fmt.Fprintln(w, "  waystone github auth logout [--plain-file-store]")
}

func printGitLabUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone gitlab import [--out <dir>] [--api-base <url>] <group/project>")
}

func printIssueUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone issue create --source <owner/repo> --title <title> [--body <body> | --body-file <file>]")
	fmt.Fprintln(w, "  waystone issue edit --source <owner/repo> --issue <number> [--title <title>] [--body <body> | --body-file <file>]")
	fmt.Fprintln(w, "  waystone issue comment --source <owner/repo> --issue <number> [--body <body> | --body-file <file>]")
	fmt.Fprintln(w, "  waystone issue close --source <owner/repo> --issue <number>")
	fmt.Fprintln(w, "  waystone issue reopen --source <owner/repo> --issue <number>")
	fmt.Fprintln(w, "  waystone issue label add --source <owner/repo> --issue <number> <label>")
	fmt.Fprintln(w, "  waystone issue label remove --source <owner/repo> --issue <number> <label>")
	fmt.Fprintln(w, "  waystone issue list [--source <source>] [--state open|closed|all]")
	fmt.Fprintln(w, "  waystone issue search [--source <source>] [--state open|closed|all] [--field <field>] <text>")
	fmt.Fprintln(w, "  waystone issue show [--source <source>] [--with-comments] [--json] <number>")
	fmt.Fprintln(w, "  waystone issue comments [--source <source>] [--json] <number>")
	fmt.Fprintln(w, "  waystone issue timeline [--source <source>] [--json] <number>")
}

func printIdentityUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone identity init [--ledger <dir>] [--name <name>]")
	fmt.Fprintln(w, "  waystone identity list [--ledger <dir>] [--json]")
	fmt.Fprintln(w, "  waystone identity show [--ledger <dir>] [--json]")
	fmt.Fprintln(w, "  waystone identity status [--ledger <dir>] [--json]")
	fmt.Fprintln(w, "  waystone identity trust [--ledger <dir>] <identity-id>")
	fmt.Fprintln(w, "  waystone identity untrust [--ledger <dir>] <identity-id>")
}

func printLabelUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone label list [--ledger <dir>] [--source <source>] [--json]")
	fmt.Fprintln(w, "  waystone label create --source <owner/repo> --slug <slug> --name <name> [--color <hex>] [--description <text>]")
}

func printMilestoneUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone milestone list [--ledger <dir>] [--source <source>] [--json]")
}

func printMigrateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone migrate inspect [--allow-unknown] <plan>")
	fmt.Fprintln(w, "  waystone migrate loss-report --from <source> [--from <source>] --to <source> --json")
	fmt.Fprintln(w, "  waystone migrate plan --from <source> [--from <source>] --to <source> --out <file> [--numbering-strategy <strategy>] [--strategy-file <file>]")
	fmt.Fprintln(w, "  waystone migrate report --from <source> [--from <source>] --to <source> [--numbering-strategy <strategy>] [--json]")
	fmt.Fprintln(w, "  waystone migrate verify <plan>")
}

func printLedgerUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone ledger doctor [--ledger <dir>] [--stale-after <duration>] [--json]")
	fmt.Fprintln(w, "  waystone ledger diff --source <source> --since <operation>")
	fmt.Fprintln(w, "  waystone ledger export [--ledger <dir>] [--source <source>] [--out <file>] [--format archive|json]")
	fmt.Fprintln(w, "  waystone ledger import [--ledger <dir>] [--unsafe] <archive>")
	fmt.Fprintln(w, "  waystone ledger inspect [--json] <archive>")
	fmt.Fprintln(w, "  waystone ledger summary [--ledger <dir>] [--json]")
	fmt.Fprintln(w, "  waystone ledger status [--ledger <dir>] [--json]")
	fmt.Fprintln(w, "  waystone ledger history [--ledger <dir>] [--json]")
	fmt.Fprintln(w, "  waystone ledger show-operation [--ledger <dir>] [--json] <operation-id>")
	fmt.Fprintln(w, "  waystone ledger verify [--ledger <dir>] [--strict] [--signatures] [--json]")
}

func printSourceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone source default [--ledger <dir>] [--clear] [source]")
	fmt.Fprintln(w, "  waystone source inspect [--ledger <dir>] [--stale-after <duration>] [--json] <source>")
	fmt.Fprintln(w, "  waystone source list [--ledger <dir>] [--json]")
	fmt.Fprintln(w, "  waystone source refresh [--ledger <dir>] [--source <source>] [--sources <sources>] [--v]")
	fmt.Fprintln(w, "  waystone source show [--ledger <dir>] [--json] <source>")
	fmt.Fprintln(w, "  waystone source status [--ledger <dir>] [--stale-after <duration>] [--json]")
}

func printPullRequestUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  waystone pr list [--source <source>]")
	fmt.Fprintln(w, "  waystone pr search [--source <source>] [--field <field>] <text>")
	fmt.Fprintln(w, "  waystone pr show [--source <source>] [--with-comments] [--json] <number>")
	fmt.Fprintln(w, "  waystone pr comments [--source <source>] [--json] <number>")
	fmt.Fprintln(w, "  waystone pr timeline [--source <source>] [--json] <number>")
}
