package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/freepik-company/jev-mcp/internal/config"
	"github.com/freepik-company/jev-mcp/internal/systemone"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testInput = `{
  "state":{"ticket":"Please refund the duplicate charge."},
  "questions":{
    "department":{"type":"choice","instructions":"Which team?","criteria":{"billing":"Payments","technical":"Bugs"}},
    "refund":{"type":"noul","instructions":"Is a refund requested?"},
    "urgency":{"type":"score","instructions":"How urgent?","criteria":["Low","High"]}
  }
}`

const testOutput = `{
  "id":"decision-123","provider":"TypeSafe","model":"typesafe/jev-1.13",
  "answers":{
    "department":{"type":"choice","choice":"billing","confidence":0.95,"probabilities":{"billing":0.98,"technical":0.02}},
    "refund":{"type":"noul","noul":0.97},
    "urgency":{"type":"score","score":0.8,"confidence":0.7,"probabilities":{"0":0.2,"1":0.8},"legend":{"0":"Low","1":"High"}}
  },
  "usage":{"input_tokens":25,"output_tokens":5,"cost":0.00003}
}`

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func testSession(t *testing.T, transport roundTripFunc) *mcp.ClientSession {
	t.Helper()
	client := systemone.NewClient(config.Config{BaseURL: "https://openrouter.ai/api/", APIKey: "secret-for-test", Model: "jev-latest"}, transport)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(client).Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { serverSession.Close() })
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestDecideMCPContract(t *testing.T) {
	requests := make(chan *http.Request, 2)
	bodies := make(chan map[string]any, 2)
	session := testSession(t, func(req *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			return nil, err
		}
		requests <- req
		bodies <- body
		return response(200, testOutput), nil
	})
	ctx := context.Background()
	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 5 {
		t.Fatalf("catálogo inesperado: %+v", list.Tools)
	}
	for _, model := range []string{"", "typesafe/jev-1.13"} {
		var input map[string]any
		if err := json.Unmarshal([]byte(testInput), &input); err != nil {
			t.Fatal(err)
		}
		if model != "" {
			input["model"] = model
		}
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "decide", Arguments: input})
		if err != nil || result.IsError {
			t.Fatalf("decide: %v, %+v", err, result)
		}
		var want any
		if err := json.Unmarshal([]byte(testOutput), &want); err != nil {
			t.Fatal(err)
		}
		gotJSON, _ := json.Marshal(result.StructuredContent)
		wantJSON, _ := json.Marshal(want)
		if string(gotJSON) != string(wantJSON) {
			t.Fatalf("se perdió respuesta, metadatos o coste: %s", gotJSON)
		}
		var textOutput any
		if len(result.Content) == 1 {
			if content, ok := result.Content[0].(*mcp.TextContent); ok {
				if err := json.Unmarshal([]byte(content.Text), &textOutput); err != nil {
					t.Fatal(err)
				}
			}
		}
		textJSON, _ := json.Marshal(textOutput)
		if string(textJSON) != string(wantJSON) {
			t.Fatalf("falta el resultado de texto para clientes antiguos: %+v", result.Content)
		}
		req, body := <-requests, <-bodies
		if req.URL.String() != "https://openrouter.ai/api/v1/systemone" || req.Method != "POST" {
			t.Fatalf("ruta/protocolo incorrecto: %s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer secret-for-test" || req.Header.Get("Content-Type") != "application/json" {
			t.Fatal("faltan la autenticación o el tipo de contenido")
		}
		if model == "" {
			model = "jev-latest"
		}
		input["model"] = model
		gotBody, _ := json.Marshal(body)
		wantBody, _ := json.Marshal(input)
		if string(gotBody) != string(wantBody) {
			t.Fatalf("entrada alterada: %s", gotBody)
		}
	}
}

func TestInvalidInputNeverReachesProvider(t *testing.T) {
	var calls atomic.Int32
	session := testSession(t, func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return response(200, testOutput), nil
	})
	for _, input := range []string{
		`{}`,
		`{"state":"hello","questions":{}}`,
		`{"state":null,"questions":{"a":{"type":"noul","instructions":"Hello?"}}}`,
		`{"state":"hello","questions":{"a":{"type":"text","instructions":"Hello?"}}}`,
		`{"state":"hello","questions":{"a":{"type":"noul","instructions":""}}}`,
		`{"state":"hello","questions":{"a":{"type":"choice","instructions":"Hello?","criteria":{"a":"only"}}}}`,
		`{"state":"hello","questions":{"a":{"type":"score","instructions":"Hello?","criteria":{"a":"low","b":"high"}}}}`,
		`{"state":"hello","questions":{"a":{"type":"noul","instructions":"Hello?","criteria":{"true":"yes"}}}}`,
		`{"state":"hello","questions":{"a":{"type":"noul","instructions":"Hello?"}},"api_key":"do-not-accept"}`,
		`{"state":"hello","questions":{"a":{"type":"noul","instructions":"Hello?"}},"base_url":"https://attacker.test"}`,
		`{"state":"hello","questions":{"a":{"type":"noul","instructions":"Hello?","threshold_typo":0.99}}}`,
	} {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "decide", Arguments: json.RawMessage(input)})
		if err == nil && !result.IsError {
			t.Errorf("aceptó entrada inválida: %s", input)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("%d entradas inválidas llegaron al proveedor", calls.Load())
	}
}

