# RFC 001 - Claude Protocol & Client Interface Design

- 状态：Draft（待确认）
- 日期：2026-02-09
- 目标：先定义稳定接口，再按接口实现 `claude/types.go`、`claude/parser.go`、`claude/protocol.go`、`claude/client.go` 与测试。

## 1. 背景与范围

根据 `docs/CLAUDE_AGENT_SDK_SPEC.md`：

1. Claude CLI 通过 `--output-format stream-json` 输出 NDJSON（`system` / `assistant` / `user` / `result` / `stream_event`）。
2. MCP 协议使用 JSON-RPC 2.0（`initialize` / `initialized` / `tools/list` / `tools/call`）。

本 RFC 采用“同一底层 stdio 连接，兼容消息流读取 + JSON-RPC 请求”的抽象；`Client` 对外实现 `Protocol` 接口，内部做简单转发。

补充约束（根据确认）：

- 用户输入（prompt）归属 `Protocol` 层，不放在 `ClientBuilder`。
- `Protocol` 体现 chat 交互入口，`ClientBuilder` 只负责进程与运行参数构造。

## 2. 文件与职责

### `claude/types.go`

- 定义 CLI stream-json 相关类型（Message family）
- 定义 JSON-RPC 2.0 通用类型
- 定义 MCP 方法参数/结果类型

### `claude/parser.go`

- 提供 NDJSON 行级解析器
- 将每一行动态解析为强类型 Message（按 `type` 字段分派）
- 保留 unknown 类型兜底（向前兼容）

### `claude/protocol.go`

- 构造函数接收 `io.Reader` / `io.Writer`
- 对外仅暴露 interface（符合你的要求）
- 提供：
  - chat 输入方法（写入用户输入）
  - 流式读取 Claude CLI 消息
  - MCP 具体操作方法（非通用 request/notify）

### `claude/client.go`

- `ClientBuilder`（Builder 模式）配置 claude 启动参数
- 启动 `claude` 进程，基于 `cmd.StdoutPipe` + `cmd.StdinPipe` 构建 `Protocol`
- `Client` 实现 `Protocol`（对内部 protocol 成员简单转发）

## 3. 类型草案（`types.go`）

### 3.1 Message 基础

```go
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
```

### 3.2 主要消息结构

- `SystemMessage`
- `AssistantMessage`
- `UserMessage`
- `ResultMessage`
- `StreamEventMessage`
- `UnknownMessage`（包含 `Raw json.RawMessage`）

### 3.3 JSON-RPC / MCP 类型

```go
type JSONRPCRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int64       `json:"id,omitempty"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int64           `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *JSONRPCError   `json:"error,omitempty"`
}

// MCP typed payloads
// InitializeParams / InitializeResult / Tool / ToolsListResult / ToolsCallParams / ToolsCallResult ...
```

> 说明：ID 暂定 `int64`，避免 `interface{}` 带来的比较复杂度；若你希望兼容 string id，我会改为 `json.RawMessage` 或 `any` 并处理匹配逻辑。

## 4. 解析器草案（`parser.go`）

```go
type MessageParser interface {
    ParseLine(line []byte) (Message, error)
    Next() (Message, error)
}

func NewMessageParser(r io.Reader) MessageParser
```

行为约定：

- `Next()` 逐行读取（scanner）
- 空行跳过
- 非法 JSON 返回错误
- 未知 `type` 返回 `*UnknownMessage`（不报错）
- EOF 返回 `io.EOF`

## 5. 协议接口草案（`protocol.go`）

> 重点：本文件所有公共方法以 interface 暴露。

```go
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

func NewProtocol(r io.Reader, w io.Writer) Protocol
```

实现策略（内部，不导出）：

- 使用 `MessageParser` 处理 stream-json 行
- 内部保留 `request(...)` / `notify(...)` helper（不导出）
- `MCPInitialize/MCPToolsList/MCPToolsCall` 通过内部 `request(...)` 实现
- `MCPInitialized` 通过内部 `notify(...)` 实现
- 简化并发模型：**文档中声明非并发安全**（调用方串行）

