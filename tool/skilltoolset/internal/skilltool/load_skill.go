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

// Package skilltool provides the standard tools for the skill toolset.
package skilltool

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	adk "google.golang.org/adk/v2"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
	"google.golang.org/adk/v2/tool/skilltoolset/skill"
)

// LoadSkillArgs represents the input to load a skill.
type LoadSkillArgs struct {
	Name string `json:"name"`
}

type FrontmatterJSON struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  []string          `json:"allowed-tools,omitempty"`
}

// LoadSkillResult represents the output of a loaded skill.
type LoadSkillResult struct {
	SkillName    string           `json:"skill_name,omitempty"`
	Instructions string           `json:"instructions,omitempty"`
	Frontmatter  *FrontmatterJSON `json:"frontmatter,omitempty"`
}

// LoadSkill creates a tool.Tool to load a skill's instructions.
func LoadSkill(source skill.Source) (tool.Tool, error) {
	// 结构体标签只负责字段名；字段描述统一从 adk 常量注入，避免提示词散落在 tag 中。
	inputSchema, err := jsonschema.For[LoadSkillArgs](nil)
	if err != nil {
		return nil, fmt.Errorf("create load skill input schema: %w", err)
	}
	inputSchema.Properties["name"].Description = adk.LoadSkillNameToLoadDescription

	return functiontool.New(
		functiontool.Config{
			Name:        "load_skill",
			Description: adk.LoadSkillToolDescription,
			InputSchema: inputSchema,
		},
		func(ctx agent.Context, args LoadSkillArgs) (*LoadSkillResult, error) {
			return loadSkill(ctx, args, source)
		},
	)
}

func loadSkill(ctx agent.Context, args LoadSkillArgs, source skill.Source) (*LoadSkillResult, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("skill name is required to load a skill")
	}
	frontmatter, err := source.LoadFrontmatter(ctx, args.Name)
	if err != nil {
		return nil, fmt.Errorf("load frontmatter for skill %q: %w", args.Name, err)
	}
	instructions, err := source.LoadInstructions(ctx, args.Name)
	if err != nil {
		return nil, fmt.Errorf("load instructions for skill %q: %w", args.Name, err)
	}
	return &LoadSkillResult{
		SkillName:    args.Name,
		Instructions: instructions,
		Frontmatter: &FrontmatterJSON{
			Name:          frontmatter.Name,
			Description:   frontmatter.Description,
			License:       frontmatter.License,
			Compatibility: frontmatter.Compatibility,
			Metadata:      frontmatter.Metadata,
			AllowedTools:  frontmatter.AllowedTools,
		},
	}, nil
}
