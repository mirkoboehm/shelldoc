package run

// This file is part of shelldoc.
// © 2023, Mirko Boehm <mirko@kde.org> and the shelldoc contributors
// SPDX-License-Identifier: LGPL-3.0

import (
	"fmt"
	"io"
	"os"
)

// ReadInput reads either the files specified on the command line or stdin and returns the bytes.
// Markdown.Parse expects bytes, not a stream.
func ReadInput(args []string) ([]byte, error) {
	if len(args) > 0 {
		var result []byte
		for _, filename := range args {
			content, err := os.ReadFile(filename)
			if err != nil {
				return nil, fmt.Errorf("unable to read file %s", filename)
			}
			result = append(result, content[:]...)
		}
		return result, nil
	}
	result, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("unable to read from stdin: %v", err)
	}
	return result, nil
}
