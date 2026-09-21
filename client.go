package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	maxRequestBytes  = 1 << 20
	maxResponseBytes = 2 << 20
)

type decisionClient struct {
	endpoint string
	apiKey   string
	model    string
	http     *http.Client
}

func newClient(baseURL, apiKey, model string) (*decisionClient, error) {
	if strings.TrimSpace(apiKey) == "" || strings.HasPrefix(apiKey, "${") {
		return nil, fmt.Errorf("falta API_KEY; configúrala como secreto del servidor MCP")
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api"
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("BASE_URL debe ser una URL base sin credenciales, query ni fragmento")
	}
	// HTTP solo sirve para pruebas o proxies locales; la clave nunca sale en claro.
	local := base.Hostname() == "127.0.0.1" || base.Hostname() == "::1" || base.Hostname() == "localhost"
	if base.Scheme != "https" && !(base.Scheme == "http" && local) {
		return nil, fmt.Errorf("BASE_URL debe usar HTTPS (HTTP solo en loopback)")
	}
	if model == "" {
		model = "jev-latest"
	}
	return &decisionClient{
		endpoint: strings.TrimRight(base.String(), "/") + "/v1/systemone",
		apiKey:   apiKey, model: model,
		http: &http.Client{
			Timeout: 30 * time.Second,
			// Ni reintentos de inferencias de pago ni redirecciones con credenciales.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (c *decisionClient) decide(ctx context.Context, in decideRequest) (map[string]any, error) {
	if in.Model == "" {
		in.Model = c.model
	}
	body, err := json.Marshal(in)
	if err != nil || len(body) > maxRequestBytes {
		return nil, fmt.Errorf("Decision input must be valid JSON no larger than 1 MiB")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Cannot create the System One request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	res, err := c.http.Do(req)
	if err != nil {
		// Los errores de transporte pueden incluir la URL o cabeceras: no van al modelo.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("System One request cancelled")
		}
		return nil, fmt.Errorf("System One request failed or timed out")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Un proveedor puede reflejar la clave en su body de error. Solo sale el estado.
		return nil, fmt.Errorf("System One returned HTTP %d; check the MCP credential, quota and endpoint", res.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return nil, fmt.Errorf("System One response is unreadable or exceeds 2 MiB")
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("System One returned invalid JSON")
	}
	if err := validateAnswers(in.Questions, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Además del esquema, comprobamos la correspondencia: un JSON bien tipado puede
// omitir preguntas o elegir opciones que nadie ofreció. Eso nunca es una decisión.
func validateAnswers(questions map[string]question, out map[string]any) error {
	answers, ok := out["answers"].(map[string]any)
	if !ok || len(answers) != len(questions) {
		return fmt.Errorf("System One returned an incomplete answer set")
	}
	for id, q := range questions {
		answer, ok := answers[id].(map[string]any)
		if !ok || answer["type"] != q.Type {
			return fmt.Errorf("System One returned a missing or mismatched answer")
		}
		switch q.Type {
		case "noul":
			if !inRange(answer["noul"], 0, 1) {
				return fmt.Errorf("System One returned an invalid noul probability")
			}
		case "choice", "score":
			probabilities, ok := answer["probabilities"].(map[string]any)
			if !ok || !inRange(answer["confidence"], 0, 1) {
				return fmt.Errorf("System One returned invalid confidence or probabilities")
			}
			var keys []string
			if q.Type == "choice" {
				criteria, _ := q.Criteria.(map[string]any)
				choice, ok := answer["choice"].(string)
				if _, exists := criteria[choice]; !ok || !exists {
					return fmt.Errorf("System One chose an option that was not offered")
				}
				for key := range criteria {
					keys = append(keys, key)
				}
			} else {
				criteria, _ := q.Criteria.([]any)
				if !inRange(answer["score"], 0, float64(len(criteria)-1)) {
					return fmt.Errorf("System One returned a score outside the rubric")
				}
				for index := range criteria {
					keys = append(keys, strconv.Itoa(index))
				}
			}
			if len(keys) != len(probabilities) {
				return fmt.Errorf("System One returned an incomplete probability distribution")
			}
			var total, mean, maximum float64
			for _, key := range keys {
				if !inRange(probabilities[key], 0, 1) {
					return fmt.Errorf("System One returned an invalid probability distribution")
				}
				probability := probabilities[key].(float64)
				total += probability
				maximum = math.Max(maximum, probability)
				if q.Type == "score" {
					index, _ := strconv.Atoi(key)
					mean += float64(index) * probability
				}
			}
			// Tolerancia explícita para proveedores que redondeen a dos decimales.
			// No se normaliza ni se arregla la respuesta: se conserva tal como llegó.
			if math.Abs(total-1) > 0.01+1e-12 {
				return fmt.Errorf("System One returned probabilities that do not sum to one")
			}
			if q.Type == "choice" && probabilities[answer["choice"].(string)].(float64)+1e-12 < maximum {
				return fmt.Errorf("System One choice contradicts its probability distribution")
			}
			if q.Type == "score" {
				if math.Abs(answer["score"].(float64)-mean) > 0.01*float64(len(keys)-1)+1e-12 {
					return fmt.Errorf("System One score contradicts its probability distribution")
				}
				legend, ok := answer["legend"].(map[string]any)
				criteria := q.Criteria.([]any)
				if !ok || len(legend) != len(criteria) {
					return fmt.Errorf("System One returned an incomplete score legend")
				}
				for index, criterion := range criteria {
					value, exists := legend[strconv.Itoa(index)]
					if !exists || !reflect.DeepEqual(value, criterion) {
						return fmt.Errorf("System One score legend differs from the requested rubric")
					}
				}
			}
		}
	}
	return nil
}

func inRange(value any, low, high float64) bool {
	number, ok := value.(float64)
	return ok && number >= low && number <= high
}
