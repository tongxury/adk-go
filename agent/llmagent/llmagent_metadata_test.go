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

package llmagent_test

import (
	"context"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/internal/agent/runconfig"
	icontext "google.golang.org/adk/v2/internal/context"
	"google.golang.org/adk/v2/internal/testutil"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
)

// metadataNamedMock distinguishes parent and child models without making any
// network calls. Its embedded mock exercises the real streaming aggregator.
type metadataNamedMock struct {
	testutil.MockModel
	modelName string
}

func (m *metadataNamedMock) Name() string { return m.modelName }

type metadataLiveMock struct {
	metadataNamedMock
	client *genai.Client
}

func (m *metadataLiveMock) Client() *genai.Client { return m.client }

// Use a local Gemini Live protocol fake: this pins RunLive's annotation path
// without credentials, a real model, or outbound network traffic.
func TestLLMAgentLiveModelMetadata(t *testing.T) {
	var upgrader websocket.Upgrader
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			t.Error("failed to upgrade local live connection")
			return
		}
		defer func() { _ = conn.Close() }()
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		for _, frame := range []string{
			`{"setupComplete":{}}`,
			`{"serverContent":{"modelTurn":{"parts":[{"text":"done"}],"role":"model"},"turnComplete":true}}`,
		} {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
				return
			}
		}
		// Stay connected until the consumer closes; an early server disconnect
		// would trigger reconnection and obscure the single event under test.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI, APIKey: "local-test-key",
		HTTPOptions: genai.HTTPOptions{BaseURL: strings.Replace(ts.URL, "http", "ws", 1)},
	})
	if err != nil {
		t.Fatal(err)
	}
	a, err := llmagent.New(llmagent.Config{
		Name: "live_agent", Model: &metadataLiveMock{metadataNamedMock: metadataNamedMock{modelName: "live-model"}, client: client},
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := session.InMemoryService().Create(ctx, &session.CreateRequest{AppName: "app", UserID: "user"})
	if err != nil {
		t.Fatal(err)
	}
	// Live execution policy is carried on the internal context, separately from
	// the public text-run configuration exposed by InvocationContext.RunConfig.
	liveCtx := runconfig.ToContext(ctx, &runconfig.RunConfig{StreamingMode: runconfig.StreamingModeBidi, Live: &agent.LiveRunConfig{}})
	ic := icontext.NewInvocationContext(liveCtx, icontext.InvocationContextParams{Agent: a, Session: created.Session, RunConfig: &agent.RunConfig{}})
	liveAgent := a.(interface {
		RunLive(agent.InvocationContext) (agent.LiveSession, iter.Seq2[*session.Event, error], error)
	})
	live, events, err := liveAgent.RunLive(ic)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = live.Close() }()
	for ev, err := range events {
		if err != nil {
			t.Fatal(err)
		}
		if ev != nil && ev.Content != nil {
			if ev.Metadata["model"] != "live-model" {
				t.Error("live event lost its model annotation")
			}
			return
		}
	}
	t.Fatal("live fake emitted no model event")
}

// A callback can skip the model entirely. Its event still belongs to the LLM
// agent and must carry that agent's configured model, without invoking it.
func TestLLMAgentCallbackModelMetadata(t *testing.T) {
	m := &metadataNamedMock{modelName: "callback-model"}
	a, err := llmagent.New(llmagent.Config{
		Name: "callback_agent", Model: m,
		BeforeAgentCallbacks: []agent.BeforeAgentCallback{func(_ agent.Context) (*genai.Content, error) {
			return genai.NewContentFromText("cached", genai.RoleModel), nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for ev, err := range testutil.NewTestAgentRunner(t, a).Run(t, "callback-metadata", "start") {
		if err != nil {
			t.Fatal(err)
		}
		seen = true
		if ev == nil || ev.Metadata["model"] != "callback-model" {
			t.Error("callback event lost its configured model annotation")
		}
	}
	if !seen || len(m.Requests) != 0 {
		t.Fatal("callback did not short-circuit the model call")
	}
}

func TestLLMAgentEventModelMetadata(t *testing.T) {
	for _, mode := range []agent.StreamingMode{agent.StreamingModeNone, agent.StreamingModeSSE} {
		t.Run(string(mode), func(t *testing.T) {
			// Simulate a response model that differs from the configured model;
			// adding event annotations must not overwrite that response version.
			afterModel := []llmagent.AfterModelCallback{func(_ agent.Context, resp *model.LLMResponse, _ error) (*model.LLMResponse, error) {
				resp.ModelVersion = "response-model"
				return resp, nil
			}}
			childModel := &metadataNamedMock{modelName: "child-model", MockModel: testutil.MockModel{
				Responses: []*genai.Content{genai.NewContentFromText("done", genai.RoleModel)},
			}}
			child, err := llmagent.New(llmagent.Config{Name: "child", Model: childModel, AfterModelCallbacks: afterModel})
			if err != nil {
				t.Fatal(err)
			}
			parentModel := &metadataNamedMock{modelName: "parent-model", MockModel: testutil.MockModel{
				Responses: []*genai.Content{{Role: genai.RoleModel, Parts: []*genai.Part{{
					FunctionCall: &genai.FunctionCall{Name: "transfer_to_agent", Args: map[string]any{"agent_name": "child"}},
				}}}},
			}}
			parent, err := llmagent.New(llmagent.Config{
				Name: "parent", Model: parentModel, SubAgents: []agent.Agent{child}, AfterModelCallbacks: afterModel,
			})
			if err != nil {
				t.Fatal(err)
			}
			r := testutil.NewTestAgentRunner(t, parent)
			seenParent, seenChild, seenToolResult, seenPartial := false, false, false, false
			for ev, err := range r.RunContentWithConfig(t, "metadata-session", genai.NewContentFromText("start", genai.RoleUser), agent.RunConfig{StreamingMode: mode}) {
				if err != nil {
					t.Fatal(err)
				}
				if ev == nil {
					continue
				}
				want := "parent-model"
				if ev.Author == "child" {
					want = "child-model"
					seenChild = true
				} else {
					seenParent = true
				}
				if ev.Metadata["model"] != want {
					t.Error("event annotation does not match its author's model")
				}
				seenPartial = seenPartial || ev.Partial
				if ev.Content != nil {
					for _, part := range ev.Content.Parts {
						seenToolResult = seenToolResult || part.FunctionResponse != nil
					}
					if ev.Content.Role == genai.RoleModel && ev.ModelVersion != "response-model" {
						t.Error("response ModelVersion was changed")
					}
				}
			}
			if !seenParent || !seenChild || !seenToolResult || (mode == agent.StreamingModeSSE && !seenPartial) {
				t.Fatal("missing parent, child, tool result or streaming coverage")
			}
			// Reload through the session service to check the canonical record,
			// rather than relying only on annotations seen by the live consumer.
			saved, err := r.SessionService().Get(t.Context(), &session.GetRequest{AppName: "test_app", UserID: "test_user", SessionID: "metadata-session"})
			if err != nil {
				t.Fatal(err)
			}
			for ev := range saved.Session.Events().All() {
				if ev.Author != "user" && ev.Metadata["model"] == nil {
					t.Error("stored agent event lost its model annotation")
				}
			}
		})
	}
}
