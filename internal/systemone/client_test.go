package systemone

import (
	"context"
	"github.com/freepik-company/jev-mcp/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func TestInputLimitAndCancellation(t *testing.T) {
	client := NewClient(config.Config{BaseURL: "https://openrouter.ai/api", APIKey: "key", Model: "jev-latest"}, nil)
	var calls atomic.Int32
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	if _, err := client.Decide(context.Background(), Request{State: strings.Repeat("x", maxRequestBytes)}); err == nil {
		t.Fatal("accepted input larger than 1 MiB")
	}
	if calls.Load() != 0 {
		t.Fatal("oversized input reached the network")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Decide(ctx, Request{}); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("cancellation was not propagated: %v", err)
	}
}

func TestRedirectDoesNotForwardCredential(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: opens a local HTTP server")
	}
	var forwarded atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/leak" {
			forwarded.Add(1)
		}
		http.Redirect(w, r, "/leak", http.StatusTemporaryRedirect)
	}))
	defer target.Close()
	client := NewClient(config.Config{BaseURL: target.URL, APIKey: "private-key", Model: "jev-latest"}, nil)
	if _, err := client.Decide(context.Background(), Request{}); err == nil || !strings.Contains(err.Error(), "307") {
		t.Fatalf("redirect was not rejected: %v", err)
	}
	if forwarded.Load() != 0 {
		t.Fatal("the redirect target received the credential")
	}
}
