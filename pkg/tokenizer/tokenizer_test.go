package tokenizer

// This file is part of shelldoc.
// © 2023, Mirko Boehm <mirko@kde.org> and the shelldoc contributors
// SPDX-License-Identifier: Apache-2.0

import (
	"os"
	"strings"
	"testing"

	blackfriday "github.com/russross/blackfriday/v2"
	"github.com/stretchr/testify/require"
)

var echoTrueCodeBlockCount int

func codeBlockHandler(visitor *Visitor, node *blackfriday.Node) blackfriday.WalkStatus {
	//fmt.Printf("%s: %v\n", node.Type, string(node.Literal))
	echoTrueCodeBlockCount++
	return blackfriday.GoToNext
}
func TestEchoTrue(t *testing.T) {
	data, err := os.ReadFile("samples/echotrue.md")
	require.NoError(t, err, "Unable to read sample data file")
	visitor := Visitor{codeBlockHandler, codeBlockHandler, nil}
	require.Zero(t, echoTrueCodeBlockCount, "Starting the counter")
	Tokenize(data, &visitor)
	require.Equal(t, echoTrueCodeBlockCount, 1, "There is one code block element in the sample file")
}

func TestTokenizeEchoTrue(t *testing.T) {
	data, err := os.ReadFile("samples/echotrue.md")
	require.NoError(t, err, "Unable to read sample data file")
	visitor := NewInteractionVisitor()
	Tokenize(data, visitor)
	require.Equal(t, len(visitor.Interactions), 1, "There is one code block element in the sample file")
}

func TestTokenizeHelloWorld(t *testing.T) {
	data, err := os.ReadFile("samples/helloworld.md")
	require.NoError(t, err, "Unable to read sample data file")
	visitor := NewInteractionVisitor()
	Tokenize(data, visitor)
	require.Equal(t, 4, len(visitor.Interactions), "There are three code block elements with a total of 4 interactions in the sample file")
	require.Empty(t, visitor.Interactions[0].Response, "The first command does not expect a response")
	require.NotEmpty(t, visitor.Interactions[1].Response, "The second command expects a response")
	require.Equal(t, visitor.Interactions[1].Response[0], "Hello", "The second command expects a response")
	require.NotEmpty(t, visitor.Interactions[2].Response, "The third command expects a response")
	require.Equal(t, visitor.Interactions[2].Response[0], "World", "The third command expects a response")
	fourth := visitor.Interactions[3]
	require.Equal(t, 2, len(fourth.Response), "The response of the fourth interaction contains two lines")
	require.Equal(t, "...", fourth.Response[1], "The last line of the fourth response is an ellipsis")
}

func TestTokenizeFenced(t *testing.T) {
	data, err := os.ReadFile("samples/fenced.md")
	require.NoError(t, err, "Unable to read sample data file")
	visitor := NewInteractionVisitor()
	Tokenize(data, visitor)
	require.Equal(t, len(visitor.Interactions), 2, "There are two fenced code block in the sample file.")
	first := visitor.Interactions[0]
	require.Equal(t, first.Language, "shell", "shell was the specified languagwe for the first code block")
	require.Equal(t, first.Attributes["shelldocexitcode"], "1", "1 was the specified value for shelldocexitcode")
	require.Equal(t, first.Attributes["shelldocwhatever"], "", "shelldocwhatever comes with no value")
	_, exists := first.Attributes["shelldocnonsense"]
	require.False(t, exists, "shelldocnonsense was not defined")
	second := visitor.Interactions[1]
	require.Empty(t, second.Language, "No language was specified in the second block")
	require.Empty(t, second.Attributes, "No attributes where specified in the second block")
}

func TestParseCodeBlockInfoString(t *testing.T) {
	tests := []struct {
		name           string
		infostring     string
		expectLang     string
		expectAttrs    map[string]string
	}{
		{
			name:        "empty info string",
			infostring:  "",
			expectLang:  "",
			expectAttrs: map[string]string{},
		},
		{
			name:        "language only",
			infostring:  "bash",
			expectLang:  "",
			expectAttrs: map[string]string{},
		},
		{
			name:        "language with attributes",
			infostring:  "shell {shelldocexitcode=0}",
			expectLang:  "shell",
			expectAttrs: map[string]string{"shelldocexitcode": "0"},
		},
		{
			name:        "language with multiple attributes",
			infostring:  "bash {shelldocexitcode=1 shelldocwhatever}",
			expectLang:  "bash",
			expectAttrs: map[string]string{"shelldocexitcode": "1", "shelldocwhatever": ""},
		},
		{
			name:        "non-shelldoc attributes ignored",
			infostring:  "shell {.class other=value shelldocexitcode=2}",
			expectLang:  "shell",
			expectAttrs: map[string]string{"shelldocexitcode": "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, attrs := parseCodeBlockInfoString(tt.infostring)
			require.Equal(t, tt.expectLang, lang)
			for k, v := range tt.expectAttrs {
				require.Equal(t, v, attrs[k])
			}
		})
	}
}

func TestTokenizeEmptyCodeBlock(t *testing.T) {
	md := []byte("# Empty code block\n\n```\n```\n")
	visitor := NewInteractionVisitor()
	err := Tokenize(md, visitor)
	require.NoError(t, err)
	require.Empty(t, visitor.Interactions, "Empty code block should produce no interactions")
}

