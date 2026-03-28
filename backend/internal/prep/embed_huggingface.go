package prep

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultHFBaseURL = "https://api-inference.huggingface.co/models"

type HuggingFaceEmbeddingProviderConfig struct {
	APIKey     string
	Model      string
	BaseURL    string
	Dimension  int
	HTTPClient *http.Client
}

type HuggingFaceEmbeddingProvider struct {
	apiKey          string
	model           string
	baseURL         string
	explicitBaseURL bool
	dimension       int
	httpClient      *http.Client
}

func NewHuggingFaceEmbeddingProvider(config HuggingFaceEmbeddingProviderConfig) (*HuggingFaceEmbeddingProvider, error) {
	model := strings.TrimSpace(config.Model)
	if model == "" {
		model = defaultHFModel
	}
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = defaultHFBaseURL
	}
	dimension := config.Dimension
	if dimension <= 0 {
		dimension = 1024
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HuggingFaceEmbeddingProvider{
		apiKey:          strings.TrimSpace(config.APIKey),
		model:           model,
		baseURL:         baseURL,
		explicitBaseURL: strings.TrimSpace(config.BaseURL) != "",
		dimension:       dimension,
		httpClient:      client,
	}, nil
}

func (p *HuggingFaceEmbeddingProvider) Name() string {
	return "huggingface"
}

func (p *HuggingFaceEmbeddingProvider) Model() string {
	if p == nil {
		return defaultHFModel
	}
	return p.model
}

func (p *HuggingFaceEmbeddingProvider) Dimension() int {
	if p == nil {
		return 1024
	}
	return p.dimension
}

func (p *HuggingFaceEmbeddingProvider) Embed(texts []string) ([][]float32, error) {
	if p == nil {
		return nil, fmt.Errorf("huggingface embedding provider is nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	body, err := json.Marshal(map[string]any{"inputs": texts})
	if err != nil {
		return nil, fmt.Errorf("encode huggingface embedding request: %w", err)
	}

	endpoint, err := p.endpointURL()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create huggingface embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request huggingface embeddings: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read huggingface embedding response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("huggingface embeddings request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	embeddings, err := parseHuggingFaceEmbeddings(raw)
	if err != nil {
		return nil, err
	}
	if len(embeddings) != len(texts) {
		return nil, fmt.Errorf("huggingface embeddings length mismatch: input=%d output=%d", len(texts), len(embeddings))
	}
	if len(embeddings) > 0 {
		expected := len(embeddings[0])
		for i, vector := range embeddings {
			if len(vector) != expected {
				return nil, fmt.Errorf("huggingface embedding dimension mismatch at index %d: expected=%d actual=%d", i, expected, len(vector))
			}
		}
	}
	return embeddings, nil
}

func (p *HuggingFaceEmbeddingProvider) Validate() error {
	if p == nil {
		return fmt.Errorf("huggingface embedding provider is nil")
	}
	if !p.explicitBaseURL && strings.TrimSpace(p.apiKey) == "" {
		return &ValidationError{
			Field:   "embedding_provider",
			Message: fmt.Sprintf("%s is required when %s is not set", envPrepHFAPIKey, envPrepHFBaseURL),
		}
	}
	return nil
}

func (p *HuggingFaceEmbeddingProvider) endpointURL() (string, error) {
	if p == nil {
		return "", fmt.Errorf("huggingface embedding provider is nil")
	}
	baseURL := strings.TrimSpace(p.baseURL)
	if baseURL == "" {
		baseURL = defaultHFBaseURL
	}
	if strings.Contains(baseURL, "{model}") {
		return strings.ReplaceAll(baseURL, "{model}", url.PathEscape(p.model)), nil
	}
	trimmed := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(trimmed, "/models") {
		return trimmed + "/" + url.PathEscape(p.model), nil
	}
	return trimmed, nil
}

func parseHuggingFaceEmbeddings(raw []byte) ([][]float32, error) {
	var array2D [][]float64
	if err := json.Unmarshal(raw, &array2D); err == nil && len(array2D) > 0 {
		return convertToFloat32(array2D), nil
	}

	var array1D []float64
	if err := json.Unmarshal(raw, &array1D); err == nil && len(array1D) > 0 {
		return [][]float32{convertVectorToFloat32(array1D)}, nil
	}

	var wrapped struct {
		Embeddings [][]float64 `json:"embeddings"`
		Data       []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("decode huggingface embedding response: %w", err)
	}
	if len(wrapped.Embeddings) > 0 {
		return convertToFloat32(wrapped.Embeddings), nil
	}
	if len(wrapped.Data) > 0 {
		vectors := make([][]float32, 0, len(wrapped.Data))
		for _, item := range wrapped.Data {
			vectors = append(vectors, convertVectorToFloat32(item.Embedding))
		}
		return vectors, nil
	}
	return nil, fmt.Errorf("huggingface embedding response has no vectors")
}

func convertToFloat32(source [][]float64) [][]float32 {
	vectors := make([][]float32, 0, len(source))
	for _, vector := range source {
		vectors = append(vectors, convertVectorToFloat32(vector))
	}
	return vectors
}

func convertVectorToFloat32(source []float64) []float32 {
	vector := make([]float32, 0, len(source))
	for _, item := range source {
		vector = append(vector, float32(item))
	}
	return vector
}
