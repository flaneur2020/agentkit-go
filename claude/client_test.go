package claude

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type mockProtocol struct {
	sendInput      UserInput
	nextMessage    Message
	initParams     InitializeParams
	callParams     ToolsCallParams
	initialized    bool
	closed         bool
	nextMessageErr error
}

func (m *mockProtocol) SendUserInput(ctx context.Context, input UserInput) error {
	m.sendInput = input
	return nil
}

func (m *mockProtocol) NextMessage(ctx context.Context) (Message, error) {
	return m.nextMessage, m.nextMessageErr
}

func (m *mockProtocol) MCPInitialize(ctx context.Context, params InitializeParams) (*InitializeResult, error) {
	m.initParams = params
	return &InitializeResult{ProtocolVersion: "2024-11-05"}, nil
}

func (m *mockProtocol) MCPInitialized(ctx context.Context) error {
	m.initialized = true
	return nil
}

func (m *mockProtocol) MCPToolsList(ctx context.Context) (*ToolsListResult, error) {
	return &ToolsListResult{Tools: []ToolDefinition{{Name: "calculator"}}}, nil
}

func (m *mockProtocol) MCPToolsCall(ctx context.Context, params ToolsCallParams) (*ToolsCallResult, error) {
	m.callParams = params
	return &ToolsCallResult{Content: []ToolResultContent{{Type: "text", Text: "ok"}}}, nil
}

func (m *mockProtocol) Close() error {
	m.closed = true
	return nil
}

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

func TestClientForwardsProtocolMethods(t *testing.T) {
	mock := &mockProtocol{
		nextMessage: &ResultMessage{Type: MessageTypeResult, Subtype: "success"},
	}
	client := &Client{protocol: mock}

	if err := client.SendUserInput(context.Background(), UserInput{Prompt: "hello"}); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}
	if mock.sendInput.Prompt != "hello" {
		t.Fatalf("prompt = %q, want hello", mock.sendInput.Prompt)
	}

	msg, err := client.NextMessage(context.Background())
	if err != nil {
		t.Fatalf("NextMessage() error = %v", err)
	}
	if msg.GetType() != MessageTypeResult {
		t.Fatalf("message type = %s, want result", msg.GetType())
	}

	_, err = client.MCPInitialize(context.Background(), InitializeParams{ClientInfo: ClientInfo{Name: "agentkit"}})
	if err != nil {
		t.Fatalf("MCPInitialize() error = %v", err)
	}
	if mock.initParams.ClientInfo.Name != "agentkit" {
		t.Fatalf("init client name = %q, want agentkit", mock.initParams.ClientInfo.Name)
	}

	if err := client.MCPInitialized(context.Background()); err != nil {
		t.Fatalf("MCPInitialized() error = %v", err)
	}
	if !mock.initialized {
		t.Fatalf("MCPInitialized() was not forwarded")
	}

	_, err = client.MCPToolsCall(context.Background(), ToolsCallParams{Name: "calculator"})
	if err != nil {
		t.Fatalf("MCPToolsCall() error = %v", err)
	}
	if mock.callParams.Name != "calculator" {
		t.Fatalf("tool call name = %q, want calculator", mock.callParams.Name)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !mock.closed {
		t.Fatalf("Close() was not forwarded")
	}
}

func TestClientBuilderBuildAndProtocolIntegration(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-claude.sh")
	script := "#!/bin/sh\ncat\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	client, err := NewClientBuilder().
		WithBinary(bin).
		WithModel("sonnet").
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	input := UserInput{Prompt: "{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false}\n"}
	if err := client.SendUserInput(context.Background(), input); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}

	msg, err := client.NextMessage(context.Background())
	if err != nil {
		t.Fatalf("NextMessage() error = %v", err)
	}
	result, ok := msg.(*ResultMessage)
	if !ok {
		t.Fatalf("msg type = %T, want *ResultMessage", msg)
	}
	if result.Subtype != "success" {
		t.Fatalf("result subtype = %q, want success", result.Subtype)
	}
}
