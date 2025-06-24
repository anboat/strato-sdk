package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/anboat/strato-sdk/adapters/llm"
	"github.com/anboat/strato-sdk/config"
	tools2 "github.com/anboat/strato-sdk/core/tools"
	"github.com/anboat/strato-sdk/pkg/logging"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"strings"
	"time"
	// Anonymous imports to ensure adapter init functions are called.
	_ "github.com/anboat/strato-sdk/adapters/search/firecrawl"
	_ "github.com/anboat/strato-sdk/adapters/search/searxng"
	_ "github.com/anboat/strato-sdk/adapters/search/twitter"
	// Anonymous imports to ensure adapter init functions are called.
	_ "github.com/anboat/strato-sdk/adapters/web/firecrawl"
	_ "github.com/anboat/strato-sdk/adapters/web/jina"
)

// StreamingThought represents a single thought or piece of information streamed
// during the research process. It provides real-time updates on the agent's state and actions.
type StreamingThought struct {
	Timestamp  time.Time `json:"timestamp"`   // Timestamp of when the thought was generated.
	Stage      string    `json:"stage"`       // Current stage: thinking, searching, analyzing, synthesizing.
	Content    string    `json:"content"`     // The specific content of the thought or analysis result.
	Action     Action    `json:"action"`      // The action currently being executed.
	IsComplete bool      `json:"is_complete"` // Indicates if the entire research process is complete.
	Sources    []string  `json:"sources"`     // List of source URLs for traceability.
}

// ResearchQuestion represents a specific sub-question within the research process,
// encompassing its complete lifecycle from generation to analysis.
type ResearchQuestion struct {
	ID            string                      `json:"id"`             // Unique identifier for the question (e.g., q_iteration_index).
	Question      string                      `json:"question"`       // The content of the research question.
	Status        string                      `json:"status"`         // Status: pending, researching, completed.
	SearchResults []*tools2.SearchResponse    `json:"search_results"` // List of results from the search engine.
	WebContents   []*tools2.WebScrapeResponse `json:"web_contents"`   // Scraped web content details.
	Analysis      string                      `json:"analysis"`       // In-depth analysis based on the collected information.
	Priority      int                         `json:"priority"`       // Priority of the question (1-10), higher is more important.
}

// StreamingResearchState maintains the state of the entire research process,
// supporting multiple iterations and complex control flow.
type StreamingResearchState struct {
	OriginalQuery       string                 `json:"original_query"`       // The user's original query.
	CurrentIteration    int                    `json:"current_iteration"`    // The current iteration number, starting from 0.
	MaxIterations       int                    `json:"max_iterations"`       // The maximum number of allowed iterations.
	ResearchQuestions   []*ResearchQuestion    `json:"research_questions"`   // List of all generated research questions.
	ResearchedQuestions map[string]bool        `json:"researched_questions"` // A map to track researched questions for deduplication.
	CurrentResearchQ    *ResearchQuestion      `json:"current_research_q"`   // The question currently being researched.
	AccumulatedInfo     string                 `json:"accumulated_info"`     // Accumulated research information (reserved field).
	FinalAnswer         string                 `json:"final_answer"`         // The final answer synthesized from all research findings.
	IsComplete          bool                   `json:"is_complete"`          // Indicates if the entire research process is complete.
	CompletedQuestions  int                    `json:"completed_questions"`  // The number of completed research questions.
	ThoughtChannel      chan *StreamingThought `json:"-"`                    // Channel for transmitting streaming thoughts (not serialized to JSON).
}

// StreamingResearchAgent is an intelligent research agent based on the Eino framework,
// supporting real-time, streaming output of the research process.
type StreamingResearchAgent struct {
	chatModel  model.ToolCallingChatModel                                         // The large language model for generating questions, analyzing content, and synthesizing answers.
	searchTool tool.InvokableTool                                                 // The search tool, supporting various search engine adapters.
	webTool    tool.InvokableTool                                                 // The web scraping tool for fetching detailed web content.
	graph      compose.Runnable[*StreamingResearchState, *StreamingResearchState] // The Eino workflow graph defining the research process logic.
}

