package relayconvert

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func terminalTestGeminiChunk(text string, finishReason string, toolCall bool) *dto.GeminiChatResponse {
	response := terminalTestGeminiChunkWithoutUsage(text, finishReason)
	response.HasUsageMetadata = true
	response.UsageMetadata = dto.GeminiUsageMetadata{
		PromptTokenCount:     4,
		CandidatesTokenCount: 2,
		TotalTokenCount:      6,
	}
	if toolCall {
		response.Candidates[0].Content.Parts = []dto.GeminiPart{
			{
				FunctionCall: &dto.FunctionCall{
					FunctionName: "lookup",
					Arguments:    map[string]any{"q": "x"},
				},
			},
		}
	}
	return response
}

func terminalTestGeminiChunkWithoutUsage(text string, finishReason string) *dto.GeminiChatResponse {
	candidate := dto.GeminiChatCandidate{
		Content: dto.GeminiChatContent{
			Role:  "model",
			Parts: []dto.GeminiPart{{Text: text}},
		},
	}
	if finishReason != "" {
		candidate.FinishReason = terminalTestPtr(finishReason)
	}
	return &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{candidate},
	}
}

func terminalTestGeminiToolChunkWithoutUsage() *dto.GeminiChatResponse {
	return &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{
			{
				Content: dto.GeminiChatContent{
					Role: "model",
					Parts: []dto.GeminiPart{
						{
							FunctionCall: &dto.FunctionCall{
								FunctionName: "lookup",
								Arguments:    map[string]any{"q": "x"},
							},
						},
					},
				},
			},
		},
	}
}

func terminalTestGeminiUsageOnlyChunk() *dto.GeminiChatResponse {
	return &dto.GeminiChatResponse{
		HasUsageMetadata: true,
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     4,
			CandidatesTokenCount: 2,
			TotalTokenCount:      6,
		},
	}
}

func terminalTestFinishedChatChunks(t *testing.T, results []ResponseResult) []*dto.ChatCompletionsStreamResponse {
	t.Helper()
	finished := make([]*dto.ChatCompletionsStreamResponse, 0, 1)
	for _, result := range results {
		response, ok := result.Value.(*dto.ChatCompletionsStreamResponse)
		require.True(t, ok, "unexpected stream result type %T", result.Value)
		if response.IsFinished() {
			finished = append(finished, response)
		}
	}
	return finished
}

func terminalTestAssertClaudeTail(t *testing.T, results []ResponseResult, wantStopReason string) {
	t.Helper()
	responses := make([]*dto.ClaudeResponse, 0, len(results))
	eventCounts := make(map[string]int)
	for _, result := range results {
		response, ok := result.Value.(*dto.ClaudeResponse)
		require.True(t, ok, "unexpected stream result type %T", result.Value)
		responses = append(responses, response)
		eventCounts[response.Type]++
	}

	require.GreaterOrEqual(t, len(responses), 4)
	assert.Equal(t, "message_start", responses[0].Type)
	tail := responses[len(responses)-3:]
	assert.Equal(t, "content_block_stop", tail[0].Type)
	require.NotNil(t, tail[0].Index)
	assert.Equal(t, 0, *tail[0].Index)
	assert.Equal(t, "message_delta", tail[1].Type)
	require.NotNil(t, tail[1].Delta)
	require.NotNil(t, tail[1].Delta.StopReason)
	assert.Equal(t, wantStopReason, *tail[1].Delta.StopReason)
	require.NotNil(t, tail[1].Usage)
	assert.Equal(t, 4, tail[1].Usage.InputTokens)
	assert.Equal(t, 2, tail[1].Usage.OutputTokens)
	assert.Equal(t, "message_stop", tail[2].Type)
	assert.Equal(t, 1, eventCounts["content_block_stop"])
	assert.Equal(t, 1, eventCounts["message_delta"])
	assert.Equal(t, 1, eventCounts["message_stop"])
}

func terminalTestPtr[T any](value T) *T {
	return &value
}
