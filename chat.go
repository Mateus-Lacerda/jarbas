package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/Mateus-Lacerda/better-mem/pkg/core"
	"github.com/Mateus-Lacerda/better-mem/sdk/better-mem-go"
	ollama "github.com/ollama/ollama/api"
)

// TODO: Implement real tools
// TODO: Fix some errors on better mem (client panics if better-mem if not running)
// panic: runtime error: invalid memory address or nil pointer dereference
// [signal SIGSEGV: segmentation violation code=0x2 addr=0x10 pc=0x10492bbf4]
//
// goroutine 1 [running]:
// github.com/Mateus-Lacerda/better-mem/sdk/better-mem-go.(*BetterMemClient).SendMessage(0x14000160510, {0x1049ac754, 0xe}, {0x1400010e2f0?, 0x104a8d0a0?}, {0x14000076000?, 0x140009efb88?, 0x1049b0712?})
//         /.../.local/share/go/pkg/mod/github.com/!mateus-!lacerda/better-mem/sdk/better-mem-go@v0.0.2/client.go:78 +0x134
// main.(*Chat).sendToBetterMem(0x1400011a640?, {0x1400010e2f0?, 0x10?})
//         /.../jarbas/chat.go:112 +0xb4
// main.(*Chat).addMessage(0x1400011a640, {{0x1049a9993, 0x4}, {0x1400010e2f0, 0x10}, {0x0, 0x0}, {0x0, 0x0, 0x0}, ...})
//         /.../jarbas/chat.go:228 +0x208
// main.(*Chat).loop(0x1400011a640)
//         /.../jarbas/chat.go:261 +0x194
// main.main()
//         /.../jarbas/main.go:22 +0xec
// exit status 2

type Role string

const (
	AssistantRole Role = "assistant"
	UserRole      Role = "user"
	SystemRole    Role = "system"
	ToolRole      Role = "tool"
)

const (
	ollamaURL   string = "http://localhost:11434"
	ollamaModel string = "qwen2.5:7b"
)

// Returns the Ollama Client, panics if there is a problem parsing the URL
func getOllamaClient() *ollama.Client {
	clientURL, err := url.Parse(ollamaURL)
	if err != nil {
		log.Fatal("Please provide a valid ollamaURL")
	}

	return ollama.NewClient(clientURL, http.DefaultClient)
}

func memoriesToContext(memories []core.ScoredMemory) string {
	sb := strings.Builder{}
	sb.WriteString("These are memories of previous conversation you have had with the user:\n")
	for i, mem := range memories {
		fmt.Fprintf(&sb, "%v:\n", i)
		fmt.Fprintf(&sb, "\tThis memory was created on **%v**\n", mem.CreatedAt.String())
		if len(mem.RelatedContext) >= 0 {
			sb.WriteString("Context related with the memory:\n")
		}
		for _, relCtx := range mem.RelatedContext {
			sb.WriteString(relCtx.User + "\n")
			sb.WriteString(relCtx.Context + "\n")
		}
		sb.WriteString("The actual memory:" + fmt.Sprintf("*%v*\n", mem.Text))
	}
	return sb.String()
}

// Holds data for an active chat session
type Chat struct {
	aiResponse            string
	systemPrompt          string
	messages              []ollama.Message
	ctx                   context.Context
	client                *ollama.Client
	streamedMessageBuffer strings.Builder
	bufferSize            uint
	memory                *better_mem.BetterMemClient
	chatId                string
	toolCallResultsBuf    *ToolCallResultsBuffer
	toolDispatcher        *ToolDispatcher
}

func newChat(systemPrompt string, bufferSize uint, chatId string) *Chat {
	return &Chat{
		aiResponse:            "",
		systemPrompt:          systemPrompt,
		messages:              []ollama.Message{{Role: string(SystemRole), Content: systemPrompt}},
		ctx:                   context.Background(),
		client:                getOllamaClient(),
		streamedMessageBuffer: strings.Builder{},
		bufferSize:            bufferSize,
		memory:                better_mem.NewBetterMemClient("http://localhost:5042/api/v1"),
		chatId:                chatId,
		toolCallResultsBuf:    NewToolsResultsBuffer(),
		toolDispatcher:        NewToolDispatcher(),
	}
}

func (c *Chat) setupMemory() {
	c.memory.CreateChat(c.chatId)
}

func (c *Chat) sendToBetterMem(message string) error {
	var relatedContext []core.MessageRelatedContext
	currentChatSize := len(c.messages)
	if currentChatSize >= 3 {
		index := currentChatSize - 3
		// if we have [systemPrompt, some message, last user message], just take the middle one
		if currentChatSize == 3 {
			index = 2
		}
		for _, m := range c.messages[index : currentChatSize-1] {
			relatedContext = append(relatedContext, core.MessageRelatedContext{
				User:    m.Role,
				Context: m.Content,
			})
		}
	}
	return c.memory.SendMessage(
		c.chatId,
		message,
		relatedContext,
	)
}

