package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
)

func TestCodexExecutor_OverloadRetry_HTTP502_Recovers(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"type":"service_unavailable_error","code":"server_is_overloaded","message":"Our servers are currently overloaded."}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.created\ndata: " + codexCreatedEvent + "\n\n"))
		_, _ = w.Write([]byte("event: response.output_text.delta\ndata: " + codexOutputDeltaEvent + "\n\n"))
		_, _ = w.Write([]byte("event: response.completed\ndata: " + codexCompletedEventBody + "\n\n"))
	}))
	defer server.Close()

	retries := 2
	cfg := &config.Config{
		Codex: config.CodexConfig{
			OverloadRetry:      &retries,
			OverloadRetryDelay: "10ms",
		},
	}
	exec := NewCodexExecutor(cfg)
	auth := codexTestAuth(server.URL)
	req, opts := codexTestRequest()

	result, err := exec.ExecuteStream(context.Background(), auth, req, opts)
	if err != nil {
		t.Fatalf("expected ExecuteStream to recover on retry, got err: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil StreamResult")
	}
	var chunks []cliproxyexecutor.StreamChunk
	for chunk := range result.Chunks {
		chunks = append(chunks, chunk)
	}
	if len(chunks) == 0 {
		t.Fatal("expected stream chunks after retry recovery")
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestCodexExecutor_OverloadRetry_BootstrapOverload_Recovers(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		if current == 1 {
			// First attempt sends handshake followed by sequence 2 overload event
			_, _ = w.Write([]byte("event: response.created\ndata: " + codexCreatedEvent + "\n\n"))
			_, _ = w.Write([]byte("event: error\ndata: " + codexOverloadEvent + "\n\n"))
			return
		}
		// Second attempt succeeds
		_, _ = w.Write([]byte("event: response.created\ndata: " + codexCreatedEvent + "\n\n"))
		_, _ = w.Write([]byte("event: response.output_text.delta\ndata: " + codexOutputDeltaEvent + "\n\n"))
		_, _ = w.Write([]byte("event: response.completed\ndata: " + codexCompletedEventBody + "\n\n"))
	}))
	defer server.Close()

	retries := 2
	cfg := &config.Config{
		Codex: config.CodexConfig{
			StreamBootstrapBuffering: true,
			OverloadRetry:            &retries,
			OverloadRetryDelay:       "10ms",
		},
	}
	exec := NewCodexExecutor(cfg)
	auth := codexTestAuth(server.URL)
	req, opts := codexTestRequest()

	result, err := exec.ExecuteStream(context.Background(), auth, req, opts)
	if err != nil {
		t.Fatalf("expected ExecuteStream to recover from bootstrap overload on retry, got err: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil StreamResult")
	}
	var chunks []cliproxyexecutor.StreamChunk
	for chunk := range result.Chunks {
		chunks = append(chunks, chunk)
	}
	if len(chunks) == 0 {
		t.Fatal("expected stream chunks after retry recovery")
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestCodexExecutor_OverloadRetry_Exhausted_ReturnsError(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"type":"service_unavailable_error","code":"server_is_overloaded","message":"Our servers are currently overloaded."}}`))
	}))
	defer server.Close()

	retries := 2
	cfg := &config.Config{
		Codex: config.CodexConfig{
			OverloadRetry:      &retries,
			OverloadRetryDelay: "10ms",
		},
	}
	exec := NewCodexExecutor(cfg)
	auth := codexTestAuth(server.URL)
	req, opts := codexTestRequest()

	result, err := exec.ExecuteStream(context.Background(), auth, req, opts)
	if err == nil {
		t.Fatal("expected error after retries exhausted, got nil")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	// Initial attempt + 2 retries = 3 attempts total
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}
