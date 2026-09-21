package systemone

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
)

type Model struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	ReleaseDate   string            `json:"release_date,omitempty"`
	ContextLength int               `json:"context_length,omitempty"`
	Pricing       map[string]string `json:"pricing,omitempty"`
}

type ModelList struct {
	Models       []Model `json:"models"`
	DefaultModel string  `json:"default_model"`
}

func (c *Client) ListModels(ctx context.Context) (ModelList, error) {
	endpoint := c.baseURL + "/v1/models"
	if c.provider != "typesafe" {
		endpoint += "?output_modalities=decisions"
	}
	body, err := c.request(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ModelList{}, err
	}
	models, err := decodeModels(body, c.provider)
	return ModelList{Models: models, DefaultModel: c.model}, err
}

func decodeModels(body []byte, provider string) ([]Model, error) {
	invalid := errors.New("Provider returned an invalid model catalogue")
	models := []Model{}
	if provider == "typesafe" {
		var wire struct {
			Models *[]struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				ReleaseDate string `json:"release_date"`
			} `json:"models"`
		}
		if json.Unmarshal(body, &wire) != nil || wire.Models == nil {
			return nil, invalid
		}
		for _, model := range *wire.Models {
			models = append(models, Model{ID: model.Name, Name: model.Name, Description: model.Description, ReleaseDate: model.ReleaseDate})
		}
	} else {
		var wire struct {
			Data *[]struct {
				Model
				Architecture struct {
					OutputModalities []string `json:"output_modalities"`
				} `json:"architecture"`
			} `json:"data"`
			Links struct {
				Next *string `json:"next"`
			} `json:"links"`
		}
		if json.Unmarshal(body, &wire) != nil || wire.Data == nil {
			return nil, invalid
		}
		// Without requested pagination OpenRouter returns the whole catalogue. Pages are never silently dropped.
		if wire.Links.Next != nil && *wire.Links.Next != "" {
			return nil, errors.New("Provider returned an incomplete model catalogue")
		}
		for _, model := range *wire.Data {
			if slices.Contains(model.Architecture.OutputModalities, "decisions") {
				models = append(models, model.Model)
			}
		}
	}
	seen := map[string]bool{}
	for _, model := range models {
		if strings.TrimSpace(model.ID) == "" || seen[model.ID] {
			return nil, invalid
		}
		seen[model.ID] = true
	}
	return models, nil
}
