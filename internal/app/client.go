package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return fmt.Sprintf("%s %s failed with status %d: %s", e.Method, e.Path, e.StatusCode, e.Message)
	}
	if strings.TrimSpace(e.Body) != "" {
		return fmt.Sprintf("%s %s failed with status %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("%s %s failed with status %d", e.Method, e.Path, e.StatusCode)
}

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func IsAPIKeyCredential(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "qpk_")
}

func NewClient(baseURL, token string) *Client {
	normalizedBaseURL, err := NormalizeBaseURL(baseURL)
	if err != nil {
		normalizedBaseURL = DefaultBaseURL
	}

	return &Client{
		BaseURL: normalizedBaseURL,
		Token:   strings.TrimSpace(token),
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.Token = strings.TrimSpace(token)
}

func (c *Client) Get(ctx context.Context, path string, query url.Values, auth bool, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, auth, out)
}

func (c *Client) Post(ctx context.Context, path string, body any, auth bool, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, auth, out)
}

func (c *Client) Put(ctx context.Context, path string, body any, auth bool, out any) error {
	return c.do(ctx, http.MethodPut, path, nil, body, auth, out)
}

func (c *Client) Patch(ctx context.Context, path string, body any, auth bool, out any) error {
	return c.do(ctx, http.MethodPatch, path, nil, body, auth, out)
}

func (c *Client) Delete(ctx context.Context, path string, query url.Values, auth bool, out any) error {
	return c.do(ctx, http.MethodDelete, path, query, nil, auth, out)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, auth bool, out any) error {
	fullURL := c.BaseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		if c.Token == "" {
			return fmt.Errorf("authentication required; run 'quickpod auth login', store a credential with 'quickpod auth set-token', or set QUICKPOD_TOKEN/QUICKPOD_API_KEY")
		}
		if IsAPIKeyCredential(c.Token) {
			req.Header.Set("X-API-Key", c.Token)
			req.Header.Set("Authorization", "ApiKey "+c.Token)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Path:       path,
			Body:       strings.TrimSpace(string(responseBody)),
		}
		var errPayload map[string]any
		if json.Unmarshal(responseBody, &errPayload) == nil {
			if message := firstNonEmpty(
				StringValue(errPayload["error"]),
				StringValue(errPayload["details"]),
				StringValue(errPayload["message"]),
			); message != "" {
				apiErr.Message = message
			}
		}
		return apiErr
	}

	if out == nil || len(responseBody) == 0 {
		return nil
	}

	return json.Unmarshal(responseBody, out)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
