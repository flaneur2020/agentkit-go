package claude

import (
	"strings"
	"testing"

	agenterrors "agentkit/errors"
)

// The parser test suite covers all stream-json message kinds in the spec:
// system / assistant / user / result / stream_event,
// plus unknown type fallback, invalid JSON handling, empty-line skipping, and EOF marker.

func TestParserParseLineSystemMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"system","subtype":"init","session_id":"s1","model":"claude-sonnet"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}

	systemMsg, ok := msg.(*SystemMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *SystemMessage", msg)
	}
	if systemMsg.Subtype != "init" {
		t.Fatalf("Subtype = %q, want init", systemMsg.Subtype)
	}
	if systemMsg.SessionID != "s1" {
		t.Fatalf("SessionID = %q, want s1", systemMsg.SessionID)
	}
}

func TestParserParseLineUnknownType(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"other","foo":"bar"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}

	unknownMsg, ok := msg.(*UnknownMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *UnknownMessage", msg)
	}
	if unknownMsg.Type != "other" {
		t.Fatalf("Type = %q, want other", unknownMsg.Type)
	}
	if string(unknownMsg.Raw) != string(line) {
		t.Fatalf("Raw = %s, want %s", unknownMsg.Raw, line)
	}
}

func TestParserParseLineAssistantMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}

	assistantMsg, ok := msg.(*AssistantMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *AssistantMessage", msg)
	}
	if len(assistantMsg.Message.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(assistantMsg.Message.Content))
	}
	if assistantMsg.Message.Content[0].Type != ContentBlockTypeText {
		t.Fatalf("type = %q, want text", assistantMsg.Message.Content[0].Type)
	}
	if assistantMsg.Message.Content[0].Text == nil || assistantMsg.Message.Content[0].Text.Text != "hello" {
		t.Fatalf("text = %+v, want hello", assistantMsg.Message.Content[0].Text)
	}
}

func TestParserParseLineUserMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}

	userMsg, ok := msg.(*UserMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *UserMessage", msg)
	}
	if userMsg.Message.Role != "user" {
		t.Fatalf("role = %q, want user", userMsg.Message.Role)
	}
	if len(userMsg.Message.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(userMsg.Message.Content))
	}
	if userMsg.Message.Content[0].Type != ContentBlockTypeToolResult {
		t.Fatalf("type = %q, want tool_result", userMsg.Message.Content[0].Type)
	}
	if userMsg.Message.Content[0].ToolResult == nil || userMsg.Message.Content[0].ToolResult.ToolUseID != "toolu_1" {
		t.Fatalf("tool result = %+v, want toolu_1", userMsg.Message.Content[0].ToolResult)
	}
}

func TestParserParseLineResultMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"result","subtype":"success","is_error":false,"result":"done"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}

	resultMsg, ok := msg.(*ResultMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *ResultMessage", msg)
	}
	if resultMsg.Subtype != "success" {
		t.Fatalf("subtype = %q, want success", resultMsg.Subtype)
	}
}

func TestParserParseLineStreamEventMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hi"}}}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}

	eventMsg, ok := msg.(*StreamEventMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *StreamEventMessage", msg)
	}
	if eventMsg.Event.Type != "content_block_delta" {
		t.Fatalf("event type = %q, want content_block_delta", eventMsg.Event.Type)
	}
	if eventMsg.Event.ContentBlockDelta == nil || eventMsg.Event.ContentBlockDelta.Delta.Text != "Hi" {
		t.Fatalf("delta text = %+v, want Hi", eventMsg.Event.ContentBlockDelta)
	}
}

