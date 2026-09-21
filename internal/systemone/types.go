// Package systemone implements the System One wire contract shared by OpenRouter and TypeSafe.
package systemone

import "encoding/json"

type Request struct {
	Model     string              `json:"model,omitempty"`
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
}

type Question struct {
	Type         string          `json:"type"`
	Instructions any             `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria,omitempty"`
}

// This typed view validates the answers; the complete result is kept as raw
// JSON so provider costs and extensions are never lost.
type response struct {
	Model *string `json:"model"`
	Usage *struct {
		InputTokens  *int64   `json:"input_tokens"`
		OutputTokens *int64   `json:"output_tokens"`
		Cost         *float64 `json:"cost"`
	} `json:"usage"`
	Answers map[string]answer `json:"answers"`
}

type answer struct {
	Type          string              `json:"type"`
	Choice        *string             `json:"choice"`
	Noul          *float64            `json:"noul"`
	Score         *float64            `json:"score"`
	Confidence    *float64            `json:"confidence"`
	Probabilities map[string]*float64 `json:"probabilities"`
	Legend        map[string]any      `json:"legend"`
}
