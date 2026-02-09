package claude

import "encoding/json"

type MessageType string

const (
	MessageTypeSystem      MessageType = "system"
	MessageTypeAssistant   MessageType = "assistant"
	MessageTypeUser        MessageType = "user"
	MessageTypeResult      MessageType = "result"
	MessageTypeStreamEvent MessageType = "stream_event"
)

type Message interface {
	GetType() MessageType
}

type messageEnvelope struct {
	Type MessageType `json:"type"`
}

type SystemMessage struct {
	Type              MessageType      `json:"type"`
	Subtype           string           `json:"subtype,omitempty"`
	UUID              string           `json:"uuid,omitempty"`
	SessionID         string           `json:"session_id,omitempty"`
	CWD               string           `json:"cwd,omitempty"`
	Model             string           `json:"model,omitempty"`
	Tools             []string         `json:"tools,omitempty"`
	MCPServers        []MCPServerState `json:"mcp_servers,omitempty"`
	PermissionMode    string           `json:"permissionMode,omitempty"`
	APIKeySource      string           `json:"apiKeySource,omitempty"`
	SlashCommands     []string         `json:"slash_commands,omitempty"`
	Agents            []string         `json:"agents,omitempty"`
	ClaudeCodeVersion string           `json:"claude_code_version,omitempty"`
}

func (m *SystemMessage) GetType() MessageType {
	return m.Type
}

type MCPServerState struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AssistantMessage struct {
	Type            MessageType       `json:"type"`
	UUID            string            `json:"uuid,omitempty"`
	SessionID       string            `json:"session_id,omitempty"`
	ParentToolUseID *string           `json:"parent_tool_use_id"`
	Message         AssistantPayload  `json:"message"`
	ToolUseResult   json.RawMessage   `json:"tool_use_result,omitempty"`
	Extra           map[string]string `json:"-"`
}

func (m *AssistantMessage) GetType() MessageType {
	return m.Type
}

type AssistantPayload struct {
	Model        string                `json:"model,omitempty"`
	ID           string                `json:"id,omitempty"`
	Type         string                `json:"type,omitempty"`
	Role         string                `json:"role,omitempty"`
	Content      []MessageContentBlock `json:"content,omitempty"`
	StopReason   *string               `json:"stop_reason,omitempty"`
	StopSequence *string               `json:"stop_sequence,omitempty"`
	Usage        *Usage                `json:"usage,omitempty"`
}

type UserMessage struct {
	Type            MessageType      `json:"type"`
	UUID            string           `json:"uuid,omitempty"`
	SessionID       string           `json:"session_id,omitempty"`
	ParentToolUseID *string          `json:"parent_tool_use_id"`
	Message         UserPayload      `json:"message"`
	ToolUseResult   *ToolUseResult   `json:"tool_use_result,omitempty"`
	Usage           *Usage           `json:"usage,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
}

func (m *UserMessage) GetType() MessageType {
	return m.Type
}

type UserPayload struct {
	Role    string                `json:"role,omitempty"`
	Content []MessageContentBlock `json:"content,omitempty"`
}

type ToolUseResult struct {
	Filenames  []string `json:"filenames,omitempty"`
	DurationMS int64    `json:"durationMs,omitempty"`
	NumFiles   int      `json:"numFiles,omitempty"`
	Truncated  bool     `json:"truncated,omitempty"`
}

type ResultMessage struct {
	Type              MessageType                    `json:"type"`
	Subtype           string                         `json:"subtype,omitempty"`
	UUID              string                         `json:"uuid,omitempty"`
	SessionID         string                         `json:"session_id,omitempty"`
	IsError           bool                           `json:"is_error"`
	DurationMS        int64                          `json:"duration_ms,omitempty"`
	DurationAPIMS     int64                          `json:"duration_api_ms,omitempty"`
	NumTurns          int                            `json:"num_turns,omitempty"`
	Result            string                         `json:"result,omitempty"`
	TotalCostUSD      float64                        `json:"total_cost_usd,omitempty"`
	Usage             *Usage                         `json:"usage,omitempty"`
	ModelUsage        map[string]ModelUsage          `json:"modelUsage,omitempty"`
	PermissionDenials []PermissionDenial             `json:"permission_denials,omitempty"`
	StructuredOutput  json.RawMessage                `json:"structured_output,omitempty"`
	Errors            []string                       `json:"errors,omitempty"`
}

func (m *ResultMessage) GetType() MessageType {
	return m.Type
}

type PermissionDenial struct {
	ToolName  string          `json:"tool_name"`
	ToolUseID string          `json:"tool_use_id"`
	ToolInput json.RawMessage `json:"tool_input"`
}

type StreamEventMessage struct {
	Type            MessageType  `json:"type"`
	UUID            string       `json:"uuid,omitempty"`
	SessionID       string       `json:"session_id,omitempty"`
	ParentToolUseID *string      `json:"parent_tool_use_id"`
	Event           StreamEvent  `json:"event"`
}

func (m *StreamEventMessage) GetType() MessageType {
	return m.Type
}

type StreamEvent struct {
	Type         string          `json:"type"`
	Index        *int            `json:"index,omitempty"`
	Delta        *StreamDelta    `json:"delta,omitempty"`
	Message      json.RawMessage `json:"message,omitempty"`
	ContentBlock json.RawMessage `json:"content_block,omitempty"`
}

type StreamDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

type MessageContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

type Usage struct {
	InputTokens             int64                   `json:"input_tokens,omitempty"`
	OutputTokens            int64                   `json:"output_tokens,omitempty"`
	CacheCreationInputToken int64                   `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens    int64                   `json:"cache_read_input_tokens,omitempty"`
	ServerToolUse           map[string]int64        `json:"server_tool_use,omitempty"`
	ServiceTier             string                  `json:"service_tier,omitempty"`
	CacheCreation           map[string]int64        `json:"cache_creation,omitempty"`
}

type ModelUsage struct {
	InputTokens             int64   `json:"inputTokens,omitempty"`
	OutputTokens            int64   `json:"outputTokens,omitempty"`
	CacheReadInputTokens    int64   `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputToken int64   `json:"cacheCreationInputTokens,omitempty"`
	WebSearchRequests       int64   `json:"webSearchRequests,omitempty"`
	CostUSD                 float64 `json:"costUSD,omitempty"`
	ContextWindow           int64   `json:"contextWindow,omitempty"`
	MaxOutputTokens         int64   `json:"maxOutputTokens,omitempty"`
}

type UnknownMessage struct {
	Type MessageType     `json:"type,omitempty"`
	Raw  json.RawMessage `json:"-"`
}

func (m *UnknownMessage) GetType() MessageType {
	return m.Type
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64      `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type UserInput struct {
	Prompt string
}

type InitializeParams struct {
	ClientInfo   ClientInfo          `json:"clientInfo,omitempty"`
	Capabilities map[string]any      `json:"capabilities,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion,omitempty"`
	ServerInfo      ServerInfo             `json:"serverInfo,omitempty"`
	Capabilities    map[string]any         `json:"capabilities,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools,omitempty"`
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

type ToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type ToolsCallResult struct {
	Content []ToolResultContent `json:"content,omitempty"`
	IsError bool                `json:"isError,omitempty"`
}

type ToolResultContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
