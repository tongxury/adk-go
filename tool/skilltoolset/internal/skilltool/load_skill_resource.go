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

package skilltool

import (
	"fmt"
	"io"

	"github.com/google/jsonschema-go/jsonschema"
	adk "google.golang.org/adk/v2"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
	"google.golang.org/adk/v2/tool/skilltoolset/skill"
)

const maxResourceSize = 10 * 1024 * 1024 // 10 MiB

// LoadSkillResourceArgs represents the input for retrieving a resource out of a skill's resources.
type LoadSkillResourceArgs struct {
	SkillName    string `json:"skill_name"`
	ResourcePath string `json:"resource_path"`
}

// LoadSkillResourceResult encapsulates the resource content.
type LoadSkillResourceResult struct {
	SkillName string `json:"skill_name,omitempty"`
	Path      string `json:"path,omitempty"`
	Content   string `json:"content,omitempty"`
}

// LoadSkillResource creates a tool.Tool to load a resource file for a skill.
func LoadSkillResource(source skill.Source) (tool.Tool, error) {
	// 结构体标签只负责字段名；字段描述统一从 adk 常量注入，避免提示词散落在 tag 中。
	inputSchema, err := jsonschema.For[LoadSkillResourceArgs](nil)
	if err != nil {
		return nil, fmt.Errorf("create load skill resource input schema: %w", err)
	}
	inputSchema.Properties["skill_name"].Description = adk.LoadSkillNameDescription
	inputSchema.Properties["resource_path"].Description = adk.LoadSkillResourcePathDescription

	return functiontool.New(
		functiontool.Config{
			Name:        "load_skill_resource",
			Description: adk.LoadSkillResourceToolDescription,
			InputSchema: inputSchema,
		},
		func(ctx agent.Context, args LoadSkillResourceArgs) (*LoadSkillResourceResult, error) {
			return loadSkillResource(ctx, args, source)
		},
	)
}

func loadSkillResource(ctx agent.Context, args LoadSkillResourceArgs, source skill.Source) (*LoadSkillResourceResult, error) {
	if args.SkillName == "" {
		return nil, fmt.Errorf("skill name is required to load a resource")
	}
	if args.ResourcePath == "" {
		return nil, fmt.Errorf("resource path is required to load a resource for skill %q", args.SkillName)
	}
	reader, err := source.LoadResource(ctx, args.SkillName, args.ResourcePath)
	if err != nil {
		return nil, fmt.Errorf("load resource '%s' from skill '%s': %w", args.ResourcePath, args.SkillName, err)
	}
	defer func() {
		_ = reader.Close()
	}()
	content, err := io.ReadAll(io.LimitReader(reader, maxResourceSize+1))
	if err != nil {
		return nil, fmt.Errorf("read resource '%s' from skill '%s': %w", args.ResourcePath, args.SkillName, err)
	}
	if int64(len(content)) > maxResourceSize {
		return nil, fmt.Errorf("resource '%s' from skill '%s' is too large (limit: %d bytes)", args.ResourcePath, args.SkillName, maxResourceSize)
	}
	return &LoadSkillResourceResult{
		SkillName: args.SkillName,
		Path:      args.ResourcePath,
		Content:   string(content),
	}, nil
}
