package shell

// This file is part of shelldoc.
// © 2023, Mirko Boehm <mirko@kde.org> and the shelldoc contributors
// SPDX-License-Identifier: Apache-2.0

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var shellpath string

func TestMain(m *testing.M) {
	shellpath, _ = DetectShell("")
	os.Exit(m.Run())
}
func TestShellLifeCycle(t *testing.T) {
	// The most basic test, start a shell and exit it again
	shell, err := StartShell(shellpath)
	require.NoError(t, err, "Starting a shell should work")
	require.NoError(t, shell.Exit(), "Exiting ad running shell should work")
}

func TestShellLifeCycleRepeated(t *testing.T) {
	// Can the program start and stop a shell repeatedly?
	for counter := 0; counter < 16; counter++ {
		shell, err := StartShell(shellpath)
		require.NoError(t, err, "Starting a shell should work")
		require.NoError(t, shell.Exit(), "Exiting ad running shell should work")
	}
}

func TestReturnCodes(t *testing.T) {
	// Does the shell report return codes corrrectly?
	shell, err := StartShell(shellpath)
	require.NoError(t, err, "Starting a shell should work")
	defer shell.Exit()
	ctx := context.Background()
	{
		output, rc, err := shell.ExecuteCommand(ctx, "true", 0)
		require.NoError(t, err, "The true command is a builtin and should always work")
		require.Equal(t, 0, rc, "The exit code of true should always be zero")
		require.Empty(t, output, "true does not say a word")
	}
	{
		output, rc, err := shell.ExecuteCommand(ctx, "false", 0)
		require.NoError(t, err, "The false command is a builtin and should always work")
		require.NotEqual(t, 0, rc, "The exit code of false should never be zero")
		require.Empty(t, output, "false does not say a word")
	}
}

func TestCaptureOutput(t *testing.T) {
	// Does the shell capture and return the lines printed by the command correctly?
	shell, err := StartShell(shellpath)
	require.NoError(t, err, "Starting a shell should work")
	defer shell.Exit()
	ctx := context.Background()
	{
		const (
			hello = "Hello"
			world = "World"
		)
		output, rc, err := shell.ExecuteCommand(ctx, fmt.Sprintf("echo %s && echo %s", hello, world), 0)
		require.NoError(t, err, "The echo command is a builtin and should always work")
		require.Equal(t, 0, rc, "The exit code of echo should be zero")
		require.Len(t, output, 2, "echo was called twice")
		require.Equal(t, output[0], hello, "you had one job, echo")
		require.Equal(t, output[1], world, "actually, two")
	}
}

func TestTimeout(t *testing.T) {
	// Does the timeout work correctly?
	shell, err := StartShell(shellpath)
	require.NoError(t, err, "Starting a shell should work")
	defer shell.Kill() // Use Kill since shell may be in inconsistent state after timeout
	ctx := context.Background()

	// Command that completes within timeout should succeed
	output, rc, err := shell.ExecuteCommand(ctx, "echo quick", 5*time.Second)
	require.NoError(t, err, "Fast command should not timeout")
	require.Equal(t, 0, rc)
	require.Equal(t, []string{"quick"}, output)
}

func TestTimeoutExpires(t *testing.T) {
	// Does timeout trigger correctly for slow commands?
	shell, err := StartShell(shellpath)
	require.NoError(t, err, "Starting a shell should work")
	defer shell.Kill()
	ctx := context.Background()

	// Command that takes longer than timeout should fail
	start := time.Now()
	_, _, err = shell.ExecuteCommand(ctx, "sleep 10", 100*time.Millisecond)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrTimeout, "Slow command should timeout")
	require.Less(t, elapsed, 1*time.Second, "Timeout should trigger quickly, not wait for command")
}

func TestContextCancellation(t *testing.T) {
	// Does context cancellation work correctly?
	shell, err := StartShell(shellpath)
	require.NoError(t, err, "Starting a shell should work")
	defer shell.Kill()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately
	cancel()

	// Command should fail with ErrCancelled
	_, _, err = shell.ExecuteCommand(ctx, "sleep 10", 0)
	require.ErrorIs(t, err, ErrCancelled, "Command should be cancelled")
}