// NewStreamingResearchAgent creates a new StreamingResearchAgent.
// It initializes all necessary components (LLM, search tool, web scraping tool)
// and builds the workflow graph that defines the agent's behavior.
//
// Parameters:
//   - ctx: A context.Context to control the initialization lifecycle.
//
// Returns:
//   - *StreamingResearchAgent: An initialized research agent instance.
//   - error: An error if any part of the initialization fails.
func NewStreamingResearchAgent(ctx context.Context) (*StreamingResearchAgent, error) {

	// Create base components.
	chatModel, err := llm.GetDefaultChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChatModel: %w", err)
	}

	searchTool, err := tools2.NewSearchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create SearchTool: %w", err)
	}

	webTool, err := tools2.NewWebProcessTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create WebProcessTool: %w", err)
	}

	agent := &StreamingResearchAgent{
		chatModel:  chatModel,
		searchTool: searchTool,
		webTool:    webTool,
	}

	// Build the research graph.
	graph, err := agent.buildStreamingResearchGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build streaming research graph: %w", err)
	}

	agent.graph = graph
	return agent, nil
}

// ResearchWithStreaming executes a streaming research process.
// It starts an asynchronous research workflow and returns a channel that provides
// real-time thoughts and updates from the agent.
//
// Parameters:
//   - ctx: A context.Context to control the research lifecycle.
//   - query: The user's original research query.
//
// Returns:
//   - <-chan *StreamingThought: A read-only channel for receiving streaming thoughts.
//   - error: An error if the research process fails to start.
func (agent *StreamingResearchAgent) ResearchWithStreaming(ctx context.Context, query string) (<-chan *StreamingThought, error) {
	// Get the research configuration.
	researchConfig := config.GetResearchConfig()

	// Create the thought channel.
	thoughtChan := make(chan *StreamingThought, researchConfig.ChannelBuffer)

	// Initialize the research state.
	initialState := &StreamingResearchState{
		OriginalQuery:       query,
		CurrentIteration:    0,
		MaxIterations:       researchConfig.MaxIterations,
		ResearchQuestions:   make([]*ResearchQuestion, 0),
		ResearchedQuestions: make(map[string]bool),
		AccumulatedInfo:     "",
		FinalAnswer:         "",
		IsComplete:          false,
		CompletedQuestions:  0,
		ThoughtChannel:      thoughtChan,
	}

	// Execute the research in a goroutine.
	go func() {
		defer close(thoughtChan)

		logging.Infof("Starting streaming research: %s", query)

		// Invoke the research graph.
		finalState, err := agent.graph.Invoke(ctx, initialState)
		if err != nil {
			logging.Errorf("Research graph execution failed: %v", err)
			// Send an error message.
			thoughtChan <- &StreamingThought{
				Timestamp:  time.Now(),
				Stage:      StageError,
				Content:    fmt.Sprintf("Research process encountered an error: %v", err),
				Action:     ActionError,
				IsComplete: true,
			}
			return
		}

		// Send the final answer, with a nil check for finalState.
		if finalState != nil && finalState.IsComplete {
			thoughtChan <- &StreamingThought{
				Timestamp:  time.Now(),
				Stage:      StageCompleted,
				Content:    finalState.FinalAnswer,
				Action:     ActionResearchComplete,
				IsComplete: true,
				Sources:    agent.extractSources(finalState),
			}
		} else if finalState == nil {
			logging.Warnf("Research graph returned a nil finalState without an error.")
		}
	}()

	return thoughtChan, nil
}

