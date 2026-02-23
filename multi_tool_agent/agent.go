package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	adktool "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

type CityArgs struct {
	City string `json:"city"`
}

func getWeather(_ adktool.Context, args CityArgs) (map[string]any, error) {
	if strings.EqualFold(args.City, "new york") {
		return map[string]any{
			"status": "success",
			"report": "The weather in New York is sunny with a temperature of 25 degrees Celsius (77 degrees Fahrenheit).",
		}, nil
	}

	return map[string]any{
		"status":        "error",
		"error_message": fmt.Sprintf("Weather information for '%s' is not available.", args.City),
	}, nil
}

func getCurrentTime(_ adktool.Context, args CityArgs) (map[string]any, error) {
	if !strings.EqualFold(args.City, "new york") {
		return map[string]any{
			"status":        "error",
			"error_message": fmt.Sprintf("Sorry, I don't have timezone information for %s.", args.City),
		}, nil
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return map[string]any{
			"status":        "error",
			"error_message": fmt.Sprintf("Failed to load timezone data for America/New_York: %v", err),
		}, nil
	}

	now := time.Now().In(loc)
	report := fmt.Sprintf("The current time in %s is %s", args.City, now.Format("2006-01-02 15:04:05 MST-0700"))
	return map[string]any{"status": "success", "report": report}, nil
}

func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, os.Getenv("LLM_MODEL"), &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	weatherTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_weather",
			Description: "Retrieves the current weather report for a specified city.",
		},
		getWeather,
	)
	if err != nil {
		log.Fatalf("Failed to create get_weather tool: %v", err)
	}

	timeTool, err := functiontool.New(
		functiontool.Config{
			Name:        "get_current_time",
			Description: "Returns the current time in a specified city.",
		},
		getCurrentTime,
	)
	if err != nil {
		log.Fatalf("Failed to create get_current_time tool: %v", err)
	}

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "weather_time_agent",
		Model:       model,
		Description: "Agent to answer questions about the time and weather in a city.",
		Instruction: "You are a helpful agent who can answer user questions about the time and weather in a city.",
		Tools:       []adktool.Tool{weatherTool, timeTool},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	cfg := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(rootAgent),
	}

	l := full.NewLauncher()
	if err := l.Execute(ctx, cfg, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}