操作清单与语义：

- `SendUserInput`：发送用户输入（一次 turn 的起点）
- `NextMessage`：读取 Claude CLI `stream-json` 消息流
- `MCPInitialize`：发送 `initialize` 请求并返回服务端能力
- `MCPInitialized`：发送 `initialized` 通知（无返回）
- `MCPToolsList`：发送 `tools/list` 请求并返回工具清单
- `MCPToolsCall`：发送 `tools/call` 请求并返回工具执行结果

`UserInput` 草案：

```go
type UserInput struct {
    Prompt    string
    Resume    string // optional session id
    Continue  bool
}
```

交互约定：

- `SendUserInput` 后，调用方持续 `NextMessage`，直到读到 `ResultMessage` 表示该次输入完成。
- 并发模型仍为串行（一次仅处理一个 turn）。

## 6. ClientBuilder 草案（`client.go`）

```go
type ClientBuilder struct {
    // command config
}

func NewClientBuilder() *ClientBuilder
func (b *ClientBuilder) WithBinary(path string) *ClientBuilder
func (b *ClientBuilder) WithModel(model string) *ClientBuilder
func (b *ClientBuilder) WithMaxTurns(n int) *ClientBuilder
func (b *ClientBuilder) WithMaxBudgetUSD(v float64) *ClientBuilder
func (b *ClientBuilder) WithSystemPrompt(v string) *ClientBuilder
func (b *ClientBuilder) WithAppendSystemPrompt(v string) *ClientBuilder
func (b *ClientBuilder) WithAllowedTools(tools ...string) *ClientBuilder
func (b *ClientBuilder) WithDisallowedTools(tools ...string) *ClientBuilder
func (b *ClientBuilder) WithMCPConfig(path string) *ClientBuilder
func (b *ClientBuilder) WithIncludePartialMessages(enabled bool) *ClientBuilder
func (b *ClientBuilder) WithDangerouslySkipPermissions(enabled bool) *ClientBuilder
func (b *ClientBuilder) WithResume(sessionID string) *ClientBuilder
func (b *ClientBuilder) WithContinue(enabled bool) *ClientBuilder
func (b *ClientBuilder) WithCwd(cwd string) *ClientBuilder
func (b *ClientBuilder) WithEnv(key, value string) *ClientBuilder

func (b *ClientBuilder) Build(ctx context.Context) (*Client, error)
```

`Build` 生成基础命令：

```bash
claude --print --output-format stream-json --verbose [OPTIONS]
```

当用户调用 `SendUserInput` 时，由协议层将输入写入进程标准输入。

`Client` 结构：

- 持有 `*exec.Cmd`
- 持有内部 `Protocol`
- 实现 `Protocol`（转发）
- 可额外提供：`Wait() error` / `Process() *os.Process`

## 7. 测试草案（先测后写）

拟新增 `claude/client_test.go`（可拆分 parser/protocol/client 子测试）：

1. `TestParser_ParseLine_SystemMessage`
2. `TestParser_ParseLine_UnknownType`
3. `TestProtocol_InitializeAndInitialized`
4. `TestProtocol_MCPHelpers`
5. `TestClientBuilder_CommandArgs`
6. `TestClient_ForwardsProtocolMethods`
7. `TestProtocol_SendUserInput`

说明：

- protocol 测试使用 `bytes.Buffer`/`io.Pipe` 模拟双向 stdio
- client 测试通过注入命令（如 `sh -c`）模拟 claude 输出，避免依赖本机安装

## 8. 待你确认的点

1. `Protocol` 是否保留 `NextMessage(ctx)`（把 stream-json 读取纳入同一接口）？
2. JSON-RPC `id` 是否坚持 `int64`，还是需要兼容 string id？
3. `ClientBuilder` 的链式参数集合是否需要再精简（例如仅保留你当前会用到的 flags）？
4. 你是否接受我把测试拆为 `parser_test.go` + `protocol_test.go` + `client_test.go`（更清晰）？
