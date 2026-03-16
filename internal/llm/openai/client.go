package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"ai-product-assistant/internal/model"
)

const chatCompletionsPath = "/chat/completions"
const defaultMaxResponseBytes int64 = 1 << 20

type Config struct {
	BaseURL        string
	APIKey         string
	Model          string
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
	MaxResponseBytes int64
}

type Client struct {
	baseURL        string
	apiKey         string
	model          string
	maxRetries     int
	retryBaseDelay time.Duration
	maxResponseBytes int64
	httpClient     *http.Client
	sleepFn        func(ctx context.Context, d time.Duration) error
}

func NewClient(cfg Config) *Client {
	maxResponseBytes := cfg.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultMaxResponseBytes
	}

	return &Client{
		baseURL:        strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:         cfg.APIKey,
		model:          cfg.Model,
		maxRetries:     cfg.MaxRetries,
		retryBaseDelay: cfg.RetryBaseDelay,
		maxResponseBytes: maxResponseBytes,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		sleepFn: sleepWithContext,
	}
}

func (c *Client) GenerateDraft(ctx context.Context, input model.LLMGenerateParams) (model.LLMGenerateResult, error) {
	reqPayload := chatCompletionRequest{
		Model:    c.model,
		Messages: buildGenerateMessages(input),
	}

	resp, err := c.createChatCompletion(ctx, reqPayload)
	if err != nil {
		return model.LLMGenerateResult{}, fmt.Errorf("generate draft: %w", err)
	}

	content, err := parseDraftContent(resp)
	if err != nil {
		return model.LLMGenerateResult{}, fmt.Errorf("generate draft: %w", err)
	}

	return model.LLMGenerateResult{
		Content: content,
		Usage: model.LLMUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (c *Client) RefineDraft(ctx context.Context, input model.LLMRefineParams) (model.LLMRefineResult, error) {
	reqPayload := chatCompletionRequest{
		Model:    c.model,
		Messages: buildRefineMessages(input),
	}

	resp, err := c.createChatCompletion(ctx, reqPayload)
	if err != nil {
		return model.LLMRefineResult{}, fmt.Errorf("refine draft: %w", err)
	}

	content, err := parseDraftContent(resp)
	if err != nil {
		return model.LLMRefineResult{}, fmt.Errorf("refine draft: %w", err)
	}

	return model.LLMRefineResult{
		Content: content,
		Usage: model.LLMUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (c *Client) createChatCompletion(ctx context.Context, payload chatCompletionRequest) (chatCompletionResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return chatCompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	respBody, err := c.doRequestWithRetry(ctx, body)
	if err != nil {
		return chatCompletionResponse{}, err
	}

	var resp chatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return chatCompletionResponse{}, fmt.Errorf("%w: parse provider response: %v", ErrInvalidResponse, err)
	}

	return resp, nil
}

func (c *Client) doRequestWithRetry(ctx context.Context, body []byte) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		respBody, retry, retryAfter, err := c.doRequest(ctx, body)
		if err == nil {
			return respBody, nil
		}

		lastErr = err
		if !retry || attempt == c.maxRetries {
			break
		}

		waitFor := c.retryDelay(attempt)
		if retryAfter > waitFor {
			waitFor = retryAfter
		}

		if err := c.sleepFn(ctx, waitFor); err != nil {
			return nil, err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("%w: request failed", ErrProvider)
	}

	return nil, lastErr
}

func (c *Client) doRequest(ctx context.Context, body []byte) ([]byte, bool, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+chatCompletionsPath, bytes.NewReader(body))
	if err != nil {
		return nil, false, 0, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return nil, true, 0, fmt.Errorf("%w: %w", ErrTimeout, err)
		}
		if errors.Is(err, context.Canceled) {
			return nil, false, 0, err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, true, 0, fmt.Errorf("%w: %w", ErrTimeout, err)
		}
		return nil, true, 0, fmt.Errorf("%w: %w", ErrProvider, err)
	}
	defer resp.Body.Close()

	respBody, err := readResponseBody(resp.Body, c.maxResponseBytes)
	if err != nil {
		return nil, false, 0, err
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return respBody, false, 0, nil
	}

	providerErr := formatProviderHTTPError(resp.StatusCode, respBody)
	if shouldRetryStatus(resp.StatusCode) {
		return nil, true, retryAfterDelay(resp.Header.Get("Retry-After")), providerErr
	}

	return nil, false, 0, providerErr
}

func parseDraftContent(resp chatCompletionResponse) (model.DraftContent, error) {
	if len(resp.Choices) == 0 {
		return model.DraftContent{}, fmt.Errorf("%w: empty choices", ErrInvalidResponse)
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	if raw == "" {
		return model.DraftContent{}, fmt.Errorf("%w: empty message content", ErrInvalidResponse)
	}

	clean := normalizeJSON(raw)

	var content model.DraftContent
	if err := json.Unmarshal([]byte(clean), &content); err != nil {
		return model.DraftContent{}, fmt.Errorf("%w: parse structured content: %v", ErrInvalidResponse, err)
	}

	if strings.TrimSpace(content.Summary) == "" {
		return model.DraftContent{}, fmt.Errorf("%w: summary is empty", ErrInvalidResponse)
	}
	if strings.TrimSpace(content.Audience) == "" {
		return model.DraftContent{}, fmt.Errorf("%w: audience is empty", ErrInvalidResponse)
	}
	if strings.TrimSpace(content.Value) == "" {
		return model.DraftContent{}, fmt.Errorf("%w: value is empty", ErrInvalidResponse)
	}
	if content.Scenarios == nil || content.Constraints == nil || content.Risks == nil || content.Questions == nil {
		return model.DraftContent{}, fmt.Errorf("%w: required array field is nil", ErrInvalidResponse)
	}

	return content, nil
}

func normalizeJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}

func formatProviderHTTPError(statusCode int, body []byte) error {
	var providerResp providerErrorResponse
	if err := json.Unmarshal(body, &providerResp); err == nil && providerResp.Error.Message != "" {
		return fmt.Errorf("%w: status=%d type=%s code=%v message=%s", ErrProvider, statusCode, providerResp.Error.Type, providerResp.Error.Code, providerResp.Error.Message)
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Errorf("%w: status=%d", ErrProvider, statusCode)
	}

	return fmt.Errorf("%w: status=%d body=%s", ErrProvider, statusCode, trimmed)
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func readResponseBody(reader io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(reader, limit+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("%w: read provider response: %v", ErrProvider, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%w: provider response too large", ErrProvider)
	}

	return body, nil
}

func retryAfterDelay(value string) time.Duration {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}

	seconds, err := time.ParseDuration(trimmed + "s")
	if err == nil && seconds > 0 {
		return seconds
	}

	when, err := http.ParseTime(trimmed)
	if err != nil {
		return 0
	}

	delay := time.Until(when)
	if delay <= 0 {
		return 0
	}

	return delay
}

func (c *Client) retryDelay(attempt int) time.Duration {
	if c.retryBaseDelay <= 0 {
		return 0
	}

	delay := c.retryBaseDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
	}

	return delay
}
