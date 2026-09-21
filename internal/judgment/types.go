// Package judgment prepara tareas concretas sobre el cliente común de System One.
package judgment

import "encoding/json"

type Item struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type ClassifyInput struct {
	Items        []Item            `json:"items"`
	Categories   map[string]string `json:"categories"`
	Instructions string            `json:"instructions"`
	Model        string            `json:"model,omitempty"`
}

type VerifyInput struct {
	Claims   []Item `json:"claims"`
	Evidence []Item `json:"evidence"`
	Model    string `json:"model,omitempty"`
}

type RerankInput struct {
	Query      string `json:"query"`
	Candidates []Item `json:"candidates"`
	TopK       int    `json:"top_k,omitempty"`
	Model      string `json:"model,omitempty"`
}

type Classification struct {
	ID            string             `json:"id"`
	Category      string             `json:"category"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type Verification struct {
	ID            string             `json:"id"`
	Verdict       string             `json:"verdict"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type RankedItem struct {
	ID            string             `json:"id"`
	Score         float64            `json:"score"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type Result[T any] struct {
	Results []T `json:"results"`
	// La respuesta original conserva uso, coste, rúbricas y extensiones del proveedor.
	Response json.RawMessage `json:"response"`
}
