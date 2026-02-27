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
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
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

// StartShell starts a shell as a background process
func StartShell(shell string) (Shell, error) {
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
	return Shell{cmd, stdin, stdout}, nil
}

// commandResult holds the result of a command execution
type commandResult struct {
	output []string
	rc     int
	err    error
}

// ExecuteCommand runs a command in the shell and returns its output and exit code.
// The context can be used to cancel execution (e.g., on SIGINT).
// The timeout parameter specifies a per-command timeout (0 means no timeout).
func (shell *Shell) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) ([]string, int, error) {
	const (
		beginMarker = ">>>>>>>>>>SHELLDOC_MARKER>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
		endMarker   = "<<<<<<<<<<SHELLDOC_MARKER"
	)
	instruction := fmt.Sprintf("%s", strings.TrimSpace(command))
	io.WriteString(shell.stdin, fmt.Sprintf("echo \"%s\"\n", beginMarker))
	io.WriteString(shell.stdin, fmt.Sprintf("%s; echo \"%s $?\"\n", instruction, endMarker))

	beginEx := fmt.Sprintf("^%s$", beginMarker)
	beginRx := regexp.MustCompile(beginEx)
	endEx := fmt.Sprintf("^%s (.+)$", endMarker)
	endRx := regexp.MustCompile(endEx)

	// Run the scanner in a goroutine to support timeout and cancellation
	resultCh := make(chan commandResult, 1)
	go func() {
		var output []string
		var rc int
		beginFound := false
		scanner := bufio.NewScanner(shell.stdout)
		for scanner.Scan() {
			line := scanner.Text()
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
					resultCh <- commandResult{nil, -1, fmt.Errorf("unable to read exit code for shell command: %v", err)}
					return
				}
				rc = value
				break
			}
			output = append(output, line)
		}
		resultCh <- commandResult{output, rc, nil}
	}()

	// Wait for result, timeout, or context cancellation
	if timeout > 0 {
		select {
		case result := <-resultCh:
			return result.output, result.rc, result.err
		case <-time.After(timeout):
			return nil, -1, ErrTimeout
		case <-ctx.Done():
			return nil, -1, ErrCancelled
		}
	}

	// No timeout specified, wait for result or context cancellation
	select {
	case result := <-resultCh:
		return result.output, result.rc, result.err
	case <-ctx.Done():
		return nil, -1, ErrCancelled
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