// buildStreamingResearchGraph constructs the complex workflow graph using the Eino framework.
// This graph defines the execution logic, branching, and iteration control for the research process.
//
// Parameters:
//   - ctx: A context.Context for the graph compilation process.
//
// Returns:
//   - compose.Runnable: An executable workflow graph instance.
//   - error: An error if the graph construction fails.
func (agent *StreamingResearchAgent) buildStreamingResearchGraph(ctx context.Context) (compose.Runnable[*StreamingResearchState, *StreamingResearchState], error) {
	// Get research configuration.
	researchConfig := config.GetResearchConfig()

	// Create Graph
	g := compose.NewGraph[*StreamingResearchState, *StreamingResearchState]()

	// Create Lambda nodes
	generateQuestionsLambda := compose.InvokableLambda(agent.createGenerateQuestionsNode())
	selectQuestionLambda := compose.InvokableLambda(agent.createSelectQuestionNode())
	searchQuestionLambda := compose.InvokableLambda(agent.createSearchQuestionNode())
	scrapeWebContentLambda := compose.InvokableLambda(agent.createScrapeWebContentNode())
	analyzeQuestionLambda := compose.InvokableLambda(agent.createAnalyzeQuestionNode())
	synthesizeFinalAnswerLambda := compose.InvokableLambda(agent.createSynthesizeFinalAnswerNode())
	incrementIterationLambda := compose.InvokableLambda(agent.createIncrementIterationNode())

	// Add nodes
	_ = g.AddLambdaNode(NodeGenerateQuestions, generateQuestionsLambda)
	_ = g.AddLambdaNode(NodeSelectQuestion, selectQuestionLambda)
	_ = g.AddLambdaNode(NodeSearchQuestion, searchQuestionLambda)
	_ = g.AddLambdaNode(NodeScrapeWebContent, scrapeWebContentLambda)
	_ = g.AddLambdaNode(NodeAnalyzeQuestion, analyzeQuestionLambda)
	_ = g.AddLambdaNode(NodeSynthesizeFinalAnswer, synthesizeFinalAnswerLambda)
	_ = g.AddLambdaNode(NodeIncrementIteration, incrementIterationLambda)

	// Create branch conditions
	checkCompletionCondition := func(ctx context.Context, state *StreamingResearchState) (string, error) {
		// Record current research progress
		logging.Infof("Checking research status - Iteration: %d/%d, Question count: %d, Completed: %d",
			state.CurrentIteration, state.MaxIterations,
			len(state.ResearchQuestions), state.CompletedQuestions)

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageThinking,
			Content: fmt.Sprintf("Checking research progress - Iteration: %d/%d, Completed questions: %d",
				state.CurrentIteration, state.MaxIterations, state.CompletedQuestions),
			Action: ActionProgressCheck,
		})

		// Check if maximum iteration count is reached
		if state.CurrentIteration >= state.MaxIterations {
			agent.sendThought(state, &StreamingThought{
				Timestamp: time.Now(),
				Stage:     StageThinking,
				Content:   "Maximum iteration count reached, starting synthesis of final answer",
				Action:    ActionIterationComplete,
			})
			return NodeSynthesizeFinalAnswer, nil
		}

		// If enough completed questions have been reached, consider ending
		if state.CompletedQuestions > 0 {
			// Summarize completed question content
			var completedContent strings.Builder
			for _, q := range state.ResearchQuestions {
				if q.Status == QuestionStatusCompleted && q.Analysis != "" {
					completedContent.WriteString("- ")
					completedContent.WriteString(q.Question)
					completedContent.WriteString(": ")
					completedContent.WriteString(q.Analysis)
					completedContent.WriteString("\n")
				}
			}
			prompt := fmt.Sprintf(ShouldSynthesizeEarlyPromptTemplate, state.OriginalQuery, completedContent.String())
			messages := []*schema.Message{{Role: schema.User, Content: prompt}}
			response, err := agent.chatModel.Generate(ctx, messages)
			if err == nil && strings.Contains(strings.ToLower(response.Content), "true") {
				agent.sendThought(state, &StreamingThought{
					Timestamp: time.Now(),
					Stage:     StageThinking,
					Content:   fmt.Sprintf("Model judged information sufficient (completed %d), starting synthesis of final answer", state.CompletedQuestions),
					Action:    ActionModelJudgeSufficient,
				})
				return NodeSynthesizeFinalAnswer, nil
			}
		}

		// Check if there are pending questions to be researched
		hasPendingQuestions := false
		for _, q := range state.ResearchQuestions {
			if q.Status == QuestionStatusPending {
				hasPendingQuestions = true
				break
			}
		}

		if hasPendingQuestions {
			agent.sendThought(state, &StreamingThought{
				Timestamp: time.Now(),
				Stage:     StageThinking,
				Content:   "Found pending questions to continue research",
				Action:    ActionContinueResearch,
			})
			return NodeSelectQuestion, nil
		}

		// If there are no pending questions but completed questions are few, generate new questions
		if state.CompletedQuestions < researchConfig.MinQuestions && state.CurrentIteration < state.MaxIterations {
			agent.sendThought(state, &StreamingThought{
				Timestamp: time.Now(),
				Stage:     StageThinking,
				Content:   fmt.Sprintf("Completed question count is low (%d), need to generate more research questions", state.CompletedQuestions),
				Action:    ActionGenerateNewQuestions,
			})
			return NodeGenerateQuestions, nil
		}

		// Otherwise, directly synthesize final answer
		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageThinking,
			Content:   "No more pending questions, starting synthesis of final answer",
			Action:    ActionPrepareSynthesis,
		})
		return NodeSynthesizeFinalAnswer, nil
	}

	selectQuestionCondition := func(ctx context.Context, state *StreamingResearchState) (string, error) {
		// Check if a question has been successfully selected
		if state.CurrentResearchQ != nil {
			return NodeSearchQuestion, nil
		}
		// If there are no questions to select, directly proceed to final synthesis
		return NodeSynthesizeFinalAnswer, nil
	}

	checkCompletionEndNodes := map[string]bool{
		NodeSelectQuestion:        true,
		NodeGenerateQuestions:     true,
		NodeSynthesizeFinalAnswer: true,
	}

	selectEndNodes := map[string]bool{
		NodeSearchQuestion:        true,
		NodeSynthesizeFinalAnswer: true,
	}

	checkCompletionBranch := compose.NewGraphBranch(checkCompletionCondition, checkCompletionEndNodes)
	selectBranch := compose.NewGraphBranch(selectQuestionCondition, selectEndNodes)

	// Add edges and branches - starting directly from the checkCompletion branch
	_ = g.AddBranch(compose.START, checkCompletionBranch)
	_ = g.AddEdge(NodeGenerateQuestions, NodeIncrementIteration)
	_ = g.AddBranch(NodeSelectQuestion, selectBranch)
	_ = g.AddEdge(NodeSearchQuestion, NodeScrapeWebContent)
	_ = g.AddEdge(NodeScrapeWebContent, NodeAnalyzeQuestion)
	_ = g.AddBranch(NodeAnalyzeQuestion, checkCompletionBranch)
	_ = g.AddBranch(NodeIncrementIteration, checkCompletionBranch)
	_ = g.AddEdge(NodeSynthesizeFinalAnswer, compose.END)

	// Compile the graph, using the max steps from the configuration.
	return g.Compile(ctx, compose.WithGraphName(GraphNameStreamingResearch), compose.WithMaxRunSteps(researchConfig.MaxSteps))
}

