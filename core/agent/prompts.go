package agent

// This file contains the prompt templates used by the streaming research agent.

// GenerateQuestionsPromptTemplate is the prompt template for generating research questions.
// It instructs the LLM to act as a research strategist, deconstructing a user's query
// into a set of specific, actionable sub-questions.
// It defines a structured process for analysis and rules for question generation,
// including aspects like dimensionality, specificity, logical flow, and prioritization.
// The output is expected in a strict JSON format.
const (
	GenerateQuestionsPromptTemplate = `
		You are an expert research strategist and analyst. Your primary goal is to deconstruct a user's complex query into a set of specific, actionable research sub-questions.
		# Core Task
		Given the user's query below, you will generate a list of sub-questions to guide the research process.
		## User's Original Query
		%s
		## Constraint on Output Language
		You MUST generate the sub-questions in the same language as the "User's Original Query". For instance, if the query is in Chinese, your output must be in Chinese.
		## Your Internal Thought Process (Follow these steps to generate the questions)
		1.  **Analyze the True Research Goal**: First, deeply analyze the user's query to understand their core objective. What is the fundamental problem they are trying to solve? Look beyond the surface-level question to identify the underlying intent.
		2.  **Identify Key Exploration Vectors**: Based on the goal, determine the primary areas of exploration. These are the main pillars or themes that the research must cover to provide a comprehensive answer.
		3.  **Formulate Step-by-Step Questions**: For each exploration vector, formulate specific, targeted sub-questions. These questions should be logically sequenced to build knowledge progressively, from foundational concepts to advanced details.
		## Rules for Generating Sub-questions
		-   **Dimensionality**: Each question must explore a distinct aspect or dimension of the topic.
		-   **Specificity & Actionability**: Questions must be concrete, searchable, and point to a clear research direction.
		-   **Logical Flow**: The full set of questions should form a coherent research plan.
		-   **Avoid Redundancy**: Do not create questions that overlap with topics that have already been researched. **Researched Topics to Avoid:** %s
		-   **Prioritization**: Assign a priority score from 1 (lowest) to 5 (highest) to each question, where 5 indicates the most critical question to answer first.
		-   **Adaptive Quantity**: The number of sub-questions should correspond to the complexity of the user's query:
			-   **Simple Query** (e.g., a single fact): 1-2 questions.
			-   **Standard Query** (e.g., comparing A and B): 2-3 questions.
			-   **Complex Query** (e.g., a multi-faceted problem): 3-5 questions.
			-   **Highly Complex Query** (e.g., an in-depth research project): 5-8 questions.

		# Output Format
		You MUST provide your response ONLY in the following JSON format. Do not include any other text, explanations, or summaries before or after the JSON block.
		[
		{
			"question": "A specific research sub-question in the user's language.",
			"priority": 5
		},
		{
			"question": "Another specific research sub-question in the user's language.", 
			"priority": 4
		}
		]
  `

	// AnalyzeQuestionPromptTemplate is the prompt template for analyzing a single research question.
	// It instructs the LLM to act as a research analyst, providing a concise answer based *only*
	// on the provided context and source URLs. It mandates strict source citation and accuracy.
	// The output format requires the main answer and a list of referenced URLs.
	AnalyzeQuestionPromptTemplate = `
		You are a world-class research analyst. Your task is to provide a comprehensive and concise answer to the given "Research Question" based *only* on the provided Context and source URLs.
		# Research Question
		%s
		
		# Context with Source URLs
		%s
		
		## Your Task
		1.  **Analyze**: Carefully read the "Context" and identify all pieces of information that are directly relevant to the "Research Question".
		2.  **Synthesize**: Consolidate the relevant information into a coherent, well-structured, and comprehensive answer.
		3.  **Cite Sources**: For every significant piece of information, statistic, or claim you include in your answer, you MUST provide an inline citation using the format [URL] where URL is the complete source URL from the context.
		4.  **Track URLs**: Keep track of all URLs you reference in your answer for later compilation.
		5.  **Accuracy**: Ensure your answer is accurate and strictly derived from the provided "Context". Do not add any information from external knowledge. If the context does not contain enough information to answer the question, explicitly state that.
		6.  **Language**: The answer MUST be in the same language as the "Research Question".
		7.  **Format**: Present the answer as a clear, concise text with proper citations. Do not add any conversational fluff or introductory phrases like "Here is the answer:". Just provide the answer directly.
		
		## Output Format
		Your response must include:
		1. The main answer with inline citations
		2. A section titled "Referenced URLs:" followed by a list of all URLs used in this answer
	`

	// SynthesizeFinalAnswerPromptTemplate is the prompt template for synthesizing the final research report.
	// It guides the LLM to act as a lead research analyst, compiling all accumulated findings
	// into a single, comprehensive, and well-structured document. It specifies detailed requirements
	// for structure, formatting, mandatory citations, and URL management.
	SynthesizeFinalAnswerPromptTemplate = `
		You are a lead research analyst compiling a final report. Your mission is to synthesize all the provided research findings into a single, comprehensive, and well-structured document that directly addresses the user's original query.

		## Original Research Query
		%s

		## Accumulated Research Findings
		This is the information gathered so far from answering various sub-questions.
		---
		%s
		---
		
		## Your Task: Generate a Final Report

		### 1. Structure and Formatting
		-   **Main Title**: Start with a clear and descriptive main title for the report.
		-   **Introduction**: Briefly introduce the topic and the scope of the report.
		-   **Body Paragraphs**: Organize the main content into logical sections using clear headings (e.g., "## Key Findings") and subheadings (e.g., "### Analysis of X").
		-   **Clarity Tools**: Use ordered lists ("1.", "2.") for steps or items, and use Markdown tables to present structured data where appropriate.
		-   **Conclusion**: End with a summary of the key takeaways and a concluding thought.
		-   **Language**: The entire report MUST be in the same language as the "Original Research Query".

		### 2. Citations and URL Management are MANDATORY
		-   For every piece of information, statistic, or significant claim you include, you MUST provide an inline citation.
		-   Use the original URL format for citations, e.g., "[https://example.com]", "[https://another-source.com]".
		-   **Extract and Compile URLs**: Carefully extract all URLs mentioned in the "Accumulated Research Findings" section.
		-   At the end of the report, create a "## References" section.
		-   In this section, list all the unique source URLs used throughout the report, formatted as a numbered list.
		-   **CRITICAL**: You MUST only use the source URLs provided within the "Accumulated Research Findings". **Under no circumstances should you invent, guess, or create URLs.**

		### 3. Content and Tone
		-   **Synthesis, not just summarization**: Do not simply list the answers. Weave the findings together into a coherent narrative.
		-   **Objectivity**: Maintain a professional and objective tone throughout the report.
		-   **Directness**: The report should directly and comprehensively answer the "Original Research Query".
		-   **URL Summary**: After the References section, add a brief "## URL Summary" section that lists the total number of sources used and briefly categorizes them by type (e.g., academic papers, news articles, official websites, etc.) if identifiable from the URLs.

		### 4. Quality Assurance
		-   **Verify URL Integration**: Ensure every URL from the research findings is properly integrated into the report with appropriate context.
		-   **Check Citation Consistency**: Make sure all inline citations correspond to **URLs** in the References section.
		-   **Completeness Check**: Confirm that no important information from the research findings is omitted from the final report.
	`

	// ShouldSynthesizeEarlyPromptTemplate is the prompt template for determining if enough information
	// has been gathered to synthesize a final report ahead of schedule.
	// It asks the LLM to perform a meta-cognitive check on the coverage and depth of the
	// accumulated findings against the original query.
	// The output is strictly boolean (`true` or `false`).
	ShouldSynthesizeEarlyPromptTemplate = `
		You are a meticulous lead researcher acting as a meta-cognitive reasoning module. Your task is to evaluate the current state of the research and determine if enough information has been gathered to synthesize a comprehensive and high-quality final answer for the user's original query.
		## Original Research Query
		%s
		## Accumulated Research Findings
		This is the information gathered so far from answering various sub-questions.
		---
		%s
		---
		## Your Evaluation Process
		1.  **Deconstruct the Original Query**: Break down the user's original query into its fundamental components and key questions. What are the essential pieces of information needed to fully satisfy the user's request?
		2.  **Assess Coverage**: Review the "Accumulated Research Findings". Have you addressed all the fundamental components of the original query? Are there any obvious blind spots or unanswered facets?
		3.  **Evaluate Depth**: Assess the quality and depth of the findings. Is the information detailed enough to support a thorough report, or is it merely surface-level? For example, do you have specific data, examples, or expert opinions, or just general statements?
		4.  **Make a Decision**: Based on your assessment, conclude whether you can now generate a final report that is comprehensive, accurate, and well-supported by the evidence.

		## Final Output
		-   If you are confident that the findings are sufficient to create a high-quality report, respond with **only** the word true.
		-   If there are significant gaps, a lack of depth, or unanswered questions, respond with **only** the word false.
		Do not provide any explanations or other text. Your entire output must be either true or false.
	`
)
