package prep

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHuggingFaceEmbeddingProviderDefaultEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer hf_test_key" {
			t.Fatalf("expected bearer token header")
		}
		if !strings.Contains(r.URL.Path, "/models/BAAI/bge-m3") && !strings.Contains(r.URL.EscapedPath(), "/models/BAAI%2Fbge-m3") {
			t.Fatalf("expected model path to contain model name, got path=%q escaped=%q", r.URL.Path, r.URL.EscapedPath())
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), "inputs") {
			t.Fatalf("expected inputs payload, got %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[[0.1, 0.2], [0.3, 0.4]]`))
	}))
	defer server.Close()

	provider, err := NewHuggingFaceEmbeddingProvider(HuggingFaceEmbeddingProviderConfig{
		APIKey:    "hf_test_key",
		Model:     "BAAI/bge-m3",
		BaseURL:   server.URL + "/models",
		Dimension: 2,
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	vectors, err := provider.Embed([]string{"first", "second"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != 2 || len(vectors[1]) != 2 {
		t.Fatalf("unexpected vector dimensions: %#v", vectors)
	}
}

func TestHuggingFaceEmbeddingProviderTEIResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embed" {
			t.Fatalf("expected /embed path, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[1.25,2.5]]}`))
	}))
	defer server.Close()

	provider, err := NewHuggingFaceEmbeddingProvider(HuggingFaceEmbeddingProviderConfig{
		Model:     "BAAI/bge-m3",
		BaseURL:   server.URL + "/embed",
		Dimension: 2,
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	vectors, err := provider.Embed([]string{"single"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vectors) != 1 {
		t.Fatalf("expected 1 vector, got %d", len(vectors))
	}
	if len(vectors[0]) != 2 {
		t.Fatalf("expected 2 dimensions, got %d", len(vectors[0]))
	}
}

func TestHuggingFaceEmbeddingProviderRequiresAPIKeyWithoutCustomBaseURL(t *testing.T) {
	t.Parallel()

	provider, err := NewHuggingFaceEmbeddingProvider(HuggingFaceEmbeddingProviderConfig{
		Model: "intfloat/multilingual-e5-small",
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Embed([]string{"test"})
	if err == nil {
		t.Fatal("expected missing api key validation error")
	}
	if !strings.Contains(err.Error(), envPrepHFAPIKey) {
		t.Fatalf("expected error mentioning %s, got %v", envPrepHFAPIKey, err)
	}
}
