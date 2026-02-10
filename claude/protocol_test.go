package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	clerrors "github.com/flaneur2020/agentkit-go/claude/errors"
)

// The protocol test suite covers:
// - chat inputs (prompt / permission / raw) and validation errors,
// - MCP initialize/initialized/tools/list/tools/call,
// - JSON-RPC error/EOF/non-matching response branches.

func TestProtocolSendUserInput(t *testing.T) {
	var out bytes.Buffer
	p := NewProtocol(strings.NewReader(""), &out)

	input := UserInput{Prompt: "hello world"}
	if err := p.SendUserInput(context.Background(), input); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}
	if out.String() != "hello world" {
		t.Fatalf("written = %q, want %q", out.String(), "hello world")
	}
}

func TestProtocolSendUserInputRaw(t *testing.T) {
	var out bytes.Buffer
	p := NewProtocol(strings.NewReader(""), &out)

	input := UserInput{Type: UserInputTypeRaw, Raw: "{\"action\":\"continue\"}"}
	if err := p.SendUserInput(context.Background(), input); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}
	if out.String() != input.Raw {
		t.Fatalf("written = %q, want %q", out.String(), input.Raw)
	}
}

func TestProtocolSendUserInputErrors(t *testing.T) {
	var out bytes.Buffer
	p := NewProtocol(strings.NewReader(""), &out)

	if err := p.SendUserInput(context.Background(), UserInput{Type: UserInputTypePrompt, Prompt: "   "}); err == nil {
		t.Fatalf("prompt empty should return error")
	}
	if err := p.SendUserInput(context.Background(), UserInput{Type: UserInputTypeRaw, Raw: ""}); err == nil {
		t.Fatalf("raw empty should return error")
	}
	if err := p.SendUserInput(context.Background(), UserInput{Type: UserInputTypePermission}); err == nil {
		t.Fatalf("nil permission should return error")
	}
	if err := p.SendUserInput(context.Background(), UserInput{
		Type: UserInputTypePermission,
		Permission: &PermissionInput{
			Decision: "maybe",
		},
	}); err == nil {
		t.Fatalf("invalid permission decision should return error")
	}
	if err := p.SendUserInput(context.Background(), UserInput{Type: "unknown", Prompt: "x"}); err == nil {
		t.Fatalf("unknown input type should return error")
	}
}

func TestProtocolNextMessageContextCancelled(t *testing.T) {
	p := NewProtocol(strings.NewReader(""), &bytes.Buffer{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.NextMessage(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("NextMessage() err = %v, want context.Canceled", err)
	}
}

func TestProtocolNextMessageContextCancelledWhileBlocked(t *testing.T) {
	r, w := io.Pipe()
	p := NewProtocol(r, &bytes.Buffer{})
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := p.NextMessage(ctx)
		errCh <- err
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("NextMessage() err = %v, want context.Canceled", err)
	}
}

func TestProtocolMCPRequestError(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}` + "\n")
	p := NewProtocol(in, &bytes.Buffer{})

	_, err := p.MCPToolsList(context.Background())
	if err == nil {
		t.Fatalf("MCPToolsList() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "jsonrpc error -32601") {
		t.Fatalf("error = %v, want jsonrpc error", err)
	}
}

func TestProtocolMCPRequestDecodeError(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":"bad"}` + "\n")
	p := NewProtocol(in, &bytes.Buffer{})

	_, err := p.MCPToolsList(context.Background())
	if err == nil {
		t.Fatalf("MCPToolsList() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "decode tools/list result") {
		t.Fatalf("error = %v, want decode error", err)
	}
}

func TestProtocolMCPRequestSkipsNonMatchingResponses(t *testing.T) {
	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":999,"result":{"tools":[{"name":"wrong"}]}}` + "\n" +
			`{"type":"system","subtype":"init"}` + "\n" +
			`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"calculator"}]}}` + "\n",
	)
	var out bytes.Buffer
	p := NewProtocol(in, &out)

	resp, err := p.MCPToolsList(context.Background())
	if err != nil {
		t.Fatalf("MCPToolsList() error = %v", err)
	}
	if len(resp.Tools) != 1 || resp.Tools[0].Name != "calculator" {
		t.Fatalf("tools = %+v, want calculator", resp.Tools)
	}
}

func TestProtocolMCPRequestEOF(t *testing.T) {
	p := NewProtocol(strings.NewReader(""), &bytes.Buffer{})
	_, err := p.MCPToolsList(context.Background())
	if !clerrors.IsEOF(err) {
		t.Fatalf("MCPToolsList() err = %v, want EOF", err)
	}
}