func (c *Chat) getMemories(message string) string {
	memories, err := c.memory.FetchMemories(
		c.chatId,
		core.MemoryFetchRequest{
			Text:                  message,
			Limit:                 2,
			VectorSearchLimit:     10,
			VectorSearchThreshold: 0.4,
			LongTermThreshold:     0.6,
		},
	)
	if err != nil || len(memories) == 0 {
		return ""
	}
	memoriesPrompt := memoriesToContext(memories)
	fmt.Println(memoriesPrompt)
	return memoriesPrompt
}

func (c *Chat) joinStreamedMessage(gr ollama.ChatResponse) error {
	c.streamedMessageBuffer.WriteString(gr.Message.Content)
	return nil
}

func (c *Chat) printResponseSteps() func(gr ollama.ChatResponse) error {
	return func(gr ollama.ChatResponse) error {
		toolCallsNum := len(gr.Message.ToolCalls)
		if toolCallsNum > 0 {
			fmt.Println("The agent did some tools calls!")

			c.toolCallResultsBuf.Grow(toolCallsNum)

			for _, t := range gr.Message.ToolCalls {
				result, err := c.toolDispatcher.DispatchToolCall(t.Function)
				if err != nil {
					fmt.Println("Error calling tool:")
					fmt.Println(err.Error())
					continue
				}
				c.toolCallResultsBuf.AddResult(
					t,
					result,
				)
			}
		}

		if err := c.joinStreamedMessage(gr); err != nil {
			return err
		}
		fmt.Print(gr.Message.Content)
		if gr.Done {
			fmt.Println()
		}
		return nil
	}
}

// Resets the streamedMessageBuffer
func (c *Chat) resetStreamedMessageBuffer() {
	c.streamedMessageBuffer.Reset()
}

// Returns the streamedMessageBuffer
func (c *Chat) getResponse(clean bool) string {
	response := c.streamedMessageBuffer.String()
	if clean {
		c.resetStreamedMessageBuffer()
	}
	return response
}

// Generates response
func (c *Chat) generateResponse() {

	streamResponse := true

	tools := ollama.Tools{
		{
			Type: "tool",
			Function: ollama.ToolFunction{
				Name:        "SomeTool",
				Description: "Call when you feel like it",
			},
		},
	}
	tools = append(tools, GetCalendarTool())

	req := &ollama.ChatRequest{
		Model:    ollamaModel,
		Messages: c.messages,
		Stream:   &streamResponse,
		Tools:    tools,
	}

	ctx := context.Background()

	if err := c.client.Chat(ctx, req, c.printResponseSteps()); err != nil {
		log.Fatal(err)
	}
}

func (c *Chat) addMessage(m ollama.Message) {

	if uint(len(c.messages))+1 > c.bufferSize {
		c.messages = slices.Delete(c.messages, 1, 2)
	}

	c.messages = append(c.messages, m)
	if m.Role == string(UserRole) {
		c.sendToBetterMem(m.Content)
	}
}

func (c *Chat) addMemories(m string) {
	c.messages[0].Content = c.systemPrompt + c.getMemories(m)
}

func debugMessage(m ollama.Message) {
	fmt.Printf(
		"ROLE: %v\nCONTENT: %v\nTOOL_CALLS: %v\nTOOL_CALL_ID: %v\nTOOL_CALL_NAME: %v\n\n",
		m.Role, m.Content, m.ToolCalls, m.ToolCallID, m.ToolName,
	)
}

func (c *Chat) loop() {
	reader := bufio.NewReader(os.Stdin)

MainLoop:
	for {
		// var userInput string
		fmt.Print("> ")
		userInput, _ := reader.ReadString('\n')

		if strings.TrimSpace(userInput) == "exit" {
			break MainLoop
		}

		userMessage := ollama.Message{
			Role:    string(UserRole),
			Content: userInput,
		}

		c.addMessage(userMessage)
		c.addMemories(userInput)

		canProceed := false
		recursions := 0
		maxRecursions := 5
		for (c.toolCallResultsBuf.HasToolResults() || !canProceed) && recursions < maxRecursions {
			c.generateResponse()

			assistantResponse := c.getResponse(true)
			aiResponse := ollama.Message{
				Role:      string(AssistantRole),
				Content:   assistantResponse,
				ToolCalls: c.toolCallResultsBuf.GetOriginalToolCalls(),
			}

			c.addMessage(aiResponse)
			if c.toolCallResultsBuf.HasToolResults() {
				for _, m := range c.toolCallResultsBuf.GetResultsAsOllamaMessages() {
					c.addMessage(m)
				}
			} else {
				canProceed = true
			}
			// fmt.Println("MESSAGES DEBUG STARTED")
			// for _, m := range c.messages {
			// 	debugMessage(m)
			// }
			// fmt.Println("MESSAGES DEBUG FINISHED")
			c.toolCallResultsBuf.Clear()
			recursions += 1
		}
		fmt.Println()
	}
}