func TestProviderFailuresAreErrorsWithoutSecretsOrRetries(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{"authentication", 401, "secret-for-test", nil},
		{"rate_limit", 429, "secret-for-test", nil},
		{"server", 503, "secret-for-test", nil},
		{"redirect", 307, "secret-for-test", nil},
		{"transport", 0, "", fmt.Errorf("Authorization: Bearer secret-for-test")},
		{"invalid_json", 200, "secret-for-test", nil},
		{"too_large", 200, strings.Repeat("x", (2<<20)+1), nil},
		{"missing_answer", 200, strings.Replace(testOutput, `"refund"`, `"different"`, 1), nil},
		{"wrong_type", 200, strings.Replace(testOutput, `"type":"noul"`, `"type":"choice"`, 1), nil},
		{"unoffered_choice", 200, strings.Replace(testOutput, `"choice":"billing"`, `"choice":"secret-for-test"`, 1), nil},
		{"out_of_range", 200, strings.Replace(testOutput, `"noul":0.97`, `"noul":1.97`, 1), nil},
		{"invalid_score", 200, strings.Replace(testOutput, `"score":0.8`, `"score":2.8`, 1), nil},
		{"invalid_distribution", 200, strings.Replace(testOutput, `"technical":0.02`, `"other":0.02`, 1), nil},
		{"unnormalized_distribution", 200, strings.Replace(testOutput, `"technical":0.02`, `"technical":0.8`, 1), nil},
		{"choice_not_argmax", 200, strings.Replace(testOutput, `"choice":"billing"`, `"choice":"technical"`, 1), nil},
		{"score_not_mean", 200, strings.Replace(testOutput, `"score":0.8`, `"score":0.2`, 1), nil},
		{"different_legend", 200, strings.Replace(testOutput, `"0":"Low"`, `"0":"Critical"`, 1), nil},
		{"missing_envelope", 200, `{"model":"jev","usage":{}}`, nil},
		{"null_envelope", 200, `{"model":"jev","answers":null,"usage":{}}`, nil},
		{"empty_envelope", 200, `{"model":"jev","answers":{},"usage":{}}`, nil},
		{"null_probability", 200, strings.Replace(testOutput, `"technical":0.02`, `"technical":null`, 1), nil},
		{"missing_confidence", 200, strings.Replace(testOutput, `"confidence":0.95,`, ``, 1), nil},
		{"null_noul", 200, strings.Replace(testOutput, `"noul":0.97`, `"noul":null`, 1), nil},
		{"invalid_confidence", 200, strings.Replace(testOutput, `"confidence":0.95`, `"confidence":-1`, 1), nil},
		{"missing_usage", 200, strings.Replace(testOutput, `"usage"`, `"missing_usage"`, 1), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			session := testSession(t, func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if tc.err != nil {
					return nil, tc.err
				}
				return response(tc.status, tc.body), nil
			})
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "decide", Arguments: json.RawMessage(testInput)})
			if err == nil && !result.IsError {
				t.Fatalf("fallo presentado como decisión: %+v", result)
			}
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded)+fmt.Sprint(err), "secret-for-test") {
				t.Fatal("el error filtró la credencial")
			}
			if calls.Load() != 1 {
				t.Fatalf("reintentó una llamada facturable: %d", calls.Load())
			}
		})
	}
}

func TestDistributionRoundingAndTies(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  string
		valid bool
	}{
		{"rounded_sum_and_mean", strings.Replace(testOutput, `"1":0.8`, `"1":0.79`, 1), true},
		{"sum_below_tolerance", strings.Replace(testOutput, `"1":0.8`, `"1":0.789`, 1), false},
		{"mean_outside_tolerance", strings.Replace(testOutput, `"score":0.8`, `"score":0.811`, 1), false},
		{"choice_tie", strings.Replace(testOutput, `"billing":0.98,"technical":0.02`, `"billing":0.5,"technical":0.5`, 1), true},
		{"probability_string", strings.Replace(testOutput, `"technical":0.02`, `"technical":"0.02"`, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := testSession(t, func(*http.Request) (*http.Response, error) {
				return response(200, tc.body), nil
			})
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "decide", Arguments: json.RawMessage(testInput)})
			valid := err == nil && !result.IsError
			if valid != tc.valid {
				t.Fatalf("valid=%v: %v, %+v", tc.valid, err, result)
			}
		})
	}
}
