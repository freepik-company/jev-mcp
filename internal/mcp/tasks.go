package mcpserver

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/freepik-company/jev-mcp/internal/judgment"
	"github.com/freepik-company/jev-mcp/internal/systemone"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed tasks.schema.json
var taskSchemasJSON []byte

func registerTasks(server *mcp.Server, client *systemone.Client, decisionSchema map[string]any) {
	var schemas map[string]map[string]any
	if err := json.Unmarshal(taskSchemasJSON, &schemas); err != nil {
		panic(err)
	}
	for _, name := range []string{"classify", "verify", "rerank"} {
		schemas[name]["$defs"] = schemas["$defs"]
	}
	service := judgment.Service{Client: client}
	registerTask(server, "classify", "Classify 1-64 items into your categories in one paid call. IDs and input order are preserved. Supply an explicit fallback category if needed. Returns categories, confidence, probabilities, and the full provider response including usage. No automatic acceptance thresholds.", schemas["classify"], taskOutput(decisionSchema, "category", "string"), service.Classify)
	registerTask(server, "verify", "Assess 1-64 claims against supplied evidence only in one paid call. Returns supported, contradicted, or insufficient_evidence for each ID, with confidence and probabilities. Does not search for sources or establish objective truth; conflicting or missing evidence is insufficient. Preserves the full provider response and usage.", schemas["verify"], taskOutput(decisionSchema, "verdict", "string"), service.Verify)
	registerTask(server, "rerank", "Score 1-64 candidates independently for relevance to a query in one paid call. Returns descending scores on a 0-4 rubric; ties preserve input order. All candidates can score zero. Optional top_k limits returned results, not evaluation or cost. Includes the full provider response and usage.", schemas["rerank"], taskOutput(decisionSchema, "score", "number"), service.Rerank)
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_models", Description: "List decision models from the configured provider, with IDs usable in the model argument. Adapts OpenRouter and TypeSafe catalogues. Does not perform inference or test whether your account can invoke each model. The configured default_model may be an alias absent from the catalogue.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: ptr(true)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, systemone.ModelList, error) {
		out, err := client.ListModels(ctx)
		return nil, out, err
	})
}

func registerTask[I, O any](server *mcp.Server, name, description string, input, output map[string]any, run func(context.Context, I) (judgment.Result[O], error)) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description, InputSchema: input, OutputSchema: output, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(true)}}, func(ctx context.Context, _ *mcp.CallToolRequest, in I) (*mcp.CallToolResult, judgment.Result[O], error) {
		out, err := run(ctx, in)
		return nil, out, err
	})
}

func taskOutput(decision map[string]any, field, kind string) map[string]any {
	value := map[string]any{"type": kind}
	if field == "score" {
		value["minimum"], value["maximum"] = 0, 4
	}
	if field == "verdict" {
		value["enum"] = []string{"supported", "contradicted", "insufficient_evidence"}
	}
	return map[string]any{
		"type": "object", "required": []string{"results", "response"}, "additionalProperties": false,
		"$defs": decision["$defs"],
		"properties": map[string]any{
			"response": decision,
			"results": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"id", field, "confidence", "probabilities"},
				"properties": map[string]any{"id": map[string]any{"type": "string"}, field: value, "confidence": map[string]any{"$ref": "#/$defs/probability"}, "probabilities": map[string]any{"$ref": "#/$defs/distribution"}},
			}},
		},
	}
}
