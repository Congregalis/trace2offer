package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"trace2offer/backend/agent/provider"
)

const DefaultBaseURL = "https://api.openai.com/v1/responses"

// Config configures OpenAI Responses provider.
type Config struct {
	Name       string
	APIKey     string
	BaseURL    string
	Model      string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// Provider implements provider.Provider using OpenAI Responses API.
type Provider struct {
	name       string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func New(config Config) (*Provider, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return nil, errors.New("openai api key is required")
	}
	model := strings.TrimSpace(config.Model)
	if model == "" {
		return nil, errors.New("openai model is required")
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	name := strings.TrimSpace(config.Name)
	if name == "" {
		name = "openai-responses"
	}

	client := config.HTTPClient
	if client == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	return &Provider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: client,
	}, nil
}

func (p *Provider) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

func (p *Provider) Generate(ctx context.Context, request provider.Request) (provider.Response, error) {
	if p == nil {
		return provider.Response{}, errors.New("openai provider is nil")
	}
	if len(request.Messages) == 0 {
		return provider.Response{}, errors.New("messages are required")
	}

	payload := responsesRequest{Model: p.model}
	if strings.TrimSpace(request.Model) != "" {
		payload.Model = strings.TrimSpace(request.Model)
	}

	payload.Input = make([]responsesInputMessage, 0, len(request.Messages))
	for _, msg := range request.Messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			continue
		}
		contentType := inputContentTypeForRole(role)

		payload.Input = append(payload.Input, responsesInputMessage{
			Role: role,
			Content: []responsesInputContent{{
				Type: contentType,
				Text: msg.Content,
			}},
		})
	}

	if len(payload.Input) == 0 {
		return provider.Response{}, errors.New("at least one valid message is required")
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return provider.Response{}, fmt.Errorf("encode openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(encoded))
	if err != nil {
		return provider.Response{}, fmt.Errorf("build openai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return provider.Response{}, fmt.Errorf("call openai responses api: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return provider.Response{}, fmt.Errorf("read openai response body: %w", err)
	}

	if httpResp.StatusCode >= http.StatusBadRequest {
		var apiErr responsesErrorEnvelope
		if json.Unmarshal(body, &apiErr) == nil && strings.TrimSpace(apiErr.Error.Message) != "" {
			return provider.Response{}, fmt.Errorf("openai responses api error (%d): %s", httpResp.StatusCode, apiErr.Error.Message)
		}
		return provider.Response{}, fmt.Errorf("openai responses api error (%d): %s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed responsesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return provider.Response{}, fmt.Errorf("decode openai response: %w", err)
	}

	text := strings.TrimSpace(parsed.OutputText)
	if text == "" {
		for _, output := range parsed.Output {
			for _, content := range output.Content {
				candidate := strings.TrimSpace(content.Text)
				if candidate != "" {
					text = candidate
					break
				}
			}
			if text != "" {
				break
			}
		}
	}
	if text == "" {
		return provider.Response{}, errors.New("openai response contains no text output")
	}

	return provider.Response{Content: text}, nil
}

func inputContentTypeForRole(role string) string {
	switch strings.TrimSpace(role) {
	case "assistant":
		return "output_text"
	default:
		return "input_text"
	}
}

type responsesRequest struct {
	Model string                  `json:"model"`
	Input []responsesInputMessage `json:"input"`
}

type responsesInputMessage struct {
	Role    string                  `json:"role"`
	Content []responsesInputContent `json:"content"`
}

type responsesInputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type responsesResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

type responsesErrorEnvelope struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
