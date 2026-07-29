package relayconvert

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/require"
)

func rawJSON(s string) json.RawMessage {
	return json.RawMessage(s)
}

// deepCopyFixture guards against converters mutating shared fixture state
// between subtests (JSON round-trip through the concrete type).
func deepCopyFixture(t *testing.T, v any) any {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	switch v.(type) {
	case *dto.GeneralOpenAIRequest:
		out := &dto.GeneralOpenAIRequest{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.ClaudeRequest:
		out := &dto.ClaudeRequest{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.GeminiChatRequest:
		out := &dto.GeminiChatRequest{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.OpenAIResponsesRequest:
		out := &dto.OpenAIResponsesRequest{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.OpenAITextResponse:
		out := &dto.OpenAITextResponse{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.ClaudeResponse:
		out := &dto.ClaudeResponse{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.GeminiChatResponse:
		out := &dto.GeminiChatResponse{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.OpenAIResponsesResponse:
		out := &dto.OpenAIResponsesResponse{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.ChatCompletionsStreamResponse:
		out := &dto.ChatCompletionsStreamResponse{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	case *dto.ResponsesStreamResponse:
		out := &dto.ResponsesStreamResponse{}
		require.NoError(t, json.Unmarshal(data, out))
		return out
	default:
		t.Fatalf("deepCopyFixture: unsupported fixture type %T", v)
		return nil
	}
}

func chatStreamChunk(raw string) *dto.ChatCompletionsStreamResponse {
	var r dto.ChatCompletionsStreamResponse
	mustUnmarshalFixture(raw, &r)
	return &r
}

func claudeStreamChunk(raw string) *dto.ClaudeResponse {
	var r dto.ClaudeResponse
	mustUnmarshalFixture(raw, &r)
	return &r
}

func geminiStreamChunk(raw string) *dto.GeminiChatResponse {
	var r dto.GeminiChatResponse
	mustUnmarshalFixture(raw, &r)
	return &r
}

func responsesStreamChunk(raw string) *dto.ResponsesStreamResponse {
	var r dto.ResponsesStreamResponse
	mustUnmarshalFixture(raw, &r)
	return &r
}

func mustUnmarshalFixture(raw string, out any) {
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		panic(fmt.Sprintf("bad fixture JSON: %v", err))
	}
}
