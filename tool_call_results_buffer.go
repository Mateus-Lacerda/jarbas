package main

import (
	"fmt"
	"slices"

	ollama "github.com/ollama/ollama/api"
)

type ToolCallWithResult struct {
	ToolCall ollama.ToolCall
	Result   string
}

type ToolCallResultsBuffer struct {
	b []ToolCallWithResult
}

func NewToolsResultsBuffer() *ToolCallResultsBuffer {
	return &ToolCallResultsBuffer{make([]ToolCallWithResult, 0)}
}

func (t *ToolCallResultsBuffer) Clear() {
	t.b = t.b[:0]
}

func (t *ToolCallResultsBuffer) Grow(s int) {
	t.b = slices.Grow(t.b, s)
}

func (t *ToolCallResultsBuffer) HasToolResults() bool {
	return len(t.b) > 0
}

func (t *ToolCallResultsBuffer) AddResult(toolCall ollama.ToolCall, result string) {
	t.b = append(
		t.b,
		ToolCallWithResult{toolCall, result},
	)
}

func (t *ToolCallResultsBuffer) GetResults() []ToolCallWithResult {
	return t.b
}

func (t *ToolCallResultsBuffer) GetOriginalToolCalls() []ollama.ToolCall {
	if !t.HasToolResults() {
		return nil
	}
	toolCalls := make([]ollama.ToolCall, len(t.b))
	for i, t := range t.GetResults() {
		toolCalls[i] = t.ToolCall
	}
	fmt.Println(toolCalls)
	return toolCalls
}

func (t *ToolCallResultsBuffer) GetResultsAsOllamaMessages() []ollama.Message {
	var messages []ollama.Message
	for _, t := range t.GetResults() {
		messages = append(
			messages,
			ollama.Message{
				Role:       string(ToolRole),
				Content:    t.Result,
				ToolName:   t.ToolCall.Function.Name,
				ToolCallID: t.ToolCall.ID,
			},
		)
	}
	return messages
}
