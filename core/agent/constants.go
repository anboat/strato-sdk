package agent

// Action represents the type of action performed by the agent.
type Action string

// Constants definition
const (
	// Stage constants represent the different stages of the research process.
	StageThinking     = "thinking"     // Thinking stage
	StageSearching    = "searching"    // Searching stage
	StageAnalyzing    = "analyzing"    // Analyzing stage
	StageSynthesizing = "synthesizing" // Synthesizing stage
	StageCompleted    = "completed"    // Completed stage
	StageError        = "error"        // Error stage

	// Action constants represent specific actions within the agent's workflow.
	ActionGenerateQuestions    Action = "generate_questions"
	ActionQuestionSelection    Action = "question_selection"
	ActionNetworkSearch        Action = "network_search"
	ActionSearchComplete       Action = "search_complete"
	ActionWebScraping          Action = "web_scraping"
	ActionSkipScraping         Action = "skip_scraping"
	ActionScrapingComplete     Action = "scraping_complete"
	ActionContentAnalysis      Action = "content_analysis"
	ActionRealtimeAnalysis     Action = "realtime_analysis"
	ActionAnalysisComplete     Action = "analysis_complete"
	ActionSynthesisAnalysis    Action = "synthesis_analysis"
	ActionRealtimeSynthesis    Action = "realtime_synthesis"
	ActionIterationIncrement   Action = "iteration_increment"
	ActionProgressCheck        Action = "progress_check"
	ActionIterationComplete    Action = "iteration_complete"
	ActionModelJudgeSufficient Action = "model_judge_sufficient"
	ActionContinueResearch     Action = "continue_research"
	ActionGenerateNewQuestions Action = "generate_new_questions"
	ActionPrepareSynthesis     Action = "prepare_synthesis"
	ActionStepAllocation       Action = "step_allocation"
	ActionQuestionLimitReached Action = "question_limit_reached"
	ActionQuestionGenComplete  Action = "question_gen_complete"
	ActionResearchComplete     Action = "research_complete"
	ActionError                Action = "error"

	// Question status constants.
	QuestionStatusPending     = "pending"     // Pending research
	QuestionStatusResearching = "researching" // Currently researching
	QuestionStatusCompleted   = "completed"   // Research completed

	// Eino node name constants.
	NodeStartThinking         = "start_thinking"
	NodeGenerateQuestions     = "generate_questions"
	NodeSelectQuestion        = "select_question"
	NodeSearchQuestion        = "search_question"
	NodeScrapeWebContent      = "scrape_web_content"
	NodeAnalyzeQuestion       = "analyze_question"
	NodeSynthesizeFinalAnswer = "synthesize_final_answer"
	NodeIncrementIteration    = "increment_iteration"

	// Workflow graph name.
	GraphNameStreamingResearch = "StreamingResearchGraph"

	// Workflow step calculation constant.
	// Steps required for researching each sub-question: selectQuestion(1) + selectBranch(1) + searchQuestion(1) + scrapeWebContent(1) + analyzeQuestion(1) + checkCompletion(1)
	StepsPerQuestion = 6

	// Question ID prefix.
	QuestionIDPrefix = "q_"

	// Similarity threshold constants.
	SimilarityThreshold = 0.5 // Threshold for question similarity judgment.
	MinWordLength       = 2   // Minimum word length.
)
