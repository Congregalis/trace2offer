package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"trace2offer/backend/agent/provider"
)

func TestGenerate_AssistantMessageUsesOutputText(t *testing.T) {
	t.Parallel()

	var received responsesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"ok"}`))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{
		APIKey:     "test_key",
		Model:      "test_model",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	resp, err := p.Generate(context.Background(), provider.Request{
		Messages: []provider.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "u1"},
			{Role: "assistant", Content: "a1"},
			{Role: "user", Content: "u2"},
		},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}

	if len(received.Input) != 4 {
		t.Fatalf("expected 4 input messages, got %d", len(received.Input))
	}
	if got := received.Input[0].Content[0].Type; got != "input_text" {
		t.Fatalf("system message content type = %q, want input_text", got)
	}
	if got := received.Input[1].Content[0].Type; got != "input_text" {
		t.Fatalf("user message content type = %q, want input_text", got)
	}
	if got := received.Input[2].Content[0].Type; got != "output_text" {
		t.Fatalf("assistant message content type = %q, want output_text", got)
	}
	if got := received.Input[3].Content[0].Type; got != "input_text" {
		t.Fatalf("user message content type = %q, want input_text", got)
	}
}

func TestGenerateStream_EmitsDeltaAndReturnsFullText(t *testing.T) {
	t.Parallel()

	var received responsesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"O\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"K\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"output_text\":\"OK\"}}\n\n"))
	}))
	t.Cleanup(server.Close)

	p, err := New(Config{
		APIKey:     "test_key",
		Model:      "test_model",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	var deltas strings.Builder
	resp, err := p.GenerateStream(context.Background(), provider.Request{
		Messages: []provider.Message{
			{Role: "user", Content: "hello"},
		},
	}, func(delta string) {
		deltas.WriteString(delta)
	})
	if err != nil {
		t.Fatalf("generate stream: %v", err)
	}

	if got := deltas.String(); got != "OK" {
		t.Fatalf("unexpected deltas: %q", got)
	}
	if resp.Content != "OK" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
	if !received.Stream {
		t.Fatal("expected stream=true in request payload")
	}
}
