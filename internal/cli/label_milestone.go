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

func runLabel(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printLabelUsage(stderr)
		return errors.New("missing label command")
	}
	switch args[0] {
	case "list":
		return runLabelList(args[1:], stdout)
	default:
		printLabelUsage(stderr)
		return fmt.Errorf("unknown label command %q", args[0])
	}
}

func runLabelList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone label list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var labels []model.Label
	if sourceSet {
		labels, err = reader.SourceLabels(source)
	} else {
		labels, err = reader.Labels()
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, labels)
	}
	if sourceSet {
		fmt.Fprintf(stdout, "%-24s %-8s %s\n", "NAME", "COLOR", "DESCRIPTION")
	} else {
		fmt.Fprintf(stdout, "%-28s %-24s %-8s %s\n", "SOURCE", "NAME", "COLOR", "DESCRIPTION")
	}
	for _, label := range labels {
		if sourceSet {
			fmt.Fprintf(stdout, "%-24s %-8s %s\n", label.Name, label.Color, label.Description)
		} else {
			fmt.Fprintf(stdout, "%-28s %-24s %-8s %s\n", ledger.SourceSpec(label.Source), label.Name, label.Color, label.Description)
		}
	}
	return nil
}

func runMilestone(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printMilestoneUsage(stderr)
		return errors.New("missing milestone command")
	}
	switch args[0] {
	case "list":
		return runMilestoneList(args[1:], stdout)
	default:
		printMilestoneUsage(stderr)
		return fmt.Errorf("unknown milestone command %q", args[0])
	}
}

func runMilestoneList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone milestone list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "filter by source, e.g. github:owner/repo")
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reader := ledger.Reader{Root: *root}
	source, sourceSet, err := resolveOptionalSource(reader, *sourceFlag)
	if err != nil {
		return err
	}
	var milestones []model.Milestone
	if sourceSet {
		milestones, err = reader.SourceMilestones(source)
	} else {
		milestones, err = reader.Milestones()
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, milestones)
	}
	if sourceSet {
		fmt.Fprintf(stdout, "%-8s %-8s %s\n", "NUMBER", "STATE", "TITLE")
	} else {
		fmt.Fprintf(stdout, "%-28s %-8s %-8s %s\n", "SOURCE", "NUMBER", "STATE", "TITLE")
	}
	for _, milestone := range milestones {
		if sourceSet {
			fmt.Fprintf(stdout, "#%-7d %-8s %s\n", milestone.Number, milestone.State, milestone.Title)
		} else {
			fmt.Fprintf(stdout, "%-28s #%-7d %-8s %s\n", ledger.SourceSpec(milestone.Source), milestone.Number, milestone.State, milestone.Title)
		}
	}
	return nil
}
