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

// Package exampletool provides a tool that allows an agent to add (few-shot) examples to the LLM request.
package exampletool

import (
	"fmt"
	"strings"

	"google.golang.org/genai"

	adk "google.golang.org/adk/v2"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/internal/utils"
	"google.golang.org/adk/v2/model"
)

// Example is a single few-shot example, pairing an input with its expected output.
type Example struct {
	Input  *genai.Content   `json:"input"`
	Output []*genai.Content `json:"output"`
}

// ExampleToolConfig configures an example tool with the few-shot examples to add.
type ExampleToolConfig struct {
	Examples []*Example
}

// exampleTool is a tool that adds (few-shot) examples to the LLM request.
type exampleTool struct {
	examples []*Example
}

// New creates an example tool that adds the configured few-shot examples to the
// LLM request.
func New(config ExampleToolConfig) (*exampleTool, error) {
	return &exampleTool{examples: config.Examples}, nil
}

// Name implements tool.Tool.
func (s exampleTool) Name() string {
	return "example_tool"
}

// Description implements tool.Tool.
func (s exampleTool) Description() string {
	return adk.ExampleToolDescription
}

// ProcessRequest adds the exampleTool examples to the LLM request.
func (s exampleTool) ProcessRequest(ctx agent.Context, req *model.LLMRequest) error {
	userContent := ctx.UserContent()
	if userContent == nil {
		return nil
	}
	parts := userContent.Parts
	if len(parts) == 0 || parts[0] == nil || parts[0].Text == "" {
		return nil
	}

	instruction := buildExamplesSystemInstruction(s.examples, req.Model)
	utils.AppendInstructions(req, instruction)
	return nil
}

// IsLongRunning implements tool.Tool.
func (t exampleTool) IsLongRunning() bool {
	return false
}

// Converts a list of examples to a string that can be used in a system instruction.
func buildExamplesSystemInstruction(examples []*Example, model string) string {
	var sb strings.Builder
	sb.WriteString(adk.ExamplesIntro)
	for exampleNum, example := range examples {
		fmt.Fprintf(&sb, adk.ExampleStart, exampleNum+1)
		sb.WriteString(adk.ExampleUserPrefix)
		if example.Input != nil && len(example.Input.Parts) > 0 {
			for _, part := range example.Input.Parts {
				if part.Text != "" {
					safeText := strings.ReplaceAll(part.Text, adk.ExamplesEndMarker, "[PROTECTED]")
					sb.WriteString(safeText)
					sb.WriteString("\n")
				}
			}
		}
		gemini2 := strings.Contains(model, "gemini-2")
		previousRole := ""
		for _, content := range example.Output {
			var role string
			if content.Role == "model" {
				role = adk.ExampleModelPrefix
			} else {
				role = adk.ExampleUserPrefix
			}
			if role != previousRole {
				sb.WriteString(role)
			}
			previousRole = role
			for _, part := range content.Parts {
				if part.FunctionCall != nil {
					args := []string{}
					for k, v := range part.FunctionCall.Args {
						if _, ok := v.(string); ok {
							args = append(args, fmt.Sprintf("%s='%s'", k, v))
						} else {
							args = append(args, fmt.Sprintf("%s=%v", k, v))
						}
					}
					prefix := adk.ExampleFunctionPrefix
					if gemini2 {
						prefix = adk.ExampleFunctionCallPrefix
					}
					fmt.Fprintf(&sb, "%s%s(%s)%s", prefix, part.FunctionCall.Name, strings.Join(args, ", "), adk.ExampleFunctionCallSuffix)
				} else if part.FunctionResponse != nil {
					prefix := adk.ExampleFunctionPrefix
					if gemini2 {
						prefix = adk.ExampleFunctionResponsePrefix
					}
					fmt.Fprintf(&sb, "%s%v%s", prefix, part.FunctionResponse, adk.ExampleFunctionResponseSuffix)
				} else if part.Text != "" {
					// SANITIZATION: Again, protect the boundary tags
					safeText := strings.ReplaceAll(part.Text, adk.ExamplesEndMarker, "[PROTECTED]")
					sb.WriteString(safeText)
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString(adk.ExampleEnd)
	}
	sb.WriteString(adk.ExamplesEnd)
	return sb.String()
}
