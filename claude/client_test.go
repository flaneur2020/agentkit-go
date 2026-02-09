package claude

import (
	"context"
	"reflect"
	"testing"
	"time"

	agenterrors "agentkit/errors"
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
			if agenterrors.IsEOF(err) {
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
