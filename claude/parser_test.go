package claude

import (
	"bytes"
	"strings"
	"testing"

	clerrors "github.com/flaneur2020/agentkit-go/claude/errors"
)

// The parser test suite covers all stream-json message kinds in the spec:
// system / assistant / user / result / stream_event,
// plus unknown type fallback, invalid JSON handling, empty-line skipping, and EOF marker.

func assertRawMessage(t *testing.T, msg Message, line []byte) {
	t.Helper()
	want := bytes.TrimSpace(line)
	got := msg.Raw()
	if !bytes.Equal(got, want) {
		t.Fatalf("Raw() = %s, want %s", got, want)
	}
	if len(got) > 0 {
		got[0] ^= 0xff
		if bytes.Equal(msg.Raw(), got) {
			t.Fatalf("Raw() returned mutable internal buffer")
		}
	}
}

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
	assertRawMessage(t, systemMsg, line)
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
	assertRawMessage(t, unknownMsg, line)
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
	assertRawMessage(t, assistantMsg, line)
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
	assertRawMessage(t, userMsg, line)
}

func TestParserParseLineUserMessageToolUseResultString(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]},"tool_use_result":"command output text"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}

	userMsg, ok := msg.(*UserMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *UserMessage", msg)
	}
	if userMsg.ToolUseResult == nil {
		t.Fatalf("tool_use_result = nil, want value")
	}
	if userMsg.ToolUseResult.Text != "command output text" {
		t.Fatalf("tool_use_result.Text = %q, want command output text", userMsg.ToolUseResult.Text)
	}
	assertRawMessage(t, userMsg, line)
}

func TestParserParseLineTypedParseFailureFallsBackToUnknown(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"user","message":{"role":"user","content":[{"type":"unsupported_block"}]}}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v, want nil", err)
	}

	unknownMsg, ok := msg.(*UnknownMessage)
	if !ok {
		t.Fatalf("ParseLine() type = %T, want *UnknownMessage", msg)
	}
	if unknownMsg.Type != MessageTypeUser {
		t.Fatalf("Type = %q, want user", unknownMsg.Type)
	}
	assertRawMessage(t, unknownMsg, line)
	if !strings.Contains(unknownMsg.ParseError, "parse user message") {
		t.Fatalf("ParseError = %q, want parse user message", unknownMsg.ParseError)
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
	assertRawMessage(t, resultMsg, line)
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
	assertRawMessage(t, eventMsg, line)
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
			assertRawMessage(t, eventMsg, []byte(tc.line))
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
	assertRawMessage(t, assistantMsg, line)
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
	assertRawMessage(t, resultMsg, line)
}

func TestParserParseLineInvalidJSON(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":`)
	_, err := parser.ParseLine(line)
	if err == nil {
		t.Fatalf("ParseLine() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "raw:\n") {
		t.Fatalf("error = %v, want multiline raw content", err)
	}
	if !strings.Contains(err.Error(), string(line)) {
		t.Fatalf("error = %v, want raw line content", err)
	}
}

func TestParserNextSkipsEmptyLines(t *testing.T) {
	line := []byte(`{"type":"result","subtype":"success","is_error":false}`)
	parser := NewMessageParser(strings.NewReader("\n \n" + string(line) + "\n"))
	msg, err := parser.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if msg.GetType() != MessageTypeResult {
		t.Fatalf("type = %s, want result", msg.GetType())
	}
	assertRawMessage(t, msg, line)
}

func TestParserNextEOF(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	_, err := parser.Next()
	if !clerrors.IsEOF(err) {
		t.Fatalf("Next() err = %v, want EOF marker", err)
	}
}
func TestAppendixAGoldenSystemMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"system","subtype":"init","cwd":"/private/tmp/playing","session_id":"5620625c-b4c7-4185-9b2b-8de430dd2184","tools":["Task","TaskOutput","Bash","Glob","Grep","ExitPlanMode","Read","Edit","Write","NotebookEdit","WebFetch","TodoWrite","WebSearch","KillShell","AskUserQuestion","Skill","EnterPlanMode"],"mcp_servers":[{"name":"ruby-tools","status":"connected"}],"model":"claude-sonnet-4-5-20250929","permissionMode":"default","slash_commands":["compact","context","cost","init","pr-comments","release-notes","review","security-review"],"apiKeySource":"ANTHROPIC_API_KEY","claude_code_version":"2.1.3","output_style":"default","agents":["Bash","general-purpose","statusline-setup","Explore","Plan","claude-code-guide"],"skills":[],"plugins":[{"name":"gopls-lsp","path":"/Users/sam/.claude/plugins/cache/claude-plugins-official/gopls-lsp/1.0.0"}],"uuid":"95625b7e-3117-483b-95c9-47e54bb9ec70"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	systemMsg, ok := msg.(*SystemMessage)
	if !ok {
		t.Fatalf("type = %T, want *SystemMessage", msg)
	}
	if systemMsg.OutputStyle != "default" {
		t.Fatalf("output_style = %q, want default", systemMsg.OutputStyle)
	}
	if len(systemMsg.Plugins) != 1 || systemMsg.Plugins[0].Name != "gopls-lsp" {
		t.Fatalf("plugins = %+v, want gopls-lsp", systemMsg.Plugins)
	}
	assertRawMessage(t, systemMsg, line)
}

func TestAppendixAGoldenAssistantMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"assistant","message":{"model":"claude-sonnet-4-5-20250929","id":"msg_01Rf5Yc8FdberfJBxNjTNk3W","type":"message","role":"assistant","content":[{"type":"text","text":"I'll use the ruby-tools MCP server to get the current time and generate a random number."},{"type":"tool_use","id":"toolu_017K5vf","name":"mcp__ruby-tools__current_time","input":{}},{"type":"tool_use","id":"toolu_018ABC","name":"mcp__ruby-tools__random_number","input":{"min":1,"max":100}}],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":2,"cache_creation_input_tokens":4722,"cache_read_input_tokens":13367,"cache_creation":{"ephemeral_5m_input_tokens":4722,"ephemeral_1h_input_tokens":0},"output_tokens":28,"service_tier":"standard"},"context_management":null},"parent_tool_use_id":null,"session_id":"5620625c-b4c7-4185-9b2b-8de430dd2184","uuid":"eda52225-597f-4a1f-8ca6-a6bcd94934ac"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	assistantMsg, ok := msg.(*AssistantMessage)
	if !ok {
		t.Fatalf("type = %T, want *AssistantMessage", msg)
	}
	if len(assistantMsg.Message.Content) != 3 {
		t.Fatalf("content len = %d, want 3", len(assistantMsg.Message.Content))
	}
	if assistantMsg.Message.Content[1].ToolUse == nil || assistantMsg.Message.Content[1].ToolUse.Name != "mcp__ruby-tools__current_time" {
		t.Fatalf("tool_use[1] = %+v, want mcp__ruby-tools__current_time", assistantMsg.Message.Content[1].ToolUse)
	}
	if assistantMsg.Message.Content[2].ToolUse == nil || assistantMsg.Message.Content[2].ToolUse.Name != "mcp__ruby-tools__random_number" {
		t.Fatalf("tool_use[2] = %+v, want mcp__ruby-tools__random_number", assistantMsg.Message.Content[2].ToolUse)
	}
	assertRawMessage(t, assistantMsg, line)
}

func TestAppendixAGoldenUserToolResultMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_01LsrzxpC42FnYPxepJfr9pg","type":"tool_result","content":"/private/tmp/playing/quick_start.rb\n/private/tmp/playing/advanced_examples.rb\n/private/tmp/playing/claude_agent_sdk_demo.rb"}]},"parent_tool_use_id":null,"session_id":"c8775347-af93-45c7-b9bf-a6e009483fa5","uuid":"4ffd3635-d0fb-4057-85e2-0f0a4c302fec","tool_use_result":{"filenames":["/private/tmp/playing/quick_start.rb","/private/tmp/playing/advanced_examples.rb","/private/tmp/playing/claude_agent_sdk_demo.rb"],"durationMs":345,"numFiles":3,"truncated":false}}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	userMsg, ok := msg.(*UserMessage)
	if !ok {
		t.Fatalf("type = %T, want *UserMessage", msg)
	}
	if len(userMsg.Message.Content) != 1 || userMsg.Message.Content[0].ToolResult == nil {
		t.Fatalf("content = %+v, want one tool_result", userMsg.Message.Content)
	}
	if userMsg.ToolUseResult == nil || userMsg.ToolUseResult.NumFiles != 3 {
		t.Fatalf("tool_use_result = %+v, want numFiles=3", userMsg.ToolUseResult)
	}
	assertRawMessage(t, userMsg, line)
}

func TestAppendixAGoldenResultMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"result","subtype":"success","is_error":false,"duration_ms":7040,"duration_api_ms":12311,"num_turns":2,"result":"I found 3 Ruby files in the directory:\n\n1. ` + "`quick_start.rb`" + `\n2. ` + "`advanced_examples.rb`" + `\n3. ` + "`claude_agent_sdk_demo.rb`" + `","session_id":"c8775347-af93-45c7-b9bf-a6e009483fa5","total_cost_usd":0.0186724,"usage":{"input_tokens":7,"cache_creation_input_tokens":440,"cache_read_input_tokens":35858,"output_tokens":114,"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":"standard","cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":440}},"modelUsage":{"claude-haiku-4-5-20251001":{"inputTokens":2,"outputTokens":170,"cacheReadInputTokens":10531,"cacheCreationInputTokens":0,"webSearchRequests":0,"costUSD":0.0019051,"contextWindow":200000,"maxOutputTokens":64000},"claude-sonnet-4-5-20250929":{"inputTokens":9,"outputTokens":143,"cacheReadInputTokens":39900,"cacheCreationInputTokens":439,"webSearchRequests":0,"costUSD":0.0157882,"contextWindow":200000,"maxOutputTokens":64000}},"permission_denials":[],"uuid":"34cf1a3e-8b9f-481e-ba6f-c9e934b09424"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	resultMsg, ok := msg.(*ResultMessage)
	if !ok {
		t.Fatalf("type = %T, want *ResultMessage", msg)
	}
	if resultMsg.Subtype != "success" || resultMsg.IsError {
		t.Fatalf("subtype/is_error = %s/%v, want success/false", resultMsg.Subtype, resultMsg.IsError)
	}
	if len(resultMsg.ModelUsage) != 2 {
		t.Fatalf("modelUsage len = %d, want 2", len(resultMsg.ModelUsage))
	}
	if len(resultMsg.PermissionDenials) != 0 {
		t.Fatalf("permission_denials len = %d, want 0", len(resultMsg.PermissionDenials))
	}
	assertRawMessage(t, resultMsg, line)
}

func TestAppendixAGoldenStreamEventMessage(t *testing.T) {
	parser := NewMessageParser(strings.NewReader(""))
	line := []byte(`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Dogs are loyal"}},"session_id":"4a7c99c6-e08a-4e3c-b6ce-17c33ae8bb92","parent_tool_use_id":null,"uuid":"2bc3e3c8-d9f2-48e8-bd72-b828bbbf3732"}`)

	msg, err := parser.ParseLine(line)
	if err != nil {
		t.Fatalf("ParseLine() error = %v", err)
	}
	eventMsg, ok := msg.(*StreamEventMessage)
	if !ok {
		t.Fatalf("type = %T, want *StreamEventMessage", msg)
	}
	if eventMsg.Event.Type != StreamEventTypeContentBlockDelta {
		t.Fatalf("event type = %s, want content_block_delta", eventMsg.Event.Type)
	}
	if eventMsg.Event.ContentBlockDelta == nil || eventMsg.Event.ContentBlockDelta.Delta.Text != "Dogs are loyal" {
		t.Fatalf("delta = %+v, want Dogs are loyal", eventMsg.Event.ContentBlockDelta)
	}
	assertRawMessage(t, eventMsg, line)
}