func TestTokenizeCodeBlockWithoutTrigger(t *testing.T) {
	md := []byte("# Code without trigger\n\n```\nsome output without $ or > prefix\n```\n")
	visitor := NewInteractionVisitor()
	err := Tokenize(md, visitor)
	require.NoError(t, err)
	require.Empty(t, visitor.Interactions, "Lines without trigger should be skipped")
}

func TestTokenizeCodeBlockResponseBeforeCommand(t *testing.T) {
	md := []byte("# Response before command\n\n```\norphan response line\n$ echo hello\n```\n")
	visitor := NewInteractionVisitor()
	err := Tokenize(md, visitor)
	require.NoError(t, err)
	require.Len(t, visitor.Interactions, 1, "Should have one interaction")
	require.Equal(t, "echo hello", visitor.Interactions[0].Cmd)
}

func TestInteractionDescribe(t *testing.T) {
	interaction := &Interaction{
		Cmd:      "echo hello",
		Response: []string{"hello"},
	}
	desc := interaction.Describe()
	require.Contains(t, desc, "echo hello")
	require.Contains(t, desc, "hello")
}

func TestInteractionDescribeWithCaption(t *testing.T) {
	interaction := &Interaction{
		Cmd:      "echo hello",
		Caption:  "Test Caption",
		Response: []string{"hello"},
	}
	desc := interaction.Describe()
	require.Contains(t, desc, "Test Caption")
}

func TestInteractionDescribeNoResponse(t *testing.T) {
	interaction := &Interaction{
		Cmd:      "true",
		Response: nil,
	}
	desc := interaction.Describe()
	require.Contains(t, desc, "(no response expected)")
}

func TestInteractionDescribeFull(t *testing.T) {
	interaction := &Interaction{
		Cmd:      "echo hello",
		Response: []string{"hello"},
		Output:   []string{"hello"},
	}
	desc := interaction.DescribeFull()
	require.Contains(t, desc, "got:")
	require.Contains(t, desc, "want:")
}

func TestInteractionResult(t *testing.T) {
	tests := []struct {
		resultCode int
		response   []string
		expected   string
	}{
		{NewInteraction, nil, "not executed"},
		{ResultExecutionError, nil, "ERROR (result not evaluated)"},
		{ResultMatch, nil, "PASS (execution successful)"},
		{ResultMatch, []string{"hello"}, "PASS (match)"},
		{ResultRegexMatch, nil, "PASS (regex match)"},
		{ResultMismatch, nil, "FAIL (mismatch)"},
		{ResultError, nil, "FAIL (execution failed)"},
		{999, nil, "YOU FOUND A BUG!!11!1!"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			interaction := &Interaction{
				ResultCode: tt.resultCode,
				Response:   tt.response,
			}
			require.Equal(t, tt.expected, interaction.Result())
		})
	}
}

func TestInteractionHasFailure(t *testing.T) {
	tests := []struct {
		resultCode int
		expected   bool
	}{
		{NewInteraction, false},
		{ResultExecutionError, false},
		{ResultMatch, false},
		{ResultRegexMatch, false},
		{ResultMismatch, true},
		{ResultError, true},
	}

	for _, tt := range tests {
		interaction := &Interaction{ResultCode: tt.resultCode}
		require.Equal(t, tt.expected, interaction.HasFailure())
	}
}

func TestNewInteraction(t *testing.T) {
	interaction := New("test caption")
	require.NotNil(t, interaction)
	require.Equal(t, "test caption", interaction.Caption)
}

func TestEvaluateResponse(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		actual   []string
		match    bool
	}{
		{"exact match", []string{"hello"}, []string{"hello"}, true},
		{"mismatch", []string{"hello"}, []string{"world"}, false},
		{"both empty", []string{}, []string{}, true},
		{"ellipsis truncates", []string{"hello", "..."}, []string{"hello", "extra", "lines"}, true},
		{"ellipsis at start", []string{"..."}, []string{"anything", "here"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interaction := &Interaction{Response: tt.expected}
			result := interaction.evaluateResponse(tt.actual)
			require.Equal(t, tt.match, result)
		})
	}
}

func TestElideString(t *testing.T) {
	require.Equal(t, "short", elideString("short", 20))
	require.Equal(t, "this is...", elideString("this is a long string", 10))
	require.Equal(t, "tiny", elideString("tiny", 5))
}

func TestCompareRegex(t *testing.T) {
	interaction := &Interaction{}
	require.False(t, interaction.compareRegex([]string{"any"}))
}

func TestTokenizeMultipleCommands(t *testing.T) {
	md := []byte("```\n$ echo one\none\n$ echo two\ntwo\n```\n")
	visitor := NewInteractionVisitor()
	Tokenize(md, visitor)
	require.Len(t, visitor.Interactions, 2)
	require.Equal(t, "echo one", visitor.Interactions[0].Cmd)
	require.Equal(t, "echo two", visitor.Interactions[1].Cmd)
}

func TestTokenizeGreaterThanPrompt(t *testing.T) {
	md := []byte("```\n> echo hello\nhello\n```\n")
	visitor := NewInteractionVisitor()
	Tokenize(md, visitor)
	require.Len(t, visitor.Interactions, 1)
	require.Equal(t, "echo hello", visitor.Interactions[0].Cmd)
}

func TestInteractionDescribeLongCommand(t *testing.T) {
	longCmd := strings.Repeat("x", 50)
	interaction := &Interaction{Cmd: longCmd}
	desc := interaction.Describe()
	require.Contains(t, desc, "...")
}
