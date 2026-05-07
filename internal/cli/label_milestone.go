// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/ledger"
	"github.com/steadytao/waystone/internal/model"
)

var labelColorPattern = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
var labelSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func runLabel(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printLabelUsage(stderr)
		return errors.New("missing label command")
	}
	switch args[0] {
	case "list":
		return runLabelList(args[1:], stdout)
	case "create":
		return runLabelCreate(args[1:], stdout)
	default:
		printLabelUsage(stderr)
		return fmt.Errorf("unknown label command %q", args[0])
	}
}

func runLabelCreate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone label create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("ledger", ".waystone", "Waystone ledger directory")
	sourceFlag := fs.String("source", "", "source for the local label, e.g. owner/repo or waystone:owner/repo")
	slugFlag := fs.String("slug", "", "stable label slug")
	nameFlag := fs.String("name", "", "label display name")
	colorFlag := fs.String("color", "", "six-character label colour")
	descriptionFlag := fs.String("description", "", "label description")
	includeLocal := fs.Bool("local", false, "include local OS user and hostname in operation records")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone label create --source owner/repo --slug <slug> --name <name> [flags]")
	}
	if strings.TrimSpace(*sourceFlag) == "" {
		return errors.New("label create requires --source owner/repo")
	}
	slug, err := normalizeLabelSlug(*slugFlag)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*nameFlag) == "" {
		return errors.New("label create requires --name")
	}
	color := strings.TrimSpace(*colorFlag)
	if color != "" && !labelColorPattern.MatchString(color) {
		return errors.New("label colour must be six hex characters")
	}
	color = strings.ToLower(color)
	source, err := parseLocalIssueSource(*sourceFlag)
	if err != nil {
		return err
	}
	if source.System != "waystone" {
		return fmt.Errorf("label create only supports waystone sources, got %s", ledger.SourceSpec(source))
	}

	startedAt := time.Now().UTC()
	reader := ledger.Reader{Root: *root}
	if existing, err := reader.SourceLabelBySlug(source, slug); err == nil && existing.ID != "" {
		return fmt.Errorf("label slug already exists: %s", slug)
	}
	manifestSource := source
	if current, err := reader.Source(source); err == nil {
		manifestSource = current
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	command := "label create"
	operationID := ledger.NewOperationID(command, startedAt)
	manifestSource.Operations = append(manifestSource.Operations, sourceOperationRef(operationID, command, startedAt))
	label := localLabel(manifestSource, slug, *nameFlag, color, *descriptionFlag, startedAt)
	writer := ledger.Writer{Root: *root}
	diff, err := writer.DiffLocalLabel(label)
	if err != nil {
		return err
	}
	ledgerExisted := fileExists(filepath.Join(*root, "ledger.json"))
	if err := writer.WriteLocalLabel(label); err != nil {
		return err
	}
	if err := addLedgerMetadataChange(&diff, *root, ledgerExisted); err != nil {
		return err
	}
	operation := localIssueOperation(operationID, command, args, startedAt, time.Now().UTC(), *root, source, diff, *includeLocal)
	operation.Output.Summary.Labels = 1
	if err := writer.WriteOperation(operation); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Label created")
	writeIndentedField(stdout, "Source", ledger.SourceSpec(source))
	writeIndentedField(stdout, "ID", label.ID)
	writeIndentedField(stdout, "Slug", label.Slug)
	writeIndentedField(stdout, "Name", label.Name)
	return nil
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

func normalizeLabelSlug(value string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(value))
	if slug == "" {
		return "", errors.New("label create requires --slug")
	}
	if !labelSlugPattern.MatchString(slug) {
		return "", fmt.Errorf("label slug must start with a letter or digit and contain only lowercase letters, digits, '.', '_' or '-'")
	}
	return slug, nil
}

func localLabel(source model.Source, slug, name, color, description string, createdAt time.Time) model.Label {
	source.URL = ""
	return model.Label{
		Provenance: model.Provenance{
			ImportID: ledger.SourceSpec(source),
			Source:   source,
		},
		ID:          localLabelID(source, slug, createdAt),
		Slug:        slug,
		Name:        name,
		Color:       color,
		Description: description,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

func localLabelID(source model.Source, slug string, createdAt time.Time) string {
	seed := ledger.SourceSpec(source) + ":" + slug + ":" + createdAt.Format("20060102T150405.000000000Z")
	sum := sha256.Sum256([]byte(seed))
	return "lbl_" + hex.EncodeToString(sum[:])[:16]
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
