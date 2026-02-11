package claude

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Client struct {
	cmd      *exec.Cmd
	protocol Protocol
	stdin    io.WriteCloser
	stdout   io.ReadCloser
}

func (c *Client) SendUserInput(ctx context.Context, input UserInput) error {
	return c.protocol.SendUserInput(ctx, input)
}

func (c *Client) NextMessage(ctx context.Context) (Message, error) {
	return c.protocol.NextMessage(ctx)
}

func (c *Client) MCPInitialize(ctx context.Context, params InitializeParams) (*InitializeResult, error) {
	return c.protocol.MCPInitialize(ctx, params)
}

func (c *Client) MCPInitialized(ctx context.Context) error {
	return c.protocol.MCPInitialized(ctx)
}

func (c *Client) MCPToolsList(ctx context.Context) (*ToolsListResult, error) {
	return c.protocol.MCPToolsList(ctx)
}

func (c *Client) MCPToolsCall(ctx context.Context, params ToolsCallParams) (*ToolsCallResult, error) {
	return c.protocol.MCPToolsCall(ctx, params)
}

func (c *Client) Close() error {
	var firstErr error
	if c.protocol != nil {
		if err := c.protocol.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.stdout != nil {
		if err := c.stdout.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := c.cmd.Wait(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *Client) Wait() error {
	if c.cmd == nil {
		return nil
	}
	return c.cmd.Wait()
}

func (c *Client) Process() *os.Process {
	if c.cmd == nil {
		return nil
	}
	return c.cmd.Process
}

type ClientBuilder struct {
	binary                     string
	model                      string
	maxTurns                   *int
	maxBudgetUSD               *float64
	systemPrompt               string
	appendSystemPrompt         string
	allowedTools               []string
	disallowedTools            []string
	mcpConfigPath              string
	includePartialMessages     bool
	dangerouslySkipPermissions bool
	resumeSessionID            string
	continueSession            bool
	permissionMode             string
	cwd                        string
	env                        map[string]string
	writer                     io.Writer
	reader                     io.Reader
	stderr                     io.Writer
	commandFactory             func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewClientBuilder() *ClientBuilder {
	return &ClientBuilder{
		binary:         "claude",
		env:            map[string]string{},
		commandFactory: exec.CommandContext,
	}
}

func (b *ClientBuilder) WithBinary(path string) *ClientBuilder {
	b.binary = strings.TrimSpace(path)
	return b
}

func (b *ClientBuilder) WithModel(model string) *ClientBuilder {
	b.model = strings.TrimSpace(model)
	return b
}

func (b *ClientBuilder) WithMaxTurns(n int) *ClientBuilder {
	b.maxTurns = &n
	return b
}

func (b *ClientBuilder) WithMaxBudgetUSD(v float64) *ClientBuilder {
	b.maxBudgetUSD = &v
	return b
}

func (b *ClientBuilder) WithSystemPrompt(v string) *ClientBuilder {
	b.systemPrompt = v
	return b
}

func (b *ClientBuilder) WithAppendSystemPrompt(v string) *ClientBuilder {
	b.appendSystemPrompt = v
	return b
}

func (b *ClientBuilder) WithAllowedTools(tools ...string) *ClientBuilder {
	b.allowedTools = append([]string(nil), tools...)
	return b
}

func (b *ClientBuilder) WithDisallowedTools(tools ...string) *ClientBuilder {
	b.disallowedTools = append([]string(nil), tools...)
	return b
}

func (b *ClientBuilder) WithMCPConfig(path string) *ClientBuilder {
	b.mcpConfigPath = strings.TrimSpace(path)
	return b
}

func (b *ClientBuilder) WithIncludePartialMessages(enabled bool) *ClientBuilder {
	b.includePartialMessages = enabled
	return b
}

func (b *ClientBuilder) WithDangerouslySkipPermissions(enabled bool) *ClientBuilder {
	b.dangerouslySkipPermissions = enabled
	return b
}

func (b *ClientBuilder) WithResume(sessionID string) *ClientBuilder {
	b.resumeSessionID = strings.TrimSpace(sessionID)
	return b
}

func (b *ClientBuilder) WithContinue(enabled bool) *ClientBuilder {
	b.continueSession = enabled
	return b
}

func (b *ClientBuilder) WithPermissionMode(mode string) *ClientBuilder {
	b.permissionMode = strings.TrimSpace(mode)
	return b
}

func (b *ClientBuilder) WithCwd(cwd string) *ClientBuilder {
	b.cwd = strings.TrimSpace(cwd)
	return b
}

func (b *ClientBuilder) WithEnv(key, value string) *ClientBuilder {
	if b.env == nil {
		b.env = map[string]string{}
	}
	b.env[key] = value
	return b
}

func (b *ClientBuilder) WithStderr(w io.Writer) *ClientBuilder {
	b.stderr = w
	return b
}

func (b *ClientBuilder) WithReader(r io.Reader) *ClientBuilder {
	b.reader = r
	return b
}

func (b *ClientBuilder) WithWriter(w io.Writer) *ClientBuilder {
	b.writer = w
	return b
}

func (b *ClientBuilder) Build(ctx context.Context) (*Client, error) {
	hasReader := b.reader != nil
	hasWriter := b.writer != nil
	if hasReader || hasWriter {
		if !hasReader || !hasWriter {
			return nil, fmt.Errorf("withReadWriter requires both stdin and stdout")
		}

		p := NewProtocol(b.reader, b.writer)
		client := &Client{protocol: p}
		if stdin, ok := b.writer.(io.WriteCloser); ok {
			client.stdin = stdin
		}
		if stdout, ok := b.reader.(io.ReadCloser); ok {
			client.stdout = stdout
		}
		return client, nil
	}

	if strings.TrimSpace(b.binary) == "" {
		return nil, fmt.Errorf("binary is empty")
	}

	args := b.buildArgs()
	cmd := b.commandFactory(ctx, b.binary, args...)
	if b.cwd != "" {
		cmd.Dir = b.cwd
	}
	if b.stderr != nil {
		cmd.Stderr = b.stderr
	}

	if len(b.env) > 0 {
		env := os.Environ()
		for key, value := range b.env {
			env = append(env, key+"="+value)
		}
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("start %s: %w", b.binary, err)
	}

	p := NewProtocol(stdout, stdin)
	return &Client{
		cmd:      cmd,
		protocol: p,
		stdin:    stdin,
		stdout:   stdout,
	}, nil
}

func (b *ClientBuilder) buildArgs() []string {
	args := []string{"--print", "--output-format", "stream-json", "--verbose"}

	if b.model != "" {
		args = append(args, "--model", b.model)
	}
	if b.maxTurns != nil {
		args = append(args, "--max-turns", strconv.Itoa(*b.maxTurns))
	}
	if b.maxBudgetUSD != nil {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(*b.maxBudgetUSD, 'f', -1, 64))
	}
	if b.systemPrompt != "" {
		args = append(args, "--system-prompt", b.systemPrompt)
	}
	if b.appendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", b.appendSystemPrompt)
	}
	if len(b.allowedTools) > 0 {
		args = append(args, "--allowed-tools", strings.Join(b.allowedTools, ","))
	}
	if len(b.disallowedTools) > 0 {
		args = append(args, "--disallowed-tools", strings.Join(b.disallowedTools, ","))
	}
	if b.mcpConfigPath != "" {
		args = append(args, "--mcp-config", b.mcpConfigPath)
	}
	if b.includePartialMessages {
		args = append(args, "--include-partial-messages")
	}
	if b.dangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if b.resumeSessionID != "" {
		args = append(args, "--resume", b.resumeSessionID)
	}
	if b.continueSession {
		args = append(args, "--continue")
	}
	if b.permissionMode != "" {
		args = append(args, "--permission-mode", b.permissionMode)
	}

	return args
}
