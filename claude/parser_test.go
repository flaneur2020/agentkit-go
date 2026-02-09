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
	if assistantMsg.Message.Content[0].Text != "hello" {
		t.Fatalf("text = %q, want hello", assistantMsg.Message.Content[0].Text)
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
	if eventMsg.Event.Delta == nil || eventMsg.Event.Delta.Text != "Hi" {
		t.Fatalf("delta text = %v, want Hi", eventMsg.Event.Delta)
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
