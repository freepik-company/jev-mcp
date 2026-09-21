package main

import (
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

const testInput = `{"state":"Refund requested", "questions":{"refund":{"type":"noul","instructions":"Is a refund requested?"}}}`
const testOutput = `{"model":"jev-latest", "answers":{"refund":{"type":"noul","noul":0.99}},"usage":{"input_tokens":10,"output_tokens":2}}`

// The child uses the real stdio start-up, so the suite also catches stdout
// pollution, miswired environment or a process that never negotiates MCP.
func TestStdioRoundTrip(t *testing.T) {
	if os.Getenv("JEV_TEST_CHILD") == "1" {
		if err := run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if testing.Short() {
		t.Skip("integration: spawns the MCP process and a local HTTP server")
	}
	requests := make(chan *http.Request, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testOutput))
	}))
	defer target.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStdioRoundTrip$")
	command.Env = append(os.Environ(), "JEV_TEST_CHILD=1", "JEV_PROVIDER=openrouter", "BASE_URL="+target.URL+"/api", "API_KEY=stdio-key", "JEV_MODEL=jev-latest")
	session, err := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "decide", Arguments: json.RawMessage(testInput)})
	if err != nil || result.IsError {
		t.Fatalf("stdio MCP failed: %v, %+v", err, result)
	}
	req := <-requests
	if req.URL.Path != "/api/v1/systemone" || req.Header.Get("Authorization") != "Bearer stdio-key" {
		t.Fatal("the process did not use the configured endpoint or credential")
	}
}
