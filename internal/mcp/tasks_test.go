package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/freepik-company/jev-mcp/internal/systemone"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTaskContracts(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		ids         []string
		field       string
		values      []any
	}{
		{"classify", `{"items":[{"id":"b","text":"Refund please"},{"id":"a","text":"App crashes"}],"categories":{"billing":"Payments","technical":"Bugs"},"instructions":"Route tickets","model":"pinned"}`, []string{"b", "a"}, "category", []any{"billing", "technical"}},
		{"verify", `{"claims":[{"id":"b","text":"Sky is blue"},{"id":"a","text":"Price is 10"}],"evidence":[{"id":"source","text":"Price is 12"}]}`, []string{"b", "a"}, "verdict", []any{"insufficient_evidence", "contradicted"}},
		{"rerank", `{"query":"Refunds","candidates":[{"id":"b","text":"Other"},{"id":"a","text":"Refunds"},{"id":"c","text":"Refund policy"}],"top_k":2}`, []string{"a", "c"}, "score", []any{float64(4), float64(4)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			session := testSession(t, func(req *http.Request) (*http.Response, error) {
				calls++
				var in systemone.Request
				if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
					return nil, err
				}
				if tc.name == "classify" && in.Model != "pinned" {
					t.Error("model was not propagated")
				}
				expectedCount := 2
				if tc.name == "rerank" {
					expectedCount = 3
				}
				if len(in.Questions) != expectedCount {
					t.Fatalf("preguntas perdidas: %d", len(in.Questions))
				}
				answers := map[string]any{}
				for id, q := range in.Questions {
					instruction, ok := q.Instructions.(map[string]any)
					if !ok || instruction["item_id"] != id {
						t.Fatal("question detached from its item")
					}
					answer := map[string]any{"type": q.Type, "confidence": 0.9}
					probabilities := map[string]float64{}
					if q.Type == "choice" {
						var criteria map[string]string
						if err := json.Unmarshal(q.Criteria, &criteria); err != nil {
							return nil, err
						}
						choice := "billing"
						if id == "a" {
							choice = "technical"
						}
						if tc.name == "verify" {
							choice = "insufficient_evidence"
							if id == "a" {
								choice = "contradicted"
							}
						}
						for key := range criteria {
							probabilities[key] = 0
						}
						if _, ok := criteria[choice]; !ok {
							t.Fatal("task categories are missing")
						}
						probabilities[choice] = 1
						answer["choice"] = choice
					} else {
						var criteria []string
						if err := json.Unmarshal(q.Criteria, &criteria); err != nil {
							return nil, err
						}
						legend := map[string]string{}
						for i, value := range criteria {
							key := fmt.Sprint(i)
							legend[key] = value
							probabilities[key] = 0
						}
						score := 4
						if id == "b" {
							score = 0
						}
						probabilities[fmt.Sprint(score)] = 1
						answer["score"], answer["legend"] = score, legend
					}
					answer["probabilities"] = probabilities
					answers[id] = answer
				}
				raw, _ := json.Marshal(map[string]any{"model": "jev-test", "answers": answers, "usage": map[string]any{"input_tokens": 10, "output_tokens": 5, "cost": 0.001}, "extension": map[string]any{"trace": "kept"}})
				return response(200, string(raw)), nil
			})
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.name, Arguments: json.RawMessage(tc.input)})
			if err != nil || result.IsError {
				t.Fatalf("%v: %+v", err, result)
			}
			encoded, _ := json.Marshal(result.StructuredContent)
			var out struct {
				Results  []map[string]any `json:"results"`
				Response struct {
					Answers map[string]any `json:"answers"`
					Usage   struct {
						Cost float64 `json:"cost"`
					} `json:"usage"`
					Extension map[string]string `json:"extension"`
				} `json:"response"`
			}
			if err := json.Unmarshal(encoded, &out); err != nil {
				t.Fatal(err)
			}
			if len(out.Results) != len(tc.ids) {
				t.Fatalf("resultados: %s", encoded)
			}
			for i, row := range out.Results {
				if row["id"] != tc.ids[i] || row[tc.field] != tc.values[i] {
					t.Errorf("orden o correspondencia incorrectos: %+v", row)
				}
			}
			if calls != 1 || out.Response.Usage.Cost != 0.001 || out.Response.Extension["trace"] != "kept" {
				t.Fatal("llamadas extra o metadatos perdidos")
			}
			if tc.name == "rerank" && len(out.Response.Answers) != 3 {
				t.Fatal("top_k truncated the original response")
			}
		})
	}
}