// createGenerateQuestionsNode creates a node for generating research questions.
// It uses the LLM to analyze the original query and generate specific sub-questions.
// It includes a deduplication mechanism to avoid creating questions similar to those already researched.
// Returns a function that performs the node's logic.
func (agent *StreamingResearchAgent) createGenerateQuestionsNode() func(context.Context, *StreamingResearchState) (*StreamingResearchState, error) {
	return func(ctx context.Context, state *StreamingResearchState) (*StreamingResearchState, error) {
		// Get research configuration.
		researchConfig := config.GetResearchConfig()

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageThinking,
			Content:   "Analyzing the original query to generate specific sub-questions...",
			Action:    ActionGenerateQuestions,
		})

		// Build a list of already researched questions.
		var researchedList []string
		for question := range state.ResearchedQuestions {
			researchedList = append(researchedList, question)
		}

		researchedText := ""
		if len(researchedList) > 0 {
			researchedText = fmt.Sprintf("\n\n# Already researched questions (please avoid repeating)\n%s", strings.Join(researchedList, "\n"))
		}

		prompt := fmt.Sprintf(GenerateQuestionsPromptTemplate, state.OriginalQuery, researchedText)

		messages := []*schema.Message{
			{
				Role:    schema.User,
				Content: prompt,
			},
		}

		response, err := agent.chatModel.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("failed to generate research questions: %w", err)
		}

		// Parse the JSON response.
		var questionData []struct {
			Question string `json:"question"`
			Priority int    `json:"priority"`
		}

		if err := json.Unmarshal([]byte(response.Content), &questionData); err != nil {
			return nil, fmt.Errorf("failed to parse research questions JSON: %w", err)
		}

		// Convert to ResearchQuestion struct and limit the maximum number.
		var newQuestions []*ResearchQuestion

		// Calculate maxTotalQuestions and maxNewQuestions based on maxSteps.
		maxTotalQuestions, maxNewQuestions := calculateMaxQuestions(researchConfig.MaxSteps, len(state.ResearchQuestions))

		logging.Infof("Step allocation calculation - Max steps: %d, Steps per question: %d, Max total questions: %d, Existing questions: %d, Can add: %d",
			researchConfig.MaxSteps, StepsPerQuestion, maxTotalQuestions, len(state.ResearchQuestions), maxNewQuestions)

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageThinking,
			Content: fmt.Sprintf("Step allocation calculation: Max steps %d, Steps per question %d, Can research %d questions, Existing %d, Can add %d",
				researchConfig.MaxSteps, StepsPerQuestion, maxTotalQuestions, len(state.ResearchQuestions), maxNewQuestions),
			Action: ActionStepAllocation,
		})

		if maxNewQuestions <= 0 {
			agent.sendThought(state, &StreamingThought{
				Timestamp: time.Now(),
				Stage:     StageThinking,
				Content:   fmt.Sprintf("Reached maximum sub-question count limit (based on max steps %d calculated up to %d questions), Skip generating new questions", researchConfig.MaxSteps, maxTotalQuestions),
				Action:    ActionQuestionLimitReached,
			})
			return state, nil
		}

		for i, data := range questionData {
			// Limit the number of new questions.
			if len(newQuestions) >= maxNewQuestions {
				break
			}

			// Check if a similar question has already been researched.
			if agent.isSimilarQuestionResearched(data.Question, state.ResearchedQuestions) {
				continue
			}

			question := &ResearchQuestion{
				ID:            fmt.Sprintf("%s%d_%d", QuestionIDPrefix, state.CurrentIteration, i+1),
				Question:      data.Question,
				Status:        QuestionStatusPending,
				SearchResults: make([]*tools2.SearchResponse, 0),
				WebContents:   make([]*tools2.WebScrapeResponse, 0),
				Analysis:      "",
				Priority:      data.Priority,
			}
			newQuestions = append(newQuestions, question)
		}

		// Add new questions to the state.
		state.ResearchQuestions = append(state.ResearchQuestions, newQuestions...)

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageThinking,
			Content:   fmt.Sprintf("Generated %d new research questions, ready to continue research", len(newQuestions)),
			Action:    ActionQuestionGenComplete,
		})

		logging.Infof("Generated %d new research questions", len(newQuestions))
		return state, nil
	}
}

