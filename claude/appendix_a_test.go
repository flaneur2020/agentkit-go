package claude

import (
	"strings"
	"testing"
)

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
}
