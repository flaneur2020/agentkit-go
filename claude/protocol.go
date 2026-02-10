package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	agenterrors "agentkit/errors"
)

type StreamAPI interface {
	SendUserInput(ctx context.Context, input UserInput) error
	NextMessage(ctx context.Context) (Message, error)
}

type MCPAPI interface {
	MCPInitialize(ctx context.Context, params InitializeParams) (*InitializeResult, error)
	MCPInitialized(ctx context.Context) error
	MCPToolsList(ctx context.Context) (*ToolsListResult, error)
	MCPToolsCall(ctx context.Context, params ToolsCallParams) (*ToolsCallResult, error)
}

type Protocol interface {
	StreamAPI
	MCPAPI
	io.Closer
}

type parsedItem struct {
	msg Message
	err error
}

type protocol struct {
	parser       MessageParser
	writer       io.Writer
	writerCloser io.Closer
	readerCloser io.Closer

	writeMu sync.Mutex
	nextID  int64

	readCh chan parsedItem
}

func NewProtocol(r io.Reader, w io.Writer) Protocol {
	p := &protocol{
		parser: NewMessageParser(r),
		writer: w,
		nextID: 1,
		readCh: make(chan parsedItem, 128),
	}
	if closer, ok := w.(io.Closer); ok {
		p.writerCloser = closer
	}
	if closer, ok := r.(io.Closer); ok {
		p.readerCloser = closer
	}
	go p.runParser()
	return p
}

func (p *protocol) runParser() {
	for {
		msg, err := p.parser.Next()
		if err != nil {
			p.readCh <- parsedItem{err: err}
			close(p.readCh)
			return
		}
		p.readCh <- parsedItem{msg: msg}
	}
}

func (p *protocol) SendUserInput(ctx context.Context, input UserInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := inputPayload(input)
	if err != nil {
		return err
	}

	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	if _, err := io.WriteString(p.writer, payload); err != nil {
		return fmt.Errorf("write user input: %w", err)
	}
	return nil
}

func (p *protocol) NextMessage(ctx context.Context) (Message, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case item, ok := <-p.readCh:
			if !ok {
				return nil, agenterrors.ErrEOF
			}
			if item.err != nil {
				return nil, item.err
			}
			if item.msg == nil {
				continue
			}
			return item.msg, nil
		}
	}
}

func (p *protocol) MCPInitialize(ctx context.Context, params InitializeParams) (*InitializeResult, error) {
	var out InitializeResult
	if err := p.request(ctx, "initialize", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *protocol) MCPInitialized(ctx context.Context) error {
	return p.notify(ctx, "initialized", struct{}{})
}

func (p *protocol) MCPToolsList(ctx context.Context) (*ToolsListResult, error) {
	var out ToolsListResult
	if err := p.request(ctx, "tools/list", struct{}{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *protocol) MCPToolsCall(ctx context.Context, params ToolsCallParams) (*ToolsCallResult, error) {
	var out ToolsCallResult
	if err := p.request(ctx, "tools/call", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *protocol) Close() error {
	var firstErr error
	if p.writerCloser != nil {
		if err := p.writerCloser.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if p.readerCloser != nil {
		if err := p.readerCloser.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (p *protocol) request(ctx context.Context, method string, params interface{}, out interface{}) error {
	id, err := p.writeJSONRPCRequest(ctx, method, params)
	if err != nil {
		return err
	}

	for {
		msg, err := p.NextMessage(ctx)
		if err != nil {
			return err
		}

		unknown, ok := msg.(*UnknownMessage)
		if !ok {
			continue
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(unknown.Raw, &resp); err != nil {
			continue
		}
		if resp.JSONRPC != "2.0" || resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("jsonrpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		if out == nil || len(resp.Result) == 0 {
			return nil
		}
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("decode %s result: %w", method, err)
		}
		return nil
	}
}

func (p *protocol) notify(ctx context.Context, method string, params interface{}) error {
	_, err := p.writeJSONRPCNotification(ctx, method, params)
	return err
}

func (p *protocol) writeJSONRPCRequest(ctx context.Context, method string, params interface{}) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	id := p.nextID
	p.nextID++

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}
	if err := json.NewEncoder(p.writer).Encode(req); err != nil {
		return 0, fmt.Errorf("write jsonrpc request: %w", err)
	}
	return id, nil
}

func (p *protocol) writeJSONRPCNotification(ctx context.Context, method string, params interface{}) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	if err := json.NewEncoder(p.writer).Encode(req); err != nil {
		return 0, fmt.Errorf("write jsonrpc notification: %w", err)
	}
	return 0, nil
}

func inputPayload(input UserInput) (string, error) {
	inputType := input.Type
	if inputType == "" {
		if input.Raw != "" {
			inputType = UserInputTypeRaw
		} else if input.Permission != nil {
			inputType = UserInputTypePermission
		} else {
			inputType = UserInputTypePrompt
		}
	}

	switch inputType {
	case UserInputTypePrompt:
		if strings.TrimSpace(input.Prompt) == "" {
			return "", fmt.Errorf("prompt is empty")
		}
		return input.Prompt, nil
	case UserInputTypeRaw:
		if input.Raw == "" {
			return "", fmt.Errorf("raw input is empty")
		}
		return input.Raw, nil
	case UserInputTypePermission:
		if input.Permission == nil {
			return "", fmt.Errorf("permission input is nil")
		}
		if input.Permission.Decision != PermissionDecisionAllow && input.Permission.Decision != PermissionDecisionDeny {
			return "", fmt.Errorf("unsupported permission decision: %s", input.Permission.Decision)
		}
		payload, err := json.Marshal(input.Permission)
		if err != nil {
			return "", fmt.Errorf("marshal permission input: %w", err)
		}
		return string(payload), nil
	default:
		return "", fmt.Errorf("unsupported user input type: %s", input.Type)
	}
}
