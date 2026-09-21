// Jev es un proceso MCP independiente: solo depende de stdlib y del SDK MCP.
// Mantenerlo fuera de internal permite extraer esta carpeta a otro repositorio.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "jev-mcp:", err)
		os.Exit(1)
	}
}

func run() error {
	client, err := clientFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return newServer(client).Run(ctx, &mcp.StdioTransport{})
}

func clientFromEnv(getenv func(string) string) (*decisionClient, error) {
	provider := getenv("JEV_PROVIDER")
	baseURL, keyName := "https://openrouter.ai/api", "OPENROUTER_API_KEY"
	switch provider {
	case "", "openrouter":
	case "typesafe":
		baseURL, keyName = "https://api.typesafe.ai", "TYPESAFE_API_KEY"
	default:
		return nil, fmt.Errorf("JEV_PROVIDER debe ser openrouter o typesafe")
	}
	if configured := getenv("BASE_URL"); configured != "" {
		baseURL = configured
	}
	key := getenv("API_KEY")
	if key == "" {
		key = getenv(keyName)
	}
	return newClient(baseURL, key, getenv("JEV_MODEL"))
}
