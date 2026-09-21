// Package systemone implementa el contrato compartido por OpenRouter y TypeSafe.
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

// Esta vista tipada valida las respuestas; el resultado completo se conserva
// como JSON para no perder costes ni extensiones del proveedor.
type response struct {
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
