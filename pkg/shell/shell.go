package shell

// This file is part of shelldoc.
// © 2023, Mirko Boehm <mirko@kde.org> and the shelldoc contributors
// SPDX-License-Identifier: LGPL-3.0

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrTimeout is returned when a command exceeds its timeout
var ErrTimeout = errors.New("command execution timed out")

// ErrCancelled is returned when the context is cancelled (e.g., CTRL-C)
var ErrCancelled = errors.New("command execution cancelled")

// Shell represents the shell process that runs in the background and executes the commands.
type Shell struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	scanner     *bufio.Scanner
	mergeStderr bool
}

// DetectShell returns the path to the selected shell or the content of $SHELL
func DetectShell(selected string) (string, error) {
	if len(selected) > 0 {
		// accept what the user said
		log.Printf("Using user-specified shell %s.", selected)
	} else if selected = os.Getenv("SHELL"); len(selected) > 0 {
		log.Printf("Using shell %s (according to $SHELL).", selected)
	} else {
		return "", fmt.Errorf("no shell specified and no $SHELL variable set")
	}
	if _, err := os.Stat(selected); os.IsNotExist(err) {
		return "", fmt.Errorf("the selected shell does not exist: %v", err)
	}
	return selected, nil
}

// StartShell starts a shell as a background process.
// When mergeStderr is true, stderr from each command is redirected into stdout (2>&1).
// When false, stderr is captured separately via a temp file and returned alongside stdout.
func StartShell(shell string, mergeStderr bool) (Shell, error) {
	cmd := exec.Command(shell)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Shell{}, fmt.Errorf("Unable to set up input stream for shell %s: %v", shell, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Shell{}, fmt.Errorf("Unable to set up output stream for shell %s: %v", shell, err)
	}
	err = cmd.Start()
	if err != nil {
		return Shell{}, fmt.Errorf("Unable to start shell %s: %v", shell, err)
	}
	return Shell{cmd, stdin, stdout, bufio.NewScanner(stdout), mergeStderr}, nil
}

// commandResult holds the result of a command execution
type commandResult struct {
	stdout []string
	stderr []string
	rc     int
	err    error
}

// ExecuteCommand runs a command in the shell and returns its stdout, stderr, exit code, and any error.
// The context can be used to cancel execution (e.g., on SIGINT).
// The timeout parameter specifies a per-command timeout (0 means no timeout).
// When shell.mergeStderr is true, stderr is redirected into stdout via 2>&1 and the returned stderr slice is nil.
// When false, stderr is captured to a temp file and returned separately.
func (shell *Shell) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) ([]string, []string, int, error) {
	const (
		beginMarker = ">>>>>>>>>>SHELLDOC_MARKER>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
		endMarker   = "<<<<<<<<<<SHELLDOC_MARKER"
	)

	// Build the instruction, routing stderr as configured.
	trimmed := strings.TrimSpace(command)
	var stderrFile string
	var instruction string
	if shell.mergeStderr {
		instruction = fmt.Sprintf("{ %s; } 2>&1; echo \"%s $?\"\n", trimmed, endMarker)
	} else {
		f, err := os.CreateTemp("", "shelldoc_stderr_*")
		if err == nil {
			stderrFile = f.Name()
			f.Close()
		}
		instruction = fmt.Sprintf("{ %s; } 2>%s; echo \"%s $?\"\n", trimmed, stderrFile, endMarker)
	}

	beginEx := fmt.Sprintf("^%s$", beginMarker)
	beginRx := regexp.MustCompile(beginEx)
	endEx := fmt.Sprintf("^%s (.+)$", endMarker)
	endRx := regexp.MustCompile(endEx)

	io.WriteString(shell.stdin, fmt.Sprintf("echo \"%s\"\n", beginMarker))
	io.WriteString(shell.stdin, instruction)

	// Run the scanner in a goroutine to support timeout and cancellation
	resultCh := make(chan commandResult, 1)
	go func() {
		var stdout []string
		var rc int
		beginFound := false
		for shell.scanner.Scan() {
			line := shell.scanner.Text()
			if beginRx.MatchString(line) {
				beginFound = true
				continue
			}
			if !beginFound {
				continue
			}
			match := endRx.FindStringSubmatch(line)
			if len(match) > 1 {
				value, err := strconv.Atoi(match[1])
				if err != nil {
					resultCh <- commandResult{nil, nil, -1, fmt.Errorf("unable to read exit code for shell command: %v", err)}
					return
				}
				rc = value
				break
			}
			stdout = append(stdout, line)
		}
		// Read stderr from temp file if capturing separately
		var stderr []string
		if stderrFile != "" {
			if data, err := os.ReadFile(stderrFile); err == nil {
				os.Remove(stderrFile)
				if len(data) > 0 {
					stderr = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
				}
			}
		}
		resultCh <- commandResult{stdout, stderr, rc, nil}
	}()

	// Wait for result, timeout, or context cancellation
	if timeout > 0 {
		select {
		case result := <-resultCh:
			return result.stdout, result.stderr, result.rc, result.err
		case <-time.After(timeout):
			return nil, nil, -1, ErrTimeout
		case <-ctx.Done():
			return nil, nil, -1, ErrCancelled
		}
	}

	// No timeout specified, wait for result or context cancellation
	select {
	case result := <-resultCh:
		return result.stdout, result.stderr, result.rc, result.err
	case <-ctx.Done():
		return nil, nil, -1, ErrCancelled
	}
}

// Exit tells a running shell to exit and waits for it
func (shell *Shell) Exit() error {
	io.WriteString(shell.stdin, "exit\n")
	return shell.cmd.Wait()
}

// Kill forcefully terminates the shell process
func (shell *Shell) Kill() error {
	return shell.cmd.Process.Kill()
}
