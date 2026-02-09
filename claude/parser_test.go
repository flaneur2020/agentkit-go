package claude

import (
	"strings"
	"testing"

	agenterrors "agentkit/errors"
)

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

func TestParserNextEOF(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	_, err := parser.Next()
	if !agenterrors.IsEOF(err) {
		t.Fatalf("Next() err = %v, want EOF marker", err)
	}
}