// createSelectQuestionNode creates a node for selecting the next research question.
// It selects the pending question with the highest priority to be researched next,
// following a greedy strategy.
// Returns a function that performs the node's logic.
func (agent *StreamingResearchAgent) createSelectQuestionNode() func(context.Context, *StreamingResearchState) (*StreamingResearchState, error) {
	return func(ctx context.Context, state *StreamingResearchState) (*StreamingResearchState, error) {
		// Find the pending question with the highest priority.
		var selectedQuestion *ResearchQuestion
		maxPriority := -1

		for _, q := range state.ResearchQuestions {
			if q.Status == QuestionStatusPending && q.Priority > maxPriority {
				maxPriority = q.Priority
				selectedQuestion = q
			}
		}

		if selectedQuestion == nil {
			logging.Infof("No pending questions found")
			state.CurrentResearchQ = nil
			return state, nil
		}

		// Set the current question to be researched.
		state.CurrentResearchQ = selectedQuestion
		selectedQuestion.Status = QuestionStatusResearching

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageThinking,
			Content:   fmt.Sprintf("Selected the highest priority research question: %s", selectedQuestion.Question),
			Action:    ActionQuestionSelection,
		})

		logging.Infof("Selected research question: %s (Priority: %d)", selectedQuestion.Question, selectedQuestion.Priority)
		return state, nil
	}
}

// createSearchQuestionNode creates a node for searching for information related to a question.
// It uses the configured search tool to perform a web search for the current research question.
// It supports multiple search engines (Baidu, Bing, Google, etc.).
// Returns a function that performs the node's logic, executing the search and saving the results.
func (agent *StreamingResearchAgent) createSearchQuestionNode() func(context.Context, *StreamingResearchState) (*StreamingResearchState, error) {
	return func(ctx context.Context, state *StreamingResearchState) (*StreamingResearchState, error) {
		if state.CurrentResearchQ == nil {
			return state, nil
		}

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageSearching,
			Content:   fmt.Sprintf("Searching the web for: \"%s\"", state.CurrentResearchQ.Question),
			Action:    ActionNetworkSearch,
		})

		// Build the search request.
		searchReq := &tools2.SearchRequest{
			Query: state.CurrentResearchQ.Question,
		}

		searchReqJSON, err := json.Marshal(searchReq)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize search request: %w", err)
		}

		// Execute the search.
		resultStr, err := agent.searchTool.InvokableRun(ctx, string(searchReqJSON))
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		// Deserialize the search result.
		var searchResp tools2.SearchResponse
		if err := json.Unmarshal([]byte(resultStr), &searchResp); err != nil {
			return nil, fmt.Errorf("failed to deserialize search result: %w", err)
		}

		// Update the search results for the question.
		state.CurrentResearchQ.SearchResults = append(state.CurrentResearchQ.SearchResults, &searchResp)

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageSearching,
			Content:   fmt.Sprintf("Search complete, found %d relevant results", len(searchResp.Results)),
			Action:    ActionSearchComplete,
		})

		logging.Infof("Search complete - found %d results", len(searchResp.Results))
		return state, nil
	}
}

