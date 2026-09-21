package main

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Los esquemas son parte del contrato público, no tipos internos de Altherium.
//
//go:embed input.schema.json
var inputSchemaJSON []byte

//go:embed output.schema.json
var outputSchemaJSON []byte

type decideRequest struct {
	Model     string              `json:"model,omitempty"`
	State     any                 `json:"state"`
	Questions map[string]question `json:"questions"`
}

type question struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

func newServer(client *decisionClient) *mcp.Server {
	var inputSchema, outputSchema map[string]any
	// Son constantes embebidas: un error aquí es un fallo de programación.
	if err := json.Unmarshal(inputSchemaJSON, &inputSchema); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(outputSchemaJSON, &outputSchema); err != nil {
		panic(err)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "jev-mcp", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name: "decide",
		Description: "Evaluate named, typed questions about a shared state using Jev/System One. " +
			"Use choice for one of your options, noul for the probability of yes, and score for ordered levels. " +
			"Returns answers, confidence/probabilities where provided, and usage including cost. " +
			"This is a paid inference call, not a governance approval. Never include credentials in the input.",
		InputSchema: inputSchema, OutputSchema: outputSchema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(true)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in decideRequest) (*mcp.CallToolResult, map[string]any, error) {
		out, err := client.decide(ctx, in)
		return nil, out, err
	})
	return server
}

func ptr[T any](value T) *T { return &value }
