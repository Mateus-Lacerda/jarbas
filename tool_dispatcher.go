package main

import (
	"errors"

	ollama "github.com/ollama/ollama/api"
)

type ToolDispatcher struct {
	d map[string](func(ollama.ToolCallFunctionArguments) string)
}

func NewToolDispatcher() *ToolDispatcher {
	return &ToolDispatcher{
		make(
			map[string](func(ollama.ToolCallFunctionArguments) string),
		),
	}
}

func (t *ToolDispatcher) RegisterTool(
	toolName string,
	toolFunc func(ollama.ToolCallFunctionArguments) string,
) {
	t.d[toolName] = toolFunc
}

func (t *ToolDispatcher) DispatchToolCall(
	toolCallFunction ollama.ToolCallFunction,
) (string, error) {
	toolFunc, ok := t.d[toolCallFunction.Name]
	if !ok {
		return "", errors.New("Tool doest not exists or was not registered")
	}
	return toolFunc(toolCallFunction.Arguments), nil
}
