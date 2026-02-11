package main

import (
	"encoding/json"
	"fmt"

	ollama "github.com/ollama/ollama/api"
)

func SomeTool(_ ollama.ToolCallFunctionArguments) string {
	fmt.Println("Some tools ")
	return "Now what?"
}

type GetCalendarSchema struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

func GetCalendar(s GetCalendarSchema) string {
	return fmt.Sprintf("Sorry, no events for day %v, hour %v", s.Date, s.Time)
}

func GetCalendarTool() ollama.Tool {
	getCalendarProps := ollama.NewToolPropertiesMap()
	getCalendarProps.Set(
		"date",
		ollama.ToolProperty{
			Type:        ollama.PropertyType{"string"},
			Description: "The date to search for in the format MM/DD/YY",
		},
	)
	getCalendarProps.Set(
		"hour",
		ollama.ToolProperty{
			Type:        ollama.PropertyType{"string"},
			Description: "The hour of the day to search for in 24h format",
		},
	)

	tool := ollama.Tool{
		Type: "tool",
		Function: ollama.ToolFunction{
			Name:        "GetCalendar",
			Description: "Returns the user's calendar stuff for the given day",
			Parameters: ollama.ToolFunctionParameters{
				Type:       "object",
				Properties: getCalendarProps,
				Required:   []string{"date"},
			},
		},
	}
	return tool

}

func CallCalendarTool(arguments ollama.ToolCallFunctionArguments) string {
	bytes, err := json.Marshal(arguments)
	if err != nil {
		return "Error calling tool"
	}
	var schema GetCalendarSchema
	if err := json.Unmarshal(bytes, &schema); err != nil {
		return "Error calling tool"
	}
	return GetCalendar(schema)
}
