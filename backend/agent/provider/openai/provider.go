package openai

import (
	"bufio"
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

	payload, err := p.buildResponsesRequest(request, false)
	if err != nil {
		return provider.Response{}, err
	}

	body, err := p.executeJSONRequest(ctx, payload)
	if err != nil {
		return provider.Response{}, err
	}

	var parsed responsesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return provider.Response{}, fmt.Errorf("decode openai response: %w", err)
	}

	text := extractResponseText(parsed)
	if text == "" {
		return provider.Response{}, errors.New("openai response contains no text output")
	}

	return provider.Response{Content: text}, nil
}

func (p *Provider) GenerateStream(
	ctx context.Context,
	request provider.Request,
	onDelta func(string),
) (provider.Response, error) {
	if p == nil {
		return provider.Response{}, errors.New("openai provider is nil")
	}

	payload, err := p.buildResponsesRequest(request, true)
	if err != nil {
		return provider.Response{}, err
	}

	httpResp, err := p.executeStreamRequest(ctx, payload)
	if err != nil {
		return provider.Response{}, err
	}
	defer httpResp.Body.Close()

	reader := bufio.NewReader(httpResp.Body)
	dataLines := make([]string, 0, 4)
	var outputBuilder strings.Builder
	var completed responsesResponse
	hasCompleted := false

	emitData := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]

		trimmed := strings.TrimSpace(payload)
		if trimmed == "" {
			return nil
		}
		if trimmed == "[DONE]" {
			return nil
		}

		var event responsesStreamEvent
		if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
			return fmt.Errorf("decode openai stream event: %w", err)
		}
		if strings.TrimSpace(event.Error.Message) != "" {
			return fmt.Errorf("openai stream error: %s", strings.TrimSpace(event.Error.Message))
		}

		switch strings.TrimSpace(event.Type) {
		case "response.output_text.delta":
			delta := event.Delta
			if delta != "" {
				outputBuilder.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}
		case "response.completed":
			if event.Response != nil {
				completed = *event.Response
				hasCompleted = true
			}
		}
		return nil
	}

	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if err := emitData(); err != nil {
				return provider.Response{}, err
			}
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return provider.Response{}, fmt.Errorf("read openai stream: %w", readErr)
		}
	}

	if err := emitData(); err != nil {
		return provider.Response{}, err
	}

	content := strings.TrimSpace(outputBuilder.String())
	if content == "" && hasCompleted {
		content = extractResponseText(completed)
	}
	if content == "" {
		return provider.Response{}, errors.New("openai stream contains no text output")
	}

	return provider.Response{Content: content}, nil
}

func (p *Provider) buildResponsesRequest(request provider.Request, stream bool) (responsesRequest, error) {
	if len(request.Messages) == 0 {
		return responsesRequest{}, errors.New("messages are required")
	}

	payload := responsesRequest{Model: p.model, Stream: stream}
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
		return responsesRequest{}, errors.New("at least one valid message is required")
	}

	return payload, nil
}

func (p *Provider) executeJSONRequest(ctx context.Context, payload responsesRequest) ([]byte, error) {
	httpResp, err := p.doRequest(ctx, payload, "application/json")
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response body: %w", err)
	}
	if httpResp.StatusCode >= http.StatusBadRequest {
		return nil, formatOpenAIStatusError(httpResp.StatusCode, body)
	}

	return body, nil
}

func (p *Provider) executeStreamRequest(ctx context.Context, payload responsesRequest) (*http.Response, error) {
	httpResp, err := p.doRequest(ctx, payload, "text/event-stream")
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode >= http.StatusBadRequest {
		defer httpResp.Body.Close()
		body, readErr := io.ReadAll(httpResp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read openai stream error body: %w", readErr)
		}
		return nil, formatOpenAIStatusError(httpResp.StatusCode, body)
	}

	return httpResp, nil
}

func (p *Provider) doRequest(ctx context.Context, payload responsesRequest, accept string) (*http.Response, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build openai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(accept) != "" {
		httpReq.Header.Set("Accept", strings.TrimSpace(accept))
	}

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call openai responses api: %w", err)
	}
	return httpResp, nil
}

func formatOpenAIStatusError(statusCode int, body []byte) error {
	var apiErr responsesErrorEnvelope
	if json.Unmarshal(body, &apiErr) == nil && strings.TrimSpace(apiErr.Error.Message) != "" {
		return fmt.Errorf("openai responses api error (%d): %s", statusCode, apiErr.Error.Message)
	}
	return fmt.Errorf("openai responses api error (%d): %s", statusCode, strings.TrimSpace(string(body)))
}

func extractResponseText(parsed responsesResponse) string {
	text := strings.TrimSpace(parsed.OutputText)
	if text != "" {
		return text
	}
	for _, output := range parsed.Output {
		for _, content := range output.Content {
			candidate := strings.TrimSpace(content.Text)
			if candidate != "" {
				return candidate
			}
		}
	}
	return ""
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
	Model  string                  `json:"model"`
	Input  []responsesInputMessage `json:"input"`
	Stream bool                    `json:"stream,omitempty"`
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

type responsesStreamEvent struct {
	Type     string             `json:"type"`
	Delta    string             `json:"delta"`
	Response *responsesResponse `json:"response"`
	Error    struct {
		Message string `json:"message"`
	} `json:"error"`
}
