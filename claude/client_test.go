package claude

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	clerrors "github.com/flaneur2020/agentkit-go/claude/errors"
)

func TestClientBuilderCommandArgs(t *testing.T) {
	builder := NewClientBuilder().
		WithModel("sonnet").
		WithMaxTurns(3).
		WithMaxBudgetUSD(1.5).
		WithSystemPrompt("sys").
		WithAppendSystemPrompt("append").
		WithAllowedTools("Read", "Glob").
		WithDisallowedTools("Bash").
		WithMCPConfig("/tmp/mcp.json").
		WithIncludePartialMessages(true).
		WithDangerouslySkipPermissions(true).
		WithResume("session-1").
		WithContinue(true).
		WithPermissionMode("acceptEdits")

	args := builder.buildArgs()
	expected := []string{
		"--print", "--output-format", "stream-json", "--verbose",
		"--model", "sonnet",
		"--max-turns", "3",
		"--max-budget-usd", "1.5",
		"--system-prompt", "sys",
		"--append-system-prompt", "append",
		"--allowed-tools", "Read,Glob",
		"--disallowed-tools", "Bash",
		"--mcp-config", "/tmp/mcp.json",
		"--include-partial-messages",
		"--dangerously-skip-permissions",
		"--resume", "session-1",
		"--continue",
		"--permission-mode", "acceptEdits",
	}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("buildArgs() = %#v\nwant %#v", args, expected)
	}
}

func TestClientBuildAndChatWithRealClaude(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := NewClientBuilder().
		WithBinary("claude").
		WithMaxTurns(1).
		WithModel("haiku").
		WithPermissionMode("default").
		Build(ctx)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	input := UserInput{
		Type:   UserInputTypePrompt,
		Prompt: "Reply with exactly: OK\n",
	}
	if err := client.SendUserInput(ctx, input); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}
	go func() {
		_ = client.stdin.Close()
	}()

	var gotSystem bool
	var gotResult bool
	for {
		msg, err := client.NextMessage(ctx)
		if err != nil {
			if clerrors.IsEOF(err) {
				break
			}
			t.Fatalf("NextMessage() error = %v", err)
		}

		switch m := msg.(type) {
		case *SystemMessage:
			gotSystem = true
		case *ResultMessage:
			gotResult = true
			if m.IsError {
				t.Fatalf("result is error: subtype=%s errors=%v", m.Subtype, m.Errors)
			}
			if m.Subtype == "" {
				t.Fatalf("result subtype is empty")
			}
			if m.Result == "" {
				t.Fatalf("result text is empty")
			}
			return
		}
	}

	if !gotSystem {
		t.Fatalf("did not receive system message")
	}
	if !gotResult {
		t.Fatalf("did not receive result message")
	}
}

func TestClientBuilderWithReadWriter(t *testing.T) {
	in := strings.NewReader(`{"type":"result","subtype":"success","is_error":false,"result":"ok"}` + "\n")
	var out bytes.Buffer

	client, err := NewClientBuilder().
		WithReader(in).
		WithWriter(&out).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := client.SendUserInput(context.Background(), UserInput{Prompt: "hello"}); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}
	if out.String() != "hello" {
		t.Fatalf("written = %q, want hello", out.String())
	}

	msg, err := client.NextMessage(context.Background())
	if err != nil {
		t.Fatalf("NextMessage() error = %v", err)
	}
	result, ok := msg.(*ResultMessage)
	if !ok {
		t.Fatalf("type = %T, want *ResultMessage", msg)
	}
	if result.Result != "ok" {
		t.Fatalf("result = %q, want ok", result.Result)
	}
}

func TestClientBuilderWithReadWriterValidation(t *testing.T) {
	_, err := NewClientBuilder().WithReader(strings.NewReader("")).Build(context.Background())
	if err == nil {
		t.Fatalf("Build() error = nil, want validation error")
	}
}

func TestClientBuildWithRealClaudeIncludePartialMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := NewClientBuilder().
		WithBinary("claude").
		WithModel("haiku").
		WithMaxTurns(1).
		WithIncludePartialMessages(true).
		Build(ctx)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.SendUserInput(ctx, UserInput{Type: UserInputTypePrompt, Prompt: "Write exactly: alpha beta gamma\n"}); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}
	go func() { _ = client.stdin.Close() }()

	var gotStreamEvent bool
	var gotResult bool
	for {
		msg, err := client.NextMessage(ctx)
		if err != nil {
			if clerrors.IsEOF(err) {
				break
			}
			t.Fatalf("NextMessage() error = %v", err)
		}
		switch m := msg.(type) {
		case *StreamEventMessage:
			gotStreamEvent = true
			if m.Event.Type == StreamEventTypeContentBlockDelta {
				if m.Event.ContentBlockDelta == nil {
					t.Fatalf("ContentBlockDelta is nil")
				}
			}
		case *ResultMessage:
			gotResult = true
			if m.IsError {
				t.Fatalf("result error: %v", m.Errors)
			}
		}
	}
	if !gotStreamEvent {
		t.Fatalf("did not receive stream_event")
	}
	if !gotResult {
		t.Fatalf("did not receive result")
	}
}

func TestClientBuildWithRealClaudeFileEdit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	tmpFile, err := os.CreateTemp("", "agentkit-file-edit-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString("OLD_VALUE\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	client, err := NewClientBuilder().
		WithBinary("claude").
		WithModel("haiku").
		WithMaxTurns(4).
		WithDangerouslySkipPermissions(true).
		WithAllowedTools("Read", "Edit", "Write").
		Build(ctx)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	prompt := "Use Edit or Write tool to replace OLD_VALUE with NEW_VALUE in file: " + tmpFile.Name() + ". Then reply DONE.\n"
	if err := client.SendUserInput(ctx, UserInput{Type: UserInputTypePrompt, Prompt: prompt}); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}
	go func() { _ = client.stdin.Close() }()

	var result *ResultMessage
	for {
		msg, err := client.NextMessage(ctx)
		if err != nil {
			if clerrors.IsEOF(err) {
				break
			}
			t.Fatalf("NextMessage() error = %v", err)
		}
		if m, ok := msg.(*ResultMessage); ok {
			result = m
		}
	}
	if result == nil {
		t.Fatalf("result message not received")
	}
	if result.IsError {
		t.Fatalf("result is error: subtype=%s errors=%v", result.Subtype, result.Errors)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "NEW_VALUE") {
		t.Fatalf("file content = %q, want contains NEW_VALUE", string(content))
	}
}