func TestInvalidTasksNeverCallProvider(t *testing.T) {
	calls := 0
	session := testSession(t, func(*http.Request) (*http.Response, error) { calls++; return response(500, "unexpected"), nil })
	for _, tc := range []struct{ name, input string }{
		{"classify", `{"items":[{"id":"x","text":"one"},{"id":"x","text":"two"}],"categories":{"a":"A","b":"B"},"instructions":"Choose"}`},
		{"classify", `{"items":[],"categories":{"a":"A","b":"B"},"instructions":"Choose"}`},
		{"classify", `{"items":[{"id":"x","text":"one","ignored":true}],"categories":{"a":"A","b":"B"},"instructions":"Choose"}`},
		{"classify", `{"items":[{"id":"x","text":"one"}],"categories":{"a":"A"},"instructions":"Choose"}`},
		{"classify", `{"items":[{"id":"x","text":"one"}],"categories":{"a":"A","b":"B"},"instructions":" "}`},
		{"verify", `{"claims":[{"id":"a","text":"one"}],"evidence":[]}`},
		{"verify", `{"claims":[{"id":"a","text":"one"}],"evidence":[{"id":"e","text":"one"},{"id":"e","text":"two"}]}`},
		{"verify", `{"claims":[{"id":"a","text":"one"}],"evidence":[{"id":"e","text":" "}]}`},
		{"rerank", `{"query":"q","candidates":[{"id":"a","text":"one"}],"top_k":2}`},
		{"rerank", `{"query":"q","candidates":[{"id":"a","text":"one"}],"top_k":0}`},
		{"rerank", `{"query":"q","candidates":[{"id":"a","text":"one"}],"top_k":null}`},
		{"rerank", `{"query":" ","candidates":[{"id":"a","text":"one"}]}`},
		{"list_models", `{"api_key":"override"}`},
	} {
		t.Run(tc.name+tc.input, func(t *testing.T) {
			out, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.name, Arguments: json.RawMessage(tc.input)})
			if err == nil && !out.IsError {
				t.Fatal("accepted invalid input")
			}
		})
	}
	if calls != 0 {
		t.Fatalf("%d invalid paid calls", calls)
	}
}

func TestTaskFailureAndInputPreservation(t *testing.T) {
	longText := strings.Repeat("Evidence data. ", 1000)
	var received systemone.Request
	session := testSession(t, func(req *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(req.Body).Decode(&received); err != nil {
			return nil, err
		}
		return response(200, `{"model":"jev","answers":{},"usage":{"input_tokens":1,"output_tokens":1}}`), nil
	})
	input := map[string]any{"claims": []map[string]string{{"id": "claim", "text": "test"}}, "evidence": []map[string]string{{"id": "source", "text": longText}}}
	out, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "verify", Arguments: input})
	if err == nil && !out.IsError {
		t.Fatal("missing answers were presented as a verification")
	}
	raw, _ := json.Marshal(received.State)
	var state struct {
		Evidence []struct {
			Text string `json:"text"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Evidence) != 1 || state.Evidence[0].Text != longText {
		t.Fatal("evidencia truncada o modificada")
	}
}

func TestFiveToolsAndModelDiscovery(t *testing.T) {
	session := testSession(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || req.URL.Path != "/api/v1/models" || req.URL.Query().Get("output_modalities") != "decisions" {
			t.Fatal("wrong catalogue path")
		}
		return response(200, `{"data":[{"id":"typesafe/jev","name":"Jev","architecture":{"output_modalities":["decisions"]}}]}`), nil
	})
	list, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
		if tool.OutputSchema == nil {
			t.Fatal("output contract is missing")
		}
	}
	if !reflect.DeepEqual(names, []string{"classify", "decide", "list_models", "rerank", "verify"}) {
		t.Fatalf("unexpected catalogue: %v", names)
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_models", Arguments: map[string]any{}})
	if err != nil || result.IsError {
		t.Fatalf("%v: %+v", err, result)
	}
}
