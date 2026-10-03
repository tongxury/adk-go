// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package adk 集中保存 ADK 运行时内置、且会发送给模型的固定文案。
//
// 这里不放业务方动态传入的 system instruction，也不放普通错误消息；
// 只收敛框架自身的提示词、工具描述和模型交互协议标记，便于统一审查
// token 成本和提示词行为，避免不同工具实现各自维护一份相同文案。
package adk

const (
	// ConversationHistoryPlaceholder 会在组装默认压缩提示词时替换成会话文本。
	ConversationHistoryPlaceholder = "{conversation_history}"

	// DefaultCompactionPromptTemplate 是调用方未提供模板时，会话压缩器使用的默认提示词。
	DefaultCompactionPromptTemplate = "The following is a conversation history between a user and an AI agent." +
		" It may or may not start from a compacted history. Please identify and" +
		" reiterate the user request, summarize the context so far, focusing on" +
		" key decisions made and information obtained, as well as any unresolved" +
		" questions or tasks. " +
		"CRITICAL INSTRUCTIONS: " +
		"1. Explicitly identify and state the primary language used by the user " +
		`at the top of your summary (e.g., "Conversation Language: English"). ` +
		"2. If the agent called any tools, accurately list the exact tool names " +
		"used to maintain tool grounding. " +
		`3. Maintain a section titled "Durable facts" listing every concrete ` +
		"detail the user has stated: identifiers, names, dates, numbers, chosen " +
		"options and the reasons given for them. Copy each one verbatim. This " +
		"history may already be a summary of a summary, so any durable fact " +
		"present in the history MUST be carried forward unchanged, even if it is " +
		"old and the recent turns are about something else. Never drop or " +
		"generalize a durable fact to save space; drop narrative instead. " +
		"The rest of the summary should be concise and capture the" +
		" essence of the interaction.\n\n" + ConversationHistoryPlaceholder

	// LoadMemoryInstructions 说明模型何时应该调用显式记忆搜索工具。
	LoadMemoryInstructions = `You have memory. You can use it to answer questions. If any questions need
you to look up the memory, you should call load_memory function with a query.`
	LoadMemoryToolDescription    = "Loads the memory for the current user."
	LoadMemoryQueryDescription   = "The query to search memory for."
	PreloadMemoryToolDescription = "Preloads relevant memory for the current user."
	PreloadMemoryInstructions    = `The following content is from your previous conversations with the user.
They may be useful for answering the user's current query.
<PAST_CONVERSATIONS>
%s
</PAST_CONVERSATIONS>`

	// LoadArtifactsInstructionsTemplate 会先填入可用 artifact 名称的 JSON 列表，
	// 再追加到模型请求中。
	LoadArtifactsToolDescription      = "Loads the artifacts and adds them to the session."
	LoadArtifactsInstructionsTemplate = "You have a list of artifacts:\n  %s\n\nWhen the user asks questions about" +
		" any of the artifacts, you should call the `load_artifacts` function" +
		" to load the artifact. Do not generate any text other than the" +
		" function call. Whenever you are asked about artifacts, you" +
		" should first load it. You must always load an artifact to access its" +
		" content, even if it has been loaded before."

	// ExitLoopToolDescription 是控制流工具暴露给模型的描述。
	ExitLoopToolDescription = "Exits the loop.\n\nCall this function only when you are instructed to do so.\n"

	// LongRunningToolInstruction 会追加到长耗时工具声明中，避免中间状态未结束时重复调用。
	LongRunningToolInstruction = "NOTE: This is a long-running operation. Do not call this tool again if it has already returned some intermediate or pending status."

	// ToolConfirmationPromptTemplate 是普通函数工具、流式函数工具和 MCP 工具共用的确认提示词。
	ToolConfirmationPromptTemplate = "Please approve or reject the tool call %s() by responding with a FunctionResponse with an expected ToolConfirmation payload."

	// TransferToAgentToolDescription 及其关联常量定义内置 agent 转移工具暴露给模型的文案。
	TransferToAgentToolDescription     = "Transfer the question to another agent.\nThis tool hands off control to another agent when it's more suitable to answer the user's question according to the agent's description."
	TransferToAgentNameDescription     = "the agent name to transfer to"
	TransferToAgentInstructionTemplate = `
You have a list of other agents to transfer to:

{{range .Targets}}
Agent name: {{.Name}}
Agent description: {{.Description}}

{{end}}
If you are the best to answer the question according to your description,
you can answer it.

If another agent is better for answering the question according to its
description, call ` + "`{{.ToolName}}`" + ` function to transfer the question to that
agent. When transferring, do not generate any text other than the function
call.

**NOTE**: the only available agents for ` + "`{{.ToolName}}`" + ` function are
{{.FormattedTargets}}.
{{if .Parent}}
If neither you nor the other agents are best for the question, transfer to your parent agent {{.Parent.Name}}.
{{end}}`

	// OutputSchemaInstruction 说明需要结构化输出时，模型应如何结束请求。
	OutputSchemaInstruction = "IMPORTANT: You have access to other tools, but you must provide " +
		"your final response using the set_model_response tool with the " +
		"required structured format. After using any other tools needed " +
		"to complete the task, always call set_model_response with your " +
		"final answer in the specified schema format."
	SetModelResponseToolDescription = "Set your final response using the required output schema. Use this tool to provide your final structured answer instead of outputting text directly."

	// FinishTask 相关常量控制委派任务完成、参数校验失败反馈和成功结果。
	FinishTaskToolDescription   = "Signal that this agent has completed its delegated task. Call this when you have finished your delegated task."
	FinishTaskOutputDescription = "A brief summary of what the agent accomplished."
	FinishTaskOutputDataSuffix  = " Pass the required output data in the parameters."
	FinishTaskInstruction       = `
Do NOT call 'finish_task' prematurely. Use your available tools to
fully complete every aspect of the delegated task first. If the
task is unclear, ask the user for clarification before proceeding.
Once the task is fully complete, call 'finish_task' by itself with
no accompanying text output.
`
	FinishTaskValidationErrorTemplate = "Invoking `%s()` failed due to validation errors:\n%s\nYou could retry calling this tool, but it is IMPORTANT for you to provide all the mandatory parameters with correct types."
	FinishTaskSuccessResult           = "Task completed."

	// TaskAgentDelegationInstruction 会追加到委派 agent 的工具声明，禁止与其他工具并行委派。
	TaskAgentDelegationInstruction = "IMPORTANT: This tool delegates execution to a specialized agent. Do NOT call this tool in parallel with any other tools."
	TaskAgentRequestDescription    = "Detailed instructions or context for the task sub-agent."

	// SkillToolsetDefaultSystemInstruction 是技能发现、加载工具的默认系统约束。
	SkillToolsetDefaultSystemInstruction = "You can use specialized 'skills' to help you with complex tasks. You MUST use the skill tools to interact with these skills.\n\n" +
		"Skills are folders of instructions and resources that extend your capabilities for specialized tasks. Each skill folder contains:\n" +
		"- **SKILL.md** (required): The main instruction file with skill metadata and detailed markdown instructions.\n" +
		"- **references/** (Optional): Additional documentation or examples for skill usage.\n" +
		"- **assets/** (Optional): Templates, scripts or other resources used by the skill.\n" +
		"- **scripts/** (Optional): Executable scripts that can be run via bash.\n\n" +
		"This is very important:\n\n" +
		"1. If a skill seems relevant to the current user query, you MUST use the `load_skill` tool with `name=\"<SKILL_NAME>\"` to read its full instructions before proceeding.\n" +
		"2. Once you have read the instructions, follow them exactly as documented before replying to the user. For example, If the instruction lists multiple steps, please make sure you complete all of them in order.\n" +
		"3. The `load_skill_resource` tool is for viewing files within a skill's directory (e.g., `references/*`, `assets/*`, `scripts/*`). Do NOT use other tools to access these files.\n"
	ListSkillsToolDescription        = "Lists all available skills with their names and descriptions."
	LoadSkillToolDescription         = "Loads the SKILL.md instructions for a given skill."
	LoadSkillNameDescription         = "The name of the skill."
	LoadSkillNameToLoadDescription   = "The name of the skill to load."
	LoadSkillResourceToolDescription = "Loads a resource file (e.g., from references/ or assets/) associated with the specified skill."
	LoadSkillResourcePathDescription = "The relative path to the resource (e.g., 'references/my_doc.md', 'assets/template.txt', or 'scripts/setup.sh')."

	// 下面这些内置工具描述和协议标记同样会进入模型请求，因此也统一放在这里维护。
	GoogleSearchToolDescription        = "Performs a Google search to retrieve information from the web."
	MCPToolsetDescription              = "Connects to a MCP Server, retrieves MCP Tools into ADK Tools."
	SequentialTaskCompletedDescription = "Signals that the agent has successfully completed the user's question or task."
	SequentialTaskCompletedInstruction = "\nIf you finished the user's request according to its description, call the task_completed function to exit so the next agents can take over. When calling this function, do not generate any text other than the function call."
	SequentialTaskCompletedResult      = "Task completion signaled."

	// IdentityInstructionTemplate 和 IdentityDescriptionTemplate 用于让模型知道
	// 当前运行的 agent 身份；具体名称和描述仍由运行时动态填充。
	IdentityInstructionTemplate = "You are an agent. Your internal name is %q."
	IdentityDescriptionTemplate = "The description about you is %q."

	ExampleToolDescription = "example tool"

	// Few-shot 标记属于 exampletool 生成的模型交互协议，不是普通日志文本。
	ExamplesIntro                 = "<EXAMPLES>\nBegin few-shot\nThe following are examples of user queries and model responses using the available tools.\n\n"
	ExamplesEnd                   = "End few-shot\n<EXAMPLES>"
	ExamplesEndMarker             = "End few-shot"
	ExampleStart                  = "EXAMPLE %d:\nBegin example\n"
	ExampleEnd                    = "End example\n\n"
	ExampleUserPrefix             = "[user]\n"
	ExampleModelPrefix            = "[model]\n"
	ExampleFunctionPrefix         = "```\n"
	ExampleFunctionCallPrefix     = "```tool_code\n"
	ExampleFunctionCallSuffix     = "\n```\n"
	ExampleFunctionResponsePrefix = "```tool_outputs\n"
	ExampleFunctionResponseSuffix = "\n```\n"
)
