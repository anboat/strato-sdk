// Package main provides an example of how to use the Strato SDK for streaming research.
package main

import (
	"context"
	"fmt"
	"github.com/anboat/strato-sdk/config"
	"github.com/anboat/strato-sdk/core/agent"
	"github.com/anboat/strato-sdk/pkg/logging"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main is the entry point for the example application.
// It initializes the configuration and logger, creates a streaming research agent,
// executes a research query, and processes the streaming results.
func main() {
	// Initialize configuration from a YAML file.
	allConfig := config.LoadConfig("config.yaml")
	if allConfig == nil {
		fmt.Printf("Failed to load configuration\n")
		return
	}

	// Initialize the logger based on the loaded configuration.
	logging.InitLoggerFromConfig(&allConfig.Log)

	// Create a background context.
	ctx := context.Background()

	// Create a new streaming research agent.
	rAgent, err := agent.NewStreamingResearchAgent(ctx)
	if err != nil {
		fmt.Printf("Failed to create streaming research agent: %v\n", err)
		return
	}

	// Define the research query.
	query := "What is the future of AI in 2024?"

	// Execute the streaming research process.
	thoughtChan, err := rAgent.ResearchWithStreaming(ctx, query)
	if err != nil {
		fmt.Printf("Failed to start streaming research: %v\n", err)
		return
	}

	// Set up a channel to handle OS signals for graceful interruption.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Process the streaming thought results in a separate goroutine.
	completed := false
	var lastStage string
	go func() {
		defer func() {
			// Ensure the main loop knows processing is done.
			completed = true
		}()
		for thought := range thoughtChan {
			// Display a header when the research stage changes.
			if thought.Stage != lastStage {
				displayStageHeader(thought.Stage)
				lastStage = thought.Stage
			}
			// Print the content of the thought.
			fmt.Print(thought.Content)
			if thought.IsComplete {
				// Mark as complete and exit the goroutine.
				return
			}
		}
	}()

	// Wait for the research to complete or for an interruption signal.
	for !completed {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, shutting down.\n")
			return
		case <-time.After(10 * time.Minute):
			fmt.Println("\nResearch timed out after 10 minutes.\n")
			return
		case <-time.After(100 * time.Millisecond):
			// Continue waiting for completion.
		}
	}
	fmt.Println("\nResearch process finished.\n")
}

// displayStageHeader prints a formatted header for each research stage
// to visually separate the different phases of the agent's process.
func displayStageHeader(stage string) {
	fmt.Println() // Add a newline for better spacing.
	switch stage {
	case agent.StageThinking:
		fmt.Println("===== 🧠 Thinking Stage =====")
	case agent.StageSearching:
		fmt.Println("===== 🔎 Searching Stage =====")
	case agent.StageAnalyzing:
		fmt.Println("===== 🔬 Analyzing Stage =====")
	case agent.StageSynthesizing:
		fmt.Println("===== ✍️ Synthesizing Stage =====")
	case agent.StageCompleted:
		fmt.Println("===== ✅ Completed Stage =====")
	case agent.StageError:
		fmt.Println("===== ❌ Error Stage =====")
	default:
		fmt.Printf("===== %s Stage =====\n", stage)
	}
	fmt.Println() // Add a newline for better spacing.
}
