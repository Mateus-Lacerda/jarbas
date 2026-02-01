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

	ollama "github.com/ollama/ollama/api"
)

type Role string

const (
	Assistant Role = "assistant"
	User      Role = "user"
	System    Role = "system"
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

// Holds data for an active chat session
type Chat struct {
	aiResponse            string
	messages              []ollama.Message
	ctx                   context.Context
	client                *ollama.Client
	streamedMessageBuffer strings.Builder
	bufferSize            uint
}

func newChat(systemPrompt string, bufferSize uint) *Chat {
	return &Chat{
		aiResponse:            "",
		messages:              []ollama.Message{{Role: string(System), Content: systemPrompt}},
		ctx:                   context.Background(),
		client:                getOllamaClient(),
		streamedMessageBuffer: strings.Builder{},
		bufferSize:            bufferSize,
	}
}

func (c *Chat) joinStreamedMessage(gr ollama.ChatResponse) error {
	c.streamedMessageBuffer.WriteString(gr.Message.Content)
	return nil
}

func (c *Chat) printResponseSteps() func(gr ollama.ChatResponse) error {
	return func(gr ollama.ChatResponse) error {
		if err := c.joinStreamedMessage(gr); err != nil {
			return err
		}
		if len(gr.Message.ToolCalls) > 0 {
			fmt.Println("Tool Calls!")
		}
		fmt.Print(gr.Message.Content)
		if gr.Done {
			fmt.Println()
		}
		return nil
	}
}

func (c *Chat) resetStreamedMessageBuffer() {
	c.streamedMessageBuffer.Reset()
}

func (c *Chat) getResponse() string {
	return c.streamedMessageBuffer.String()
}

func (c *Chat) generateResponse() {

	c.resetStreamedMessageBuffer()

	streamResponse := true
	
	log.Println("Answering these messages", c.messages)

	tools := ollama.Tools{
		{Type: "tool", Function: ollama.ToolFunction{
			Name:        "SomeTool",
			Description: "Call when you feel like it",
		}},
	}

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
}

func SomeTool() {}

func (c *Chat) loop() {
	reader := bufio.NewReader(os.Stdin)
MainLoop:
	for {
		// var userInput string
		fmt.Print("> ")
		userInput, _ := reader.ReadString('\n')

		if userInput == "exit" {
			break MainLoop
		}

		userMessage := ollama.Message{
			Role:    string(User),
			Content: userInput,
		}

		c.addMessage(userMessage)
		c.generateResponse()

		aiResponse := ollama.Message{
			Role:    string(Assistant),
			Content: c.getResponse(),
		}
		c.addMessage(aiResponse)
		fmt.Println()
	}
}