// createScrapeWebContentNode creates a node for scraping web content.
// It extracts URLs from search results and uses the web scraping tool to get detailed content.
// It supports multiple scraping adapters (Firecrawl, Jina, etc.) and limits the number of scrapes to avoid overload.
// Returns a function that performs the node's logic, scraping content and saving it to the current question.
func (agent *StreamingResearchAgent) createScrapeWebContentNode() func(context.Context, *StreamingResearchState) (*StreamingResearchState, error) {
	return func(ctx context.Context, state *StreamingResearchState) (*StreamingResearchState, error) {
		if state.CurrentResearchQ == nil || len(state.CurrentResearchQ.SearchResults) == 0 {
			return state, nil
		}

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageAnalyzing,
			Content:   "Scraping content from URLs to get detailed information...",
			Action:    ActionWebScraping,
		})

		// Get URLs from the latest search results.
		latestSearch := state.CurrentResearchQ.SearchResults[len(state.CurrentResearchQ.SearchResults)-1]
		var urls []string
		for _, item := range latestSearch.Results {
			if item.URL != "" {
				urls = append(urls, item.URL)
			}
		}

		if len(urls) == 0 {
			agent.sendThought(state, &StreamingThought{
				Timestamp: time.Now(),
				Stage:     StageAnalyzing,
				Content:   "No URLs found to scrape, skipping web content scraping.",
				Action:    ActionSkipScraping,
			})
			return state, nil
		}

		// Build web scraping request.
		webReq := &tools2.WebScrapeRequest{
			URLs:   urls,
			Format: "text",
		}

		webReqJSON, err := json.Marshal(webReq)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize web scrape request: %w", err)
		}

		// Execute web scraping.
		resultStr, err := agent.webTool.InvokableRun(ctx, string(webReqJSON))
		if err != nil {
			return nil, fmt.Errorf("web scraping failed: %w", err)
		}

		// Deserialize scraping result.
		var webResp tools2.WebScrapeResponse
		if err := json.Unmarshal([]byte(resultStr), &webResp); err != nil {
			return nil, fmt.Errorf("failed to deserialize web scrape result: %w", err)
		}

		// Update the web content for the question.
		state.CurrentResearchQ.WebContents = append(state.CurrentResearchQ.WebContents, &webResp)

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageAnalyzing,
			Content:   fmt.Sprintf("Web scraping complete, successfully fetched content from %d pages", len(webResp.Results)),
			Action:    ActionScrapingComplete,
		})

		logging.Infof("Web scraping complete - successfully scraped %d pages", len(webResp.Results))
		return state, nil
	}
}

