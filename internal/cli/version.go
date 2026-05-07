// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

func runVersion(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("waystone version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "write JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: waystone version [flags]")
	}
	if *jsonOutput {
		return writeJSONOutput(stdout, map[string]string{"version": Version})
	}
	fmt.Fprintln(stdout, Version)
	return nil
}
