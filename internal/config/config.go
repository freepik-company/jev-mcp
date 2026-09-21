// Package config resuelve el entorno una sola vez, antes de arrancar el servidor.
package config

import (
	"errors"
	"net/url"
	"strings"
)

const DefaultModel = "jev-latest"

type Config struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

func Load(getenv func(string) string) (Config, error) {
	cfg := Config{Model: DefaultModel, Provider: "openrouter"}
	var keyName string
	switch getenv("JEV_PROVIDER") {
	case "", "openrouter":
		cfg.BaseURL, keyName = "https://openrouter.ai/api", "OPENROUTER_API_KEY"
	case "typesafe":
		cfg.Provider = "typesafe"
		cfg.BaseURL, keyName = "https://api.typesafe.ai", "TYPESAFE_API_KEY"
	default:
		return Config{}, errors.New("JEV_PROVIDER debe ser openrouter o typesafe")
	}
	if baseURL := getenv("BASE_URL"); baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if model := getenv("JEV_MODEL"); model != "" {
		cfg.Model = model
	}
	cfg.APIKey = getenv("API_KEY")
	if cfg.APIKey == "" {
		cfg.APIKey = getenv(keyName)
	}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.HasPrefix(cfg.APIKey, "${") {
		return Config{}, errors.New("falta API_KEY; configúrala en el entorno del servidor MCP")
	}
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return Config{}, err
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return cfg, nil
}

func validateBaseURL(raw string) error {
	base, err := url.Parse(raw)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return errors.New("BASE_URL debe ser una URL base sin credenciales, query ni fragmento")
	}
	// HTTP solo sirve para pruebas o proxies locales; la clave nunca sale en claro.
	local := base.Hostname() == "127.0.0.1" || base.Hostname() == "::1" || base.Hostname() == "localhost"
	if base.Scheme != "https" && !(base.Scheme == "http" && local) {
		return errors.New("BASE_URL debe usar HTTPS (HTTP solo en loopback)")
	}
	return nil
}
