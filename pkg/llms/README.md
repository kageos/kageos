# LLMs

`pkg/llms` is now a thin application-facing wrapper over `openai-go`.

The package deliberately supports one protocol only: OpenAI Chat Completions. If a
vendor, gateway, or local model service exposes an OpenAI-compatible `/v1`
endpoint, configure it with `api_base` and `model`. If it does not, it is outside
this layer.

## Import

```go
import "github.com/kageos/kageos/pkg/llms"
```

Environment variable:

| Protocol | Environment variable |
| --- | --- |
| OpenAI | `OPENAI_API_KEY` |

## Usage

```go
client := llms.NewOpenAIClientFromEnv()

resp, err := client.Chat(context.Background(), &llms.ChatRequest{
    Messages: []llms.Message{
        {Role: "system", Content: "You are concise."},
        {Role: "user", Content: "Explain goroutines in three sentences."},
    },
    Model:       "gpt-4o-mini",
    MaxTokens:   800,
    Temperature: 0.2,
})
```

For OpenAI-compatible services:

```go
opts := llms.DefaultClientOptions().
    WithBaseURL("https://gateway.example.com/v1").
    WithModel("model-name")

client := llms.NewOpenAIClientWithOptions("<api-key>", opts)
```

## Streaming

```go
stream, err := client.ChatStream(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

for chunk := range stream {
    if chunk.Error != "" {
        log.Fatal(chunk.Error)
    }
    fmt.Print(chunk.Content)
}
```

Streaming includes OpenAI tool-call deltas and final usage when the upstream
endpoint provides them.
