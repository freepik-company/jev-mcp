package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestConfiguration(t *testing.T) {
	for _, base := range []string{"file:///tmp/key", "http://example.com", "https://user:password@host", "https://host?key=secret", "https://host#fragment", "://"} {
		if _, err := newClient(base, "key", ""); err == nil {
			t.Errorf("aceptó BASE_URL inválida: %s", base)
		}
	}
	for _, key := range []string{"", " ", "${secrets:MISSING}"} {
		if _, err := newClient("", key, ""); err == nil {
			t.Error("aceptó una credencial ausente o sin resolver")
		}
	}
	for base, endpoint := range map[string]string{
		"":                                  "https://openrouter.ai/api/v1/systemone",
		"https://api.typesafe.ai":           "https://api.typesafe.ai/v1/systemone",
		"http://127.0.0.1:1234/prefix/api/": "http://127.0.0.1:1234/prefix/api/v1/systemone",
	} {
		client, err := newClient(base, "key", "")
		if err != nil || client.endpoint != endpoint || client.model != "jev-latest" {
			t.Fatalf("base %q: %+v, %v", base, client, err)
		}
	}
}

func TestProviderConfiguration(t *testing.T) {
	for _, tc := range []struct {
		provider string
		keyName  string
		endpoint string
	}{
		{"openrouter", "OPENROUTER_API_KEY", "https://openrouter.ai/api/v1/systemone"},
		{"typesafe", "TYPESAFE_API_KEY", "https://api.typesafe.ai/v1/systemone"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			env := map[string]string{"JEV_PROVIDER": tc.provider, tc.keyName: "provider-key", "JEV_MODEL": "jev-1.13"}
			getenv := func(name string) string { return env[name] }
			client, err := clientFromEnv(getenv)
			if err != nil || client.endpoint != tc.endpoint || client.apiKey != "provider-key" || client.model != "jev-1.13" {
				t.Fatalf("configuración de proveedor incorrecta: %+v, %v", client, err)
			}
			env["API_KEY"], env["BASE_URL"] = "explicit-key", "https://proxy.example/base"
			client, err = clientFromEnv(getenv)
			if err != nil || client.apiKey != "explicit-key" || client.endpoint != "https://proxy.example/base/v1/systemone" {
				t.Fatal("la configuración explícita debe prevalecer")
			}
			delete(env, "API_KEY")
			delete(env, tc.keyName)
			env["UNRELATED_API_KEY"] = "wrong-provider-key"
			if _, err := clientFromEnv(getenv); err == nil {
				t.Fatal("aceptó la clave de otro proveedor")
			}
		})
	}
	if _, err := clientFromEnv(func(string) string { return "unknown" }); err == nil {
		t.Fatal("aceptó un proveedor desconocido")
	}
}

func TestInputLimitAndCancellation(t *testing.T) {
	client, err := newClient("", "key", "")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	if _, err := client.decide(context.Background(), decideRequest{State: strings.Repeat("x", maxRequestBytes)}); err == nil {
		t.Fatal("aceptó entrada de más de 1 MiB")
	}
	if calls.Load() != 0 {
		t.Fatal("la entrada excesiva llegó a la red")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.decide(ctx, decideRequest{}); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("no propagó la cancelación: %v", err)
	}
}

func TestRedirectDoesNotForwardCredential(t *testing.T) {
	if testing.Short() {
		t.Skip("integración: abre HTTP local")
	}
	var forwarded atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/leak" {
			forwarded.Add(1)
		}
		http.Redirect(w, r, "/leak", http.StatusTemporaryRedirect)
	}))
	defer target.Close()
	client, err := newClient(target.URL, "private-key", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.decide(context.Background(), decideRequest{}); err == nil || !strings.Contains(err.Error(), "307") {
		t.Fatalf("no rechazó la redirección: %v", err)
	}
	if forwarded.Load() != 0 {
		t.Fatal("la redirección recibió la credencial")
	}
}

// El hijo usa el arranque real por stdio: la suite detecta también contaminación
// de stdout, env mal cableado o un proceso que no negocia el protocolo MCP.
func TestStdioRoundTrip(t *testing.T) {
	if os.Getenv("JEV_TEST_CHILD") == "1" {
		if err := run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if testing.Short() {
		t.Skip("integración: proceso MCP y HTTP local")
	}
	requests := make(chan *http.Request, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testOutput))
	}))
	defer target.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStdioRoundTrip$")
	command.Env = append(os.Environ(), "JEV_TEST_CHILD=1", "BASE_URL="+target.URL+"/api", "API_KEY=stdio-key", "JEV_MODEL=jev-latest")
	session, err := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "decide", Arguments: json.RawMessage(testInput)})
	if err != nil || result.IsError {
		t.Fatalf("falló el MCP por stdio: %v, %+v", err, result)
	}
	req := <-requests
	if req.URL.Path != "/api/v1/systemone" || req.Header.Get("Authorization") != "Bearer stdio-key" {
		t.Fatal("el proceso no usó el endpoint o la credencial configurados")
	}
}
