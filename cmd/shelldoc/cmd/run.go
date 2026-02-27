// This file is part of shelldoc.
// © 2019, Mirko Boehm <mirko@endocode.com> and the shelldoc contributors
// SPDX-License-Identifier: GPL-3.0

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mirkoboehm/shelldoc/pkg/run"
	"github.com/spf13/cobra"
)

var runContext run.Context

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute a Markdown file as a documentation test",
	Long: `Run parses a Markdown input file, detects the code blocks in it,
executes them and compares their output with the content of the code block.`,
	Run: executeRun,
}

func init() {
	runCmd.Flags().StringVarP(&runContext.ShellName, "shell", "s", "", "The shell to invoke (default: $SHELL)")
	runCmd.Flags().BoolVarP(&runContext.FailureStops, "fail", "f", false, "Stop on the first failure")
	runCmd.Flags().StringVarP(&runContext.XMLOutputFile, "xml", "x", "", "Write results to the specified output file in JUnitXML format")
	runCmd.Flags().BoolVarP(&runContext.ReplaceDots, "replace-dots-in-xml-classname", "d", true, "When using filenames as classnames, replace dots with a unicode circle")
	runCmd.Flags().BoolVarP(&runContext.DryRun, "dry-run", "n", false, "Preview commands without executing them")
	runCmd.Flags().DurationVarP(&runContext.Timeout, "timeout", "t", 0, "Timeout for each command (e.g., 30s, 1m)")
	rootCmd.AddCommand(runCmd)
}

func executeRun(cmd *cobra.Command, args []string) {
	// Set up context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT (Ctrl+C) and SIGTERM for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	runContext.Files = args
	os.Exit(runContext.ExecuteFiles(ctx))
}
