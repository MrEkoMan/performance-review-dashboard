package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// aiClient invokes one configured AI provider with a prepared prompt and
// returns the model's markdown response. One implementation per provider
// family: the OpenAI-compatible chat completions shape covers openai,
// azure_openai, and openrouter; anthropic, gemini, and ollama each have their
// own request/response shape.
type aiClient interface {
	analyze(ctx context.Context, prompt string) (string, error)
}

// aiHTTPClient is the shared outbound HTTP client for provider calls. It is a
// package variable so tests can substitute a stub transport (see
// integration_connections_test.go for the same seam around
// integrationHTTPClient).
var aiHTTPClient = &http.Client{Timeout: 120 * time.Second}

type aiProviderConfig struct {
	BaseURL    string
	Model      string
	APIVersion string
	APIKey     string
}

func newAIClient(provider string, config aiProviderConfig) (aiClient, error) {
	switch provider {
	case "openai", "openrouter":
		return openAICompatibleClient{config: config}, nil
	case "azure_openai":
		return azureOpenAIClient{openAICompatibleClient{config: config}}, nil
	case "anthropic":
		return anthropicClient{config: config}, nil
	case "gemini":
		return geminiClient{config: config}, nil
	case "ollama":
		return ollamaClient{config: config}, nil
	default:
		return nil, errors.New("Unsupported AI provider")
	}
}

const aiAnalysisPreamble = "You are assisting a people manager preparing a holistic performance review. Analyze only the evidence provided and respond in GitHub-flavored markdown. Where the evidence is thin, say so plainly rather than inventing conclusions."

// aiCompletion sends the prepared request, validates the status, decodes the
// JSON body, and hands the payload to the provider-specific extractor. Error
// messages carry only the status code — response bodies are never echoed, so
// credentials and internal details cannot leak into the UI.
func aiCompletion(request *http.Request, extract func([]byte) (string, error)) (string, error) {
	response, err := aiHTTPClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", errors.New("The AI provider returned an invalid response")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("The AI provider returned HTTP %d", response.StatusCode)
	}
	content, err := extract(body)
	if err != nil {
		return "", errors.New("The AI provider returned an invalid response")
	}
	return content, nil
}

// postJSON builds and dispatches a JSON POST, shared by every adapter.
func postJSON(ctx context.Context, endpoint, payload string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, strings.NewReader(payload))
	if err != nil {
		return nil, errors.New("Provider URL is invalid")
	}
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

// openAICompatibleClient covers the chat/completions shape used by openai,
// openrouter, and azure_openai (when no API version is configured).
type openAICompatibleClient struct {
	config aiProviderConfig
}

func (client openAICompatibleClient) analyze(ctx context.Context, prompt string) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"model": client.config.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiAnalysisPreamble},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  4096,
		"temperature": 0.2,
	})
	if err != nil {
		return "", errors.New("Failed to prepare the AI request")
	}
	request, err := postJSON(
		ctx, strings.TrimRight(client.config.BaseURL, "/")+"/chat/completions",
		string(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+client.config.APIKey)
	return aiCompletion(request, extractOpenAIContent)
}

// azureOpenAIClient adapts the older Azure deployment style: the API version
// rides on the query string and the credential goes in the api-key header. When
// no API version is stored, the newer /openai/v1 endpoint style applies and the
// base openAICompatibleClient behavior (Bearer auth) is used instead.
type azureOpenAIClient struct {
	openAICompatibleClient
}

func (client azureOpenAIClient) analyze(ctx context.Context, prompt string) (string, error) {
	if client.config.APIVersion == "" {
		return client.openAICompatibleClient.analyze(ctx, prompt)
	}
	inner := client.openAICompatibleClient.config
	endpoint := strings.TrimRight(inner.BaseURL, "/") + "/chat/completions?api-version=" + inner.APIVersion
	payload, err := json.Marshal(map[string]any{
		"model": inner.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiAnalysisPreamble},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  4096,
		"temperature": 0.2,
	})
	if err != nil {
		return "", errors.New("Failed to prepare the AI request")
	}
	request, err := postJSON(ctx, endpoint, string(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("api-key", inner.APIKey)
	return aiCompletion(request, extractOpenAIContent)
}

// anthropicClient calls the Messages API. The preamble rides in the system
// field rather than a system message, and text blocks are concatenated because
// a response may contain several.
type anthropicClient struct {
	config aiProviderConfig
}

func (client anthropicClient) analyze(ctx context.Context, prompt string) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"model":      client.config.Model,
		"max_tokens": 4096,
		"system":     aiAnalysisPreamble,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})
	if err != nil {
		return "", errors.New("Failed to prepare the AI request")
	}
	request, err := postJSON(
		ctx, strings.TrimRight(client.config.BaseURL, "/")+"/v1/messages",
		string(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("x-api-key", client.config.APIKey)
	request.Header.Set("anthropic-version", "2023-06-01")
	return aiCompletion(request, extractAnthropicContent)
}

// geminiClient calls the generateContent endpoint; the API key rides in the
// query string, which is Gemini's documented mechanism for API-key auth.
type geminiClient struct {
	config aiProviderConfig
}

func (client geminiClient) analyze(ctx context.Context, prompt string) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": aiAnalysisPreamble + "\n\n" + prompt}}},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 4096,
			"temperature":     0.2,
		},
	})
	if err != nil {
		return "", errors.New("Failed to prepare the AI request")
	}
	endpoint := strings.TrimRight(client.config.BaseURL, "/") + "/models/" +
		client.config.Model + ":generateContent?key=" + client.config.APIKey
	request, err := postJSON(ctx, endpoint, string(payload))
	if err != nil {
		return "", err
	}
	return aiCompletion(request, extractGeminiContent)
}

// ollamaClient calls the local chat endpoint with streaming disabled. No auth
// header is set, and no encryption key is required to run it.
type ollamaClient struct {
	config aiProviderConfig
}

func (client ollamaClient) analyze(ctx context.Context, prompt string) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"model": client.config.Model,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": aiAnalysisPreamble},
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", errors.New("Failed to prepare the AI request")
	}
	request, err := postJSON(
		ctx, strings.TrimRight(client.config.BaseURL, "/")+"/api/chat",
		string(payload))
	if err != nil {
		return "", err
	}
	return aiCompletion(request, extractOllamaContent)
}

// Extractors turn each provider's response payload into plain text. They are
// exported so unit tests can exercise them directly against canned payloads.

func extractOpenAIContent(body []byte) (string, error) {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if len(payload.Choices) == 0 {
		return "", errors.New("no choices in response")
	}
	return payload.Choices[0].Message.Content, nil
}

func extractAnthropicContent(body []byte) (string, error) {
	var payload struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	text := ""
	for _, block := range payload.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	if text == "" {
		return "", errors.New("no text blocks in response")
	}
	return text, nil
}

func extractGeminiContent(body []byte) (string, error) {
	var payload struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if len(payload.Candidates) == 0 {
		return "", errors.New("no candidates in response")
	}
	text := ""
	for _, part := range payload.Candidates[0].Content.Parts {
		text += part.Text
	}
	if text == "" {
		return "", errors.New("no text parts in response")
	}
	return text, nil
}

func extractOllamaContent(body []byte) (string, error) {
	var payload struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.Message.Content == "" {
		return "", errors.New("no message content in response")
	}
	return payload.Message.Content, nil
}