// createAnalyzeQuestionNode creates a node for analyzing the gathered information.
// It uses the LLM to perform an in-depth analysis of the collected web content and generate a detailed answer.
// It includes a content truncation mechanism to stay within model context limits and requires source citation.
// Returns a function that performs the node's logic, analyzing content and completing the current question.
func (agent *StreamingResearchAgent) createAnalyzeQuestionNode() func(context.Context, *StreamingResearchState) (*StreamingResearchState, error) {
	return func(ctx context.Context, state *StreamingResearchState) (*StreamingResearchState, error) {
		if state.CurrentResearchQ == nil {
			return state, nil
		}

		// Get research configuration.
		researchConfig := config.GetResearchConfig()

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageAnalyzing,
			Content:   fmt.Sprintf("Starting in-depth analysis of collected information for: %s", state.CurrentResearchQ.Question),
			Action:    ActionContentAnalysis,
		})

		// Build analysis prompt, limiting content length.
		var contentBuilder strings.Builder
		contentBuilder.WriteString("# Research Question\n")
		contentBuilder.WriteString(state.CurrentResearchQ.Question)
		contentBuilder.WriteString("\n\n# Collected Web Content\n")

		// Add web content, but limit total length.
		currentLength := 0
		webPageIndex := 1

		for _, webBatch := range state.CurrentResearchQ.WebContents {
			for _, content := range webBatch.Results {
				if currentLength >= researchConfig.MaxContentLength {
					contentBuilder.WriteString("\nNote: Due to excessive content, only a portion of the web content is displayed.\n")
					break
				}

				// If a single web page's content is too long, truncate it.
				truncatedContent := content.Content
				if len(truncatedContent) > researchConfig.MaxSingleContent {
					truncatedContent = truncatedContent[:researchConfig.MaxSingleContent] + "...(content truncated)"
				}

				entryContent := fmt.Sprintf("## Source Web Page %d\n**Link**: %s\n**Title**: %s\n**Content**:\n%s\n\n---\n\n",
					webPageIndex, content.URL, content.Title, truncatedContent)

				contentBuilder.WriteString(entryContent)
				currentLength += len(entryContent)
				webPageIndex++
			}

			if currentLength >= researchConfig.MaxContentLength {
				break
			}
		}

		analyzePrompt := fmt.Sprintf(AnalyzeQuestionPromptTemplate, contentBuilder.String())

		messages := []*schema.Message{
			{
				Role:    schema.User,
				Content: analyzePrompt,
			},
		}

		// Call the large model for streaming analysis.
		stream, err := agent.chatModel.Stream(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("Analysis failed: %w", err)
		}

		var analysisResult strings.Builder
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}

			analysisResult.WriteString(chunk.Content)

			// Send analysis content in real-time.
			agent.sendThought(state, &StreamingThought{
				Timestamp: time.Now(),
				Stage:     StageAnalyzing,
				Content:   chunk.Content,
				Action:    ActionRealtimeAnalysis,
			})
		}

		// Update the analysis result for the question.
		state.CurrentResearchQ.Analysis = analysisResult.String()
		state.CurrentResearchQ.Status = QuestionStatusCompleted

		// Mark the question as researched.
		state.ResearchedQuestions[state.CurrentResearchQ.Question] = true

		// Update the completed question count.
		state.CompletedQuestions++

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageAnalyzing,
			Content:   fmt.Sprintf("Analysis complete\n\n**Research Question**: %s\n\n**Analysis Result**:\n%s", state.CurrentResearchQ.Question, analysisResult.String()),
			Action:    ActionAnalysisComplete,
		})

		// Clear the current question to prepare for the next one.
		state.CurrentResearchQ = nil

		logging.Infof("Analysis complete - Completed questions: %d", state.CompletedQuestions)
		return state, nil
	}
}

// createSynthesizeFinalAnswerNode creates a node for synthesizing the final answer.
// This is the terminal node of the workflow, integrating analysis results from all completed questions.
// It uses the LLM to generate a comprehensive, structured answer to the original query.
// Returns a function that performs the node's logic, generating the final answer and marking the research as complete.
func (agent *StreamingResearchAgent) createSynthesizeFinalAnswerNode() func(context.Context, *StreamingResearchState) (*StreamingResearchState, error) {
	return func(ctx context.Context, state *StreamingResearchState) (*StreamingResearchState, error) {
		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageSynthesizing,
			Content:   "Starting to synthesize all research findings into a final answer...",
			Action:    ActionSynthesisAnalysis,
		})

		// Build synthesis prompt.
		var contentBuilder strings.Builder
		contentBuilder.WriteString("# Original Question\n")
		contentBuilder.WriteString(state.OriginalQuery)
		contentBuilder.WriteString("\n\n# Research Questions and Analysis Results\n")

		for i, q := range state.ResearchQuestions {
			if q.Status == QuestionStatusCompleted && q.Analysis != "" {
				contentBuilder.WriteString(fmt.Sprintf("## Research Question %d: %s\n", i+1, q.Question))
				contentBuilder.WriteString(q.Analysis)
				contentBuilder.WriteString("\n\n---\n\n")
			}
		}

		synthesizePrompt := fmt.Sprintf(SynthesizeFinalAnswerPromptTemplate, contentBuilder.String())

		messages := []*schema.Message{
			{
				Role:    schema.User,
				Content: synthesizePrompt,
			},
		}

		// Call the large model for streaming synthesis.
		stream, err := agent.chatModel.Stream(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("Failed to synthesize final answer: %w", err)
		}

		var finalAnswer strings.Builder
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}

			finalAnswer.WriteString(chunk.Content)

			// Send synthesis content in real-time.
			agent.sendThought(state, &StreamingThought{
				Timestamp: time.Now(),
				Stage:     StageSynthesizing,
				Content:   chunk.Content,
				Action:    ActionRealtimeSynthesis,
			})
		}

		// Update the final answer.
		state.FinalAnswer = finalAnswer.String()
		state.IsComplete = true

		logging.Infof("Final answer synthesis complete")
		return state, nil
	}
}

