// This file is part of shelldoc.
// © 2023, Mirko Boehm <mirko@kde.org> and the shelldoc contributors
// SPDX-License-Identifier: GPL-3.0

package cmd

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootCommandExists(t *testing.T) {
	require.NotNil(t, rootCmd, "root command should exist")
	require.Equal(t, "shelldoc", rootCmd.Use)
}

func TestRootCommandHasVerboseFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, flag, "verbose flag should exist")
	require.Equal(t, "v", flag.Shorthand)
	require.Equal(t, "false", flag.DefValue)
}

func TestVersionCommandExists(t *testing.T) {
	versionFound := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			versionFound = true
			require.Equal(t, "Print the shelldoc version", cmd.Short)
			break
		}
	}
	require.True(t, versionFound, "version command should be registered")
}

func TestVersionCommandOutput(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	require.NoError(t, err)
	require.NotEmpty(t, buf.String(), "version command should produce output")
}

func TestRunCommandExists(t *testing.T) {
	runFound := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "run" {
			runFound = true
			require.Equal(t, "Execute a Markdown file as a documentation test", cmd.Short)
			break
		}
	}
	require.True(t, runFound, "run command should be registered")
}

func TestRunCommandFlags(t *testing.T) {
	var runCommand *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "run" {
			runCommand = cmd
			break
		}
	}
	require.NotNil(t, runCommand, "run command should exist")

	tests := []struct {
		name      string
		shorthand string
		defValue  string
	}{
		{"shell", "s", ""},
		{"fail", "f", "false"},
		{"xml", "x", ""},
		{"replace-dots-in-xml-classname", "d", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := runCommand.Flags().Lookup(tt.name)
			require.NotNil(t, flag, "flag %s should exist", tt.name)
			require.Equal(t, tt.shorthand, flag.Shorthand)
			require.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestInitLoggingVerboseDisabled(t *testing.T) {
	verbose = false
	initLogging()
	// When verbose is disabled, log output should go to discard
	// After initLogging with verbose=false, log output goes to io.Discard
	// We can verify the prefix and flags are set correctly
	require.Equal(t, io.Discard, log.Writer())
	require.Equal(t, "Note: ", log.Prefix())
	require.Equal(t, 0, log.Flags())
}

func TestInitLoggingVerboseEnabled(t *testing.T) {
	verbose = true
	initLogging()
	// When verbose is enabled, log output should go to stderr
	require.Equal(t, os.Stderr, log.Writer())
	require.Equal(t, "Note: ", log.Prefix())
	require.Equal(t, 0, log.Flags())

	// Reset for other tests
	verbose = false
	initLogging()
}
