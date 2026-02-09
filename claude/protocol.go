package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
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

type protocol struct {
	parser       MessageParser
	writer       io.Writer
	writerCloser io.Closer

	mu     sync.Mutex
	nextID int64
}

func NewProtocol(r io.Reader, w io.Writer) Protocol {
	p := &protocol{
		parser: NewMessageParser(r),
		writer: w,
		nextID: 1,
	}
	if closer, ok := w.(io.Closer); ok {
		p.writerCloser = closer
	}
	return p
}

func (p *protocol) SendUserInput(ctx context.Context, input UserInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if input.Prompt == "" {
		return fmt.Errorf("prompt is empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, err := io.WriteString(p.writer, input.Prompt); err != nil {
		return fmt.Errorf("write user input: %w", err)
	}
	return nil
}

func (p *protocol) NextMessage(ctx context.Context) (Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p.parser.Next()
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
	if p.writerCloser == nil {
		return nil
	}
	return p.writerCloser.Close()
}

func (p *protocol) request(ctx context.Context, method string, params interface{}, out interface{}) error {
	id, err := p.writeJSONRPCRequest(ctx, method, params)
	if err != nil {
		return err
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		msg, err := p.parser.Next()
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

	p.mu.Lock()
	defer p.mu.Unlock()

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

	p.mu.Lock()
	defer p.mu.Unlock()

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