// createIncrementIterationNode creates a node that increments the research iteration count.
// This is an iteration control node that manages the multi-turn research process.
// It works with branch conditions to implement intelligent iteration control and termination.
// Returns a function that performs the node's logic by incrementing the iteration counter.
func (agent *StreamingResearchAgent) createIncrementIterationNode() func(context.Context, *StreamingResearchState) (*StreamingResearchState, error) {
	return func(ctx context.Context, state *StreamingResearchState) (*StreamingResearchState, error) {
		state.CurrentIteration++

		agent.sendThought(state, &StreamingThought{
			Timestamp: time.Now(),
			Stage:     StageThinking,
			Content:   fmt.Sprintf("Entering iteration %d, continuing research.", state.CurrentIteration),
			Action:    ActionIterationIncrement,
		})

		logging.Infof("Iteration count increased to: %d", state.CurrentIteration)
		return state, nil
	}
}

// isSimilarQuestionResearched checks if a new question is similar to one that has already been researched.
// It uses a simple word overlap algorithm to judge the similarity between the new question and researched ones.
//
// Parameters:
//   - newQuestion: The new question to check.
//   - researchedQuestions: The set of already researched questions.
//
// Returns:
//   - bool: true if a similar question exists, false otherwise.
func (agent *StreamingResearchAgent) isSimilarQuestionResearched(newQuestion string, researchedQuestions map[string]bool) bool {
	newQuestionLower := strings.ToLower(newQuestion)

	for researchedQuestion := range researchedQuestions {
		researchedLower := strings.ToLower(researchedQuestion)

		// Simple similarity check.
		newWords := strings.Fields(newQuestionLower)
		researchedWords := strings.Fields(researchedLower)

		commonWords := 0
		for _, newWord := range newWords {
			if len(newWord) > MinWordLength {
				for _, researchedWord := range researchedWords {
					if newWord == researchedWord {
						commonWords++
						break
					}
				}
			}
		}

		// If the overlap exceeds the similarity threshold, consider it similar.
		if len(newWords) > 0 && float64(commonWords)/float64(len(newWords)) > SimilarityThreshold {
			return true
		}
	}

	return false
}

// sendThought sends a thought to the streaming channel.
// It sends the thought non-blockingly; if the channel is full, it skips sending.
//
// Parameters:
//   - state: The current research state, which contains the thought channel.
//   - thought: The thought content to be sent.
func (agent *StreamingResearchAgent) sendThought(state *StreamingResearchState, thought *StreamingThought) {
	if state.ThoughtChannel != nil {
		select {
		case state.ThoughtChannel <- thought:
			// Successfully sent.
		default:
			// Channel is full or closed, log but do not block.
			logging.Infof("Thought channel is full or closed")
		}
	}
}

// calculateMaxQuestions calculates the maximum number of sub-questions based on the maximum steps.
//
// Parameters:
//   - maxSteps: The maximum number of steps.
//   - currentQuestionCount: The current number of questions.
//
// Returns:
//   - maxTotalQuestions: The total maximum number of questions.
//   - maxNewQuestions: The maximum number of new questions that can be added.
func calculateMaxQuestions(maxSteps, currentQuestionCount int) (int, int) {
	// Reserve some steps for generating questions, iterating, synthesizing, etc.
	reservedSteps := 5
	availableSteps := maxSteps - reservedSteps

	// Ensure there are enough steps to perform basic operations.
	if availableSteps < StepsPerQuestion {
		return 0, 0
	}

	maxTotalQuestions := availableSteps / StepsPerQuestion
	maxNewQuestions := maxTotalQuestions - currentQuestionCount

	if maxNewQuestions < 0 {
		maxNewQuestions = 0
	}

	return maxTotalQuestions, maxNewQuestions
}

// extractSources extracts the sources of information.
// It collects the source URLs from all completed research questions for citation in the final answer.
//
// Parameters:
//   - state: The current research state, containing all research questions.
//
// Returns:
//   - []string: A deduplicated list of source URLs.
func (agent *StreamingResearchAgent) extractSources(state *StreamingResearchState) []string {
	var sources []string

	for _, q := range state.ResearchQuestions {
		if q.Status == QuestionStatusCompleted {
			for _, searchResult := range q.SearchResults {
				for _, result := range searchResult.Results {
					if result.URL != "" {
						sources = append(sources, result.URL)
					}
				}
			}
		}
	}

	return sources
}