func TestProtocolSendUserInputPermission(t *testing.T) {
	var out bytes.Buffer
	p := NewProtocol(strings.NewReader(""), &out)

	input := UserInput{
		Type: UserInputTypePermission,
		Permission: &PermissionInput{
			Decision:  PermissionDecisionAllow,
			ToolUseID: "toolu_123",
			Reason:    "approved",
		},
	}
	if err := p.SendUserInput(context.Background(), input); err != nil {
		t.Fatalf("SendUserInput() error = %v", err)
	}

	var payload PermissionInput
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal permission payload: %v", err)
	}
	if payload.Decision != PermissionDecisionAllow {
		t.Fatalf("decision = %q, want allow", payload.Decision)
	}
	if payload.ToolUseID != "toolu_123" {
		t.Fatalf("tool_use_id = %q, want toolu_123", payload.ToolUseID)
	}
}

func TestProtocolMCPInitializeAndInitialized(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"test-server","version":"1.0.0"}}}` + "\n")
	var out bytes.Buffer
	p := NewProtocol(in, &out)

	resp, err := p.MCPInitialize(context.Background(), InitializeParams{ClientInfo: ClientInfo{Name: "agentkit", Version: "dev"}})
	if err != nil {
		t.Fatalf("MCPInitialize() error = %v", err)
	}
	if resp.ProtocolVersion != "2024-11-05" {
		t.Fatalf("ProtocolVersion = %q, want 2024-11-05", resp.ProtocolVersion)
	}
	if resp.ServerInfo.Name != "test-server" {
		t.Fatalf("ServerInfo.Name = %q, want test-server", resp.ServerInfo.Name)
	}

	var req JSONRPCRequest
	line := strings.Split(strings.TrimSpace(out.String()), "\n")[0]
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if req.Method != "initialize" {
		t.Fatalf("method = %q, want initialize", req.Method)
	}
	if req.ID == nil || *req.ID != 1 {
		t.Fatalf("id = %v, want 1", req.ID)
	}

	out.Reset()
	if err := p.MCPInitialized(context.Background()); err != nil {
		t.Fatalf("MCPInitialized() error = %v", err)
	}

	var notify JSONRPCRequest
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &notify); err != nil {
		t.Fatalf("unmarshal notify: %v", err)
	}
	if notify.Method != "initialized" {
		t.Fatalf("method = %q, want initialized", notify.Method)
	}
	if notify.ID != nil {
		t.Fatalf("notify id = %v, want nil", notify.ID)
	}
}

func TestProtocolMCPToolsListAndCall(t *testing.T) {
	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"calculator","description":"calc"}]}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"42"}]}}` + "\n",
	)
	var out bytes.Buffer
	p := NewProtocol(in, &out)

	listResp, err := p.MCPToolsList(context.Background())
	if err != nil {
		t.Fatalf("MCPToolsList() error = %v", err)
	}
	if len(listResp.Tools) != 1 || listResp.Tools[0].Name != "calculator" {
		t.Fatalf("tools = %+v, want calculator", listResp.Tools)
	}

	callResp, err := p.MCPToolsCall(context.Background(), ToolsCallParams{
		Name: "calculator",
		Arguments: map[string]interface{}{
			"operation": "multiply",
			"a":         7,
			"b":         6,
		},
	})
	if err != nil {
		t.Fatalf("MCPToolsCall() error = %v", err)
	}
	if len(callResp.Content) != 1 || callResp.Content[0].Text != "42" {
		t.Fatalf("call content = %+v, want text=42", callResp.Content)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("written lines = %d, want 2", len(lines))
	}

	var req1 JSONRPCRequest
	if err := json.Unmarshal([]byte(lines[0]), &req1); err != nil {
		t.Fatalf("unmarshal req1: %v", err)
	}
	if req1.Method != "tools/list" {
		t.Fatalf("req1 method = %q, want tools/list", req1.Method)
	}
	if req1.ID == nil || *req1.ID != 1 {
		t.Fatalf("req1 id = %v, want 1", req1.ID)
	}

	var req2 JSONRPCRequest
	if err := json.Unmarshal([]byte(lines[1]), &req2); err != nil {
		t.Fatalf("unmarshal req2: %v", err)
	}
	if req2.Method != "tools/call" {
		t.Fatalf("req2 method = %q, want tools/call", req2.Method)
	}
	if req2.ID == nil || *req2.ID != 2 {
		t.Fatalf("req2 id = %v, want 2", req2.ID)
	}
}