func TestParserParseLineStreamEventAllTypes(t *testing.T) {
	cases := []struct {
		name  string
		line  string
		check func(t *testing.T, m *StreamEventMessage)
	}{
		{
			name: "message_start",
			line: `{"type":"stream_event","event":{"type":"message_start","message":{}}}`,
			check: func(t *testing.T, m *StreamEventMessage) {
				if m.Event.MessageStart == nil {
					t.Fatalf("MessageStart is nil")
				}
			},
		},
		{
			name: "content_block_start",
			line: `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}}`,
			check: func(t *testing.T, m *StreamEventMessage) {
				if m.Event.ContentBlockStart == nil {
					t.Fatalf("ContentBlockStart is nil")
				}
			},
		},
		{
			name: "content_block_stop",
			line: `{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`,
			check: func(t *testing.T, m *StreamEventMessage) {
				if m.Event.ContentBlockStop == nil {
					t.Fatalf("ContentBlockStop is nil")
				}
			},
		},
		{
			name: "message_delta",
			line: `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`,
			check: func(t *testing.T, m *StreamEventMessage) {
				if m.Event.MessageDelta == nil {
					t.Fatalf("MessageDelta is nil")
				}
			},
		},
		{
			name: "message_stop",
			line: `{"type":"stream_event","event":{"type":"message_stop"}}`,
			check: func(t *testing.T, m *StreamEventMessage) {
				if m.Event.MessageStop == nil {
					t.Fatalf("MessageStop is nil")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewMessageParser(strings.NewReader(""))
			msg, err := parser.ParseLine([]byte(tc.line))
			if err != nil {
				t.Fatalf("ParseLine() error = %v", err)
			}
			eventMsg, ok := msg.(*StreamEventMessage)
			if !ok {
				t.Fatalf("type = %T, want *StreamEventMessage", msg)
			}
			tc.check(t, eventMsg)
		})
	}
}

func TestParserParseLineAssistantMessageToolUseResult(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[]},"tool_use_result":{"filenames":["a.txt"],"durationMs":7,"numFiles":1,"truncated":false}}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	assistantMsg, ok := msg.(*AssistantMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *AssistantMessage", msg)
	}
	if assistantMsg.ToolUseResult == nil {
		t.Fatalf("tool_use_result = nil, want value")
	}
	if assistantMsg.ToolUseResult.DurationMS != 7 {
		t.Fatalf("durationMs = %d, want 7", assistantMsg.ToolUseResult.DurationMS)
	}
}

func TestParserParseLineResultMessageFullFields(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"result","subtype":"error_during_execution","is_error":true,"result":"","stop_reason":"end_turn","permission_denials":[{"tool_name":"Bash","tool_use_id":"toolu_1","tool_input":{"command":"rm -rf /"}}],"modelUsage":{"claude-haiku-4-5-20251001":{"inputTokens":1,"outputTokens":2,"cacheReadInputTokens":3,"cacheCreationInputTokens":4,"webSearchRequests":0,"costUSD":0.1,"contextWindow":200000,"maxOutputTokens":64000}},"structured_output":{"ok":true},"errors":["x"]}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	resultMsg, ok := msg.(*ResultMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *ResultMessage", msg)
	}
	if !resultMsg.IsError {
		t.Fatalf("is_error = false, want true")
	}
	if resultMsg.StopReason == nil || *resultMsg.StopReason != "end_turn" {
		t.Fatalf("stop_reason = %+v, want end_turn", resultMsg.StopReason)
	}
	if len(resultMsg.PermissionDenials) != 1 || resultMsg.PermissionDenials[0].ToolName != "Bash" {
		t.Fatalf("permission_denials = %+v, want Bash", resultMsg.PermissionDenials)
	}
	if len(resultMsg.ModelUsage) != 1 {
		t.Fatalf("modelUsage len = %d, want 1", len(resultMsg.ModelUsage))
	}
	if string(resultMsg.StructuredOutput) == "" {
		t.Fatalf("structured_output is empty")
	}
	if len(resultMsg.Errors) != 1 || resultMsg.Errors[0] != "x" {
		t.Fatalf("errors = %+v, want [x]", resultMsg.Errors)
	}
}

func TestParserParseLineInvalidJSON(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	_, err := parser.ParseLine([]byte(`{"type":`))
	if err == nil {
		t.Fatalf("ParseLine() error = nil, want error")
	}
}

func TestParserNextSkipsEmptyLines(t *testing.T) {
	parser := NewMessageParser(strings.NewReader("\n \n{" + `"type":"result","subtype":"success","is_error":false` + "}\n"))
	msg, err := parser.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if msg.GetType() != MessageTypeResult {
		t.Fatalf("type = %s, want result", msg.GetType())
	}
}

func TestParserNextEOF(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	_, err := parser.Next()
	if !agenterrors.IsEOF(err) {
		t.Fatalf("Next() err = %v, want EOF marker", err)
	}
}
