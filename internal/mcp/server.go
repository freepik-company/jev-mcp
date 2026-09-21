package mcpserver

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/freepik-company/jev-mcp/internal/systemone"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The embedded schemas define the public MCP contract.
//
//go:embed input.schema.json
var inputSchemaJSON []byte

//go:embed output.schema.json
var outputSchemaJSON []byte

// Version is reported to MCP clients during initialization. Release builds
// override it through -ldflags "-X .../internal/mcp.Version=<semver>".
var Version = "dev"

func New(client *systemone.Client) *mcp.Server {
	var inputSchema, outputSchema map[string]any
	// These are embedded constants: an error here is a programming mistake.
	if err := json.Unmarshal(inputSchemaJSON, &inputSchema); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(outputSchemaJSON, &outputSchema); err != nil {
		panic(err)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "jev-mcp", Version: Version}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name: "decide",
		Description: "Evaluate named, typed questions about a shared state using Jev/System One. " +
			"Use choice for one of your options, noul for the probability of yes, and score for ordered levels. " +
			"Returns answers, confidence/probabilities where provided, and usage including cost. " +
			"This is a paid inference call, not a governance approval. Never include credentials in the input.",
		InputSchema: inputSchema, OutputSchema: outputSchema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(true)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in systemone.Request) (*mcp.CallToolResult, json.RawMessage, error) {
		out, err := client.Decide(ctx, in)
		return nil, out, err
	})
	registerTasks(server, client, outputSchema)
	return server
}

func ptr[T any](value T) *T { return &value }
