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
		return nil, fmt.Errorf("parse message envelope: %w", err)
	}

	switch env.Type {
	case MessageTypeSystem:
		var msg SystemMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return nil, fmt.Errorf("parse system message: %w", err)
		}
		return &msg, nil
	case MessageTypeAssistant:
		var msg AssistantMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return nil, fmt.Errorf("parse assistant message: %w", err)
		}
		return &msg, nil
	case MessageTypeUser:
		var msg UserMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return nil, fmt.Errorf("parse user message: %w", err)
		}
		return &msg, nil
	case MessageTypeResult:
		var msg ResultMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return nil, fmt.Errorf("parse result message: %w", err)
		}
		return &msg, nil
	case MessageTypeStreamEvent:
		var msg StreamEventMessage
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return nil, fmt.Errorf("parse stream event message: %w", err)
		}
		return &msg, nil
	default:
		return &UnknownMessage{Type: env.Type, Raw: append([]byte(nil), trimmed...)}, nil
	}
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
