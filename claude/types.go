package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
)

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
	Raw() []byte
}

type messageRaw struct {
	raw []byte
}

func (m *messageRaw) setRaw(raw []byte) {
	m.raw = bytes.Clone(raw)
}

func (m *messageRaw) Raw() []byte {
	return bytes.Clone(m.raw)
}

type messageEnvelope struct {
	Type MessageType `json:"type"`
}

type SystemMessage struct {
	messageRaw
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
	OutputStyle       string           `json:"output_style,omitempty"`
	Skills            []string         `json:"skills,omitempty"`
	Plugins           []PluginInfo     `json:"plugins,omitempty"`
}

func (m *SystemMessage) GetType() MessageType {
	return m.Type
}

func (m *SystemMessage) UnmarshalJSON(data []byte) error {
	type alias SystemMessage
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = SystemMessage(decoded)
	m.setRaw(data)
	return nil
}

type MCPServerState struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type PluginInfo struct {
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type AssistantMessage struct {
	messageRaw
	Type            MessageType      `json:"type"`
	UUID            string           `json:"uuid,omitempty"`
	SessionID       string           `json:"session_id,omitempty"`
	ParentToolUseID *string          `json:"parent_tool_use_id"`
	Message         AssistantPayload `json:"message"`
	ToolUseResult   *ToolUseResult   `json:"tool_use_result,omitempty"`
}

func (m *AssistantMessage) GetType() MessageType {
	return m.Type
}

func (m *AssistantMessage) UnmarshalJSON(data []byte) error {
	type alias AssistantMessage
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = AssistantMessage(decoded)
	m.setRaw(data)
	return nil
}

type AssistantPayload struct {
	Model             string          `json:"model,omitempty"`
	ID                string          `json:"id,omitempty"`
	Type              string          `json:"type,omitempty"`
	Role              string          `json:"role,omitempty"`
	Content           []ContentBlock  `json:"content,omitempty"`
	StopReason        *string         `json:"stop_reason,omitempty"`
	StopSequence      *string         `json:"stop_sequence,omitempty"`
	Usage             *Usage          `json:"usage,omitempty"`
	ContextManagement json.RawMessage `json:"context_management,omitempty"`
}

type UserMessage struct {
	messageRaw
	Type            MessageType     `json:"type"`
	UUID            string          `json:"uuid,omitempty"`
	SessionID       string          `json:"session_id,omitempty"`
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	Message         UserPayload     `json:"message"`
	ToolUseResult   *ToolUseResult  `json:"tool_use_result,omitempty"`
	Usage           *Usage          `json:"usage,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
}

func (m *UserMessage) GetType() MessageType {
	return m.Type
}

func (m *UserMessage) UnmarshalJSON(data []byte) error {
	type alias UserMessage
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = UserMessage(decoded)
	m.setRaw(data)
	return nil
}

type UserPayload struct {
	Role    string         `json:"role,omitempty"`
	Content []ContentBlock `json:"content,omitempty"`
}

type ToolUseResult struct {
	Filenames   []string `json:"filenames,omitempty"`
	DurationMS  int64    `json:"durationMs,omitempty"`
	NumFiles    int      `json:"numFiles,omitempty"`
	Truncated   bool     `json:"truncated,omitempty"`
	Stdout      string   `json:"stdout,omitempty"`
	Stderr      string   `json:"stderr,omitempty"`
	Interrupted bool     `json:"interrupted,omitempty"`
	IsImage     bool     `json:"isImage,omitempty"`
	Text        string   `json:"-"`
}

func (r *ToolUseResult) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return fmt.Errorf("parse tool_use_result string: %w", err)
		}
		r.Text = text
		return nil
	}

	if trimmed[0] == '{' {
		type alias ToolUseResult
		var decoded alias
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return fmt.Errorf("parse tool_use_result object: %w", err)
		}
		*r = ToolUseResult(decoded)
		return nil
	}

	return fmt.Errorf("unsupported tool_use_result json type: %s", string(trimmed))
}

type ResultMessage struct {
	messageRaw
	Type              MessageType           `json:"type"`
	Subtype           string                `json:"subtype,omitempty"`
	UUID              string                `json:"uuid,omitempty"`
	SessionID         string                `json:"session_id,omitempty"`
	IsError           bool                  `json:"is_error"`
	DurationMS        int64                 `json:"duration_ms,omitempty"`
	DurationAPIMS     int64                 `json:"duration_api_ms,omitempty"`
	NumTurns          int                   `json:"num_turns,omitempty"`
	Result            string                `json:"result,omitempty"`
	StopReason        *string               `json:"stop_reason,omitempty"`
	TotalCostUSD      float64               `json:"total_cost_usd,omitempty"`
	Usage             *Usage                `json:"usage,omitempty"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage,omitempty"`
	PermissionDenials []PermissionDenial    `json:"permission_denials,omitempty"`
	StructuredOutput  json.RawMessage       `json:"structured_output,omitempty"`
	Errors            []string              `json:"errors,omitempty"`
}

func (m *ResultMessage) GetType() MessageType {
	return m.Type
}

func (m *ResultMessage) UnmarshalJSON(data []byte) error {
	type alias ResultMessage
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = ResultMessage(decoded)
	m.setRaw(data)
	return nil
}

type PermissionDenial struct {
	ToolName  string          `json:"tool_name"`
	ToolUseID string          `json:"tool_use_id"`
	ToolInput json.RawMessage `json:"tool_input"`
}

type StreamEventMessage struct {
	messageRaw
	Type            MessageType `json:"type"`
	UUID            string      `json:"uuid,omitempty"`
	SessionID       string      `json:"session_id,omitempty"`
	ParentToolUseID *string     `json:"parent_tool_use_id"`
	Event           StreamEvent `json:"event"`
}

func (m *StreamEventMessage) GetType() MessageType {
	return m.Type
}

func (m *StreamEventMessage) UnmarshalJSON(data []byte) error {
	type alias StreamEventMessage
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = StreamEventMessage(decoded)
	m.setRaw(data)
	return nil
}

type StreamEventType string

const (
	StreamEventTypeMessageStart      StreamEventType = "message_start"
	StreamEventTypeContentBlockStart StreamEventType = "content_block_start"
	StreamEventTypeContentBlockDelta StreamEventType = "content_block_delta"
	StreamEventTypeContentBlockStop  StreamEventType = "content_block_stop"
	StreamEventTypeMessageDelta      StreamEventType = "message_delta"
	StreamEventTypeMessageStop       StreamEventType = "message_stop"
)

type StreamEvent struct {
	Type              StreamEventType         `json:"type"`
	MessageStart      *MessageStartEvent      `json:"-"`
	ContentBlockStart *ContentBlockStartEvent `json:"-"`
	ContentBlockDelta *ContentBlockDeltaEvent `json:"-"`
	ContentBlockStop  *ContentBlockStopEvent  `json:"-"`
	MessageDelta      *MessageDeltaEvent      `json:"-"`
	MessageStop       *MessageStopEvent       `json:"-"`
}

func (e *StreamEvent) UnmarshalJSON(data []byte) error {
	var probe struct {
		Type StreamEventType `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("parse stream event type: %w", err)
	}

	e.Type = probe.Type
	switch probe.Type {
	case StreamEventTypeMessageStart:
		var v MessageStartEvent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		e.MessageStart = &v
	case StreamEventTypeContentBlockStart:
		var v ContentBlockStartEvent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		e.ContentBlockStart = &v
	case StreamEventTypeContentBlockDelta:
		var v ContentBlockDeltaEvent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		e.ContentBlockDelta = &v
	case StreamEventTypeContentBlockStop:
		var v ContentBlockStopEvent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		e.ContentBlockStop = &v
	case StreamEventTypeMessageDelta:
		var v MessageDeltaEvent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		e.MessageDelta = &v
	case StreamEventTypeMessageStop:
		var v MessageStopEvent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		e.MessageStop = &v
	default:
		return fmt.Errorf("unsupported stream event type: %q", probe.Type)
	}
	return nil
}

type MessageStartEvent struct {
	Type    StreamEventType `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`
}

type ContentBlockStartEvent struct {
	Type         StreamEventType `json:"type"`
	Index        int             `json:"index"`
	ContentBlock ContentBlock    `json:"content_block"`
}

type ContentBlockDeltaEvent struct {
	Type  StreamEventType `json:"type"`
	Index int             `json:"index"`
	Delta StreamDelta     `json:"delta"`
}

type ContentBlockStopEvent struct {
	Type  StreamEventType `json:"type"`
	Index int             `json:"index"`
}

type MessageDeltaEvent struct {
	Type  StreamEventType `json:"type"`
	Delta json.RawMessage `json:"delta,omitempty"`
	Usage json.RawMessage `json:"usage,omitempty"`
}

type MessageStopEvent struct {
	Type StreamEventType `json:"type"`
}

type StreamDeltaType string

const (
	StreamDeltaTypeText StreamDeltaType = "text_delta"
)

type StreamDelta struct {
	Type       StreamDeltaType `json:"type,omitempty"`
	Text       string          `json:"text,omitempty"`
	StopReason string          `json:"stop_reason,omitempty"`
}

type ContentBlockType string

const (
	ContentBlockTypeText       ContentBlockType = "text"
	ContentBlockTypeToolUse    ContentBlockType = "tool_use"
	ContentBlockTypeToolResult ContentBlockType = "tool_result"
)

type ContentBlock struct {
	Type       ContentBlockType        `json:"type"`
	Text       *TextContentBlock       `json:"-"`
	ToolUse    *ToolUseContentBlock    `json:"-"`
	ToolResult *ToolResultContentBlock `json:"-"`
}

func (b *ContentBlock) UnmarshalJSON(data []byte) error {
	var probe struct {
		Type ContentBlockType `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("parse content block type: %w", err)
	}

	b.Type = probe.Type
	switch probe.Type {
	case ContentBlockTypeText:
		var v TextContentBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		b.Text = &v
	case ContentBlockTypeToolUse:
		var v ToolUseContentBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		b.ToolUse = &v
	case ContentBlockTypeToolResult:
		var v ToolResultContentBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		b.ToolResult = &v
	default:
		return fmt.Errorf("unsupported content block type: %q", probe.Type)
	}
	return nil
}

type TextContentBlock struct {
	Type ContentBlockType `json:"type"`
	Text string           `json:"text,omitempty"`
}

type ToolUseContentBlock struct {
	Type  ContentBlockType `json:"type"`
	ID    string           `json:"id,omitempty"`
	Name  string           `json:"name,omitempty"`
	Input json.RawMessage  `json:"input,omitempty"`
}

type ToolResultContentBlock struct {
	Type      ContentBlockType `json:"type"`
	ToolUseID string           `json:"tool_use_id,omitempty"`
	Content   json.RawMessage  `json:"content,omitempty"`
}

type Usage struct {
	InputTokens             int64            `json:"input_tokens,omitempty"`
	OutputTokens            int64            `json:"output_tokens,omitempty"`
	CacheCreationInputToken int64            `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens    int64            `json:"cache_read_input_tokens,omitempty"`
	ServerToolUse           map[string]int64 `json:"server_tool_use,omitempty"`
	ServiceTier             string           `json:"service_tier,omitempty"`
	CacheCreation           map[string]int64 `json:"cache_creation,omitempty"`
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
	messageRaw
	Type       MessageType `json:"type,omitempty"`
	ParseError string      `json:"-"`
}

func (m *UnknownMessage) GetType() MessageType {
	return m.Type
}

func (m *UnknownMessage) UnmarshalJSON(data []byte) error {
	type alias UnknownMessage
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = UnknownMessage(decoded)
	m.setRaw(data)
	return nil
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

type UserInputType string

const (
	UserInputTypePrompt     UserInputType = "prompt"
	UserInputTypePermission UserInputType = "permission"
	UserInputTypeRaw        UserInputType = "raw"
	UserInputTypeUser       UserInputType = "user"
)

type PermissionDecision string

const (
	PermissionDecisionAllow PermissionDecision = "allow"
	PermissionDecisionDeny  PermissionDecision = "deny"
)

type PermissionInput struct {
	Decision  PermissionDecision `json:"decision"`
	ToolUseID string             `json:"tool_use_id,omitempty"`
	Reason    string             `json:"reason,omitempty"`
}

type UserInput struct {
	Type       UserInputType     `json:"type,omitempty"`
	UUID       string            `json:"uuid,omitempty"`
	Prompt     string            `json:"prompt,omitempty"`
	Permission *PermissionInput  `json:"permission,omitempty"`
	Raw        string            `json:"raw,omitempty"`
	Message    *UserInputMessage `json:"message,omitempty"`
}

// UserInputMessage matches the 4.4 "User Message" schema.
type UserInputMessage struct {
	Role    string                  `json:"role,omitempty"`
	Content []UserInputContentBlock `json:"content,omitempty"`
}

// UserInputContentBlock represents a tool_result block in a user message.
type UserInputContentBlock struct {
	Type      string `json:"type,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type InitializeParams struct {
	ClientInfo   ClientInfo             `json:"clientInfo,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion,omitempty"`
	ServerInfo      ServerInfo             `json:"serverInfo,omitempty"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
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
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolsCallResult struct {
	Content []ToolResultContent `json:"content,omitempty"`
	IsError bool                `json:"isError,omitempty"`
}

type ToolResultContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
