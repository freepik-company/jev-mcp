package systemone

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/freepik-company/jev-mcp/internal/config"
)

func TestModelCatalogues(t *testing.T) {
	for _, tc := range []struct {
		name, provider, body string
		valid                bool
		count                int
	}{
		{"typesafe", "typesafe", `{"models":[{"name":"jev-1.13","description":"Typed decisions","release_date":"2026-09-17"}]}`, true, 1},
		{"openrouter", "openrouter", `{"data":[{"id":"typesafe/jev","name":"Jev","architecture":{"output_modalities":["decisions"]},"pricing":{"prompt":"0.000000042"}},{"id":"chat","architecture":{"output_modalities":["text"]}}]}`, true, 1},
		{"empty", "typesafe", `{"models":[]}`, true, 0},
		{"missing", "typesafe", `{}`, false, 0},
		{"null", "openrouter", `{"data":null}`, false, 0},
		{"bad_json", "openrouter", `<html>`, false, 0},
		{"missing_id", "typesafe", `{"models":[{}]}`, false, 0},
		{"duplicate", "typesafe", `{"models":[{"name":"jev"},{"name":"jev"}]}`, false, 0},
		{"incomplete", "openrouter", `{"data":[],"links":{"next":"https://other/page"}}`, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := NewClient(config.Config{Provider: tc.provider, BaseURL: "https://provider.test/prefix", APIKey: "private", Model: "configured"}, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Path != "/prefix/v1/models" || req.Header.Get("Authorization") != "Bearer private" {
					t.Fatal("transporte incorrecto")
				}
				if (req.URL.Query().Get("output_modalities") == "decisions") != (tc.provider == "openrouter") {
					t.Fatal("filtro de modalidad incorrecto")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			}))
			result, err := client.ListModels(context.Background())
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
			if tc.valid && (len(result.Models) != tc.count || result.DefaultModel != "configured") {
				t.Fatalf("catálogo: %+v", result)
			}
			if tc.name == "typesafe" && (result.Models[0].ID != "jev-1.13" || result.Models[0].ReleaseDate != "2026-09-17") {
				t.Fatal("adaptación TypeSafe incorrecta")
			}
			if tc.name == "openrouter" && result.Models[0].Pricing["prompt"] != "0.000000042" {
				t.Fatal("precio alterado")
			}
			if calls != 1 {
				t.Fatal("llamadas extra")
			}
		})
	}
}
