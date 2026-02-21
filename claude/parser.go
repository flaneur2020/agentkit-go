package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	clerrors "github.com/flaneur2020/agentkit-go/claude/errors"
)

type MessageParser interface {
	ParseLine(line []byte) (Message, error)
	Next() (Message, error)
}

type messageParser struct {
	scanner *bufio.Scanner
}

func NewMessageParser(r io.Reader) MessageParser {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	return &messageParser{scanner: scanner}
}

func (p *messageParser) ParseLine(line []byte) (Message, error) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil, nil
	}

	var env messageEnvelope
	if err := json.Unmarshal(trimmed, &env); err != nil {
		return nil, formatParseError("parse message envelope", trimmed, err)
	}

	switch env.Type {
	case MessageTypeSystem:
		var msg SystemMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return unknownFromParseFailure(env.Type, trimmed, fmt.Errorf("parse system message: %w", err)), nil
		}
		return &msg, nil
	case MessageTypeAssistant:
		var msg AssistantMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return unknownFromParseFailure(env.Type, trimmed, fmt.Errorf("parse assistant message: %w", err)), nil
		}
		return &msg, nil
	case MessageTypeUser:
		var msg UserMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return unknownFromParseFailure(env.Type, trimmed, fmt.Errorf("parse user message: %w", err)), nil
		}
		return &msg, nil
	case MessageTypeResult:
		var msg ResultMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return unknownFromParseFailure(env.Type, trimmed, fmt.Errorf("parse result message: %w", err)), nil
		}
		return &msg, nil
	case MessageTypeStreamEvent:
		var msg StreamEventMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return unknownFromParseFailure(env.Type, trimmed, fmt.Errorf("parse stream event message: %w", err)), nil
		}
		return &msg, nil
	default:
		msg := &UnknownMessage{Type: env.Type}
		msg.setRaw(trimmed)
		return msg, nil
	}
}

func unknownFromParseFailure(messageType MessageType, raw []byte, err error) *UnknownMessage {
	msg := &UnknownMessage{
		Type:       messageType,
		ParseError: err.Error(),
	}
	msg.setRaw(raw)
	return msg
}

func formatParseError(context string, raw []byte, err error) error {
	return fmt.Errorf("%s: %w\nraw:\n%s", context, err, formatRawForError(raw))
}

func formatRawForError(raw []byte) string {
	var indented bytes.Buffer
	if err := json.Indent(&indented, raw, "", "  "); err == nil {
		return indented.String()
	}
	return string(raw)
}

func (p *messageParser) Next() (Message, error) {
	for {
		if ok := p.scanner.Scan(); !ok {
			if err := p.scanner.Err(); err != nil {
				return nil, err
			}
			return nil, clerrors.ErrEOF
		}

		msg, err := p.ParseLine(p.scanner.Bytes())
		if err != nil {
			return nil, err
		}
		if msg == nil {
			continue
		}
		return msg, nil
	}
}
