// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

func runAudit(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printAuditUsage(stderr)
		return errors.New("missing audit command")
	}
	switch args[0] {
	case "list":
		return runAuditList(args[1:], stdout)
	case "show":
		return runAuditShow(args[1:], stdout)
	default:
		printAuditUsage(stderr)
		return fmt.Errorf("unknown audit command %q", args[0])
	}
}

func runAuditList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone audit list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "source to inspect, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone audit list [flags]")
	}
	reader := ledger.Reader{Root: *root}
	var audits []model.GitHubAudit
	var err error
	if *sourceFlag != "" {
		source, parseErr := ledger.ParseSourceSpec(*sourceFlag)
		if parseErr != nil {
			return parseErr
		}
		audits, err = reader.SourceAudits(source)
	} else {
		audits, err = reader.Audits()
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, audits)
	}
	writeField(stdout, "Audits", len(audits))
	if len(audits) == 0 {
		return nil
	}
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%-32s %-24s %-14s %s\n", "ID", "SOURCE", "GENERATED", "WORKFLOWS")
	for _, audit := range audits {
		fmt.Fprintf(stdout, "%-32s %-24s %-14s %d\n", audit.ID, ledger.SourceSpec(audit.Source), audit.GeneratedAt.Format("2006-01-02"), len(audit.Workflows))
	}
	return nil
}

func runAuditShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone audit show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	verbose := fs.Bool("verbose", false, "show detailed audit evidence")
	verboseShort := fs.Bool("v", false, "show detailed audit evidence")
	normalizedArgs, err := normalizeSingleValueCommandArgs(args, "audit", map[string]bool{
		"--json":    true,
		"-json":     true,
		"--verbose": true,
		"-verbose":  true,
		"--v":       true,
		"-v":        true,
	})
	if err != nil {
		return err
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: waystone audit show [flags] <audit>")
	}
	audit, err := (ledger.Reader{Root: *root}).Audit(fs.Arg(0))
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, audit)
	}
	writeGitHubAudit(stdout, audit, *verbose || *verboseShort)
	return nil
}
