// Package helpers provides narrowly-scoped utilities for E2E testing.
package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Response represents an HTTP response from the API.
type Response struct {
	// StatusCode is the HTTP status code (e.g., 200, 404, 500).
	StatusCode int

	// Body contains the raw response body bytes.
	Body []byte

	// Headers contains the response headers.
	Headers http.Header
}

// JSON unmarshals the response body into the provided value.
//
//	var result map[string]interface{}
//	if err := resp.JSON(&result); err != nil {
//	    return err
//	}
func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// String returns the response body as a string.
func (r *Response) String() string {
	return string(r.Body)
}

// APIClient provides HTTP request capabilities for E2E tests.
//
// Use this helper to:
//   - Test API endpoints directly
//   - Verify response status codes and bodies
//   - Test authentication and authorization
//
// APIClient intentionally exposes only a single Call method to keep
// the E2E testing interface minimal and focused.
type APIClient struct {
	baseURL string
	headers map[string]string
	client  *http.Client
}

// NewAPIClient creates a new API client with the given base URL.
//
// The base URL should include the scheme and host (e.g., "http://localhost:8080").
// Do not include a trailing slash.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		headers: make(map[string]string),
		client:  &http.Client{},
	}
}

// SetHeader sets a header that will be included in all subsequent requests.
//
// Use this to set authentication headers or API keys:
//
//	client.SetHeader("x-api-key", "test-api-key")
//	client.SetHeader("Cookie", "session_token=abc123")
func (c *APIClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// Call makes an HTTP request and returns the response.
//
// The method parameter should be an HTTP method: GET, POST, PUT, DELETE, PATCH.
// The path parameter is appended to the base URL (should start with /).
// The body parameter is JSON-encoded and sent as the request body (can be nil).
//
// Examples:
//
//	// GET request
//	resp, err := api.Call("GET", "/api/entities", nil)
//
//	// POST request with JSON body
//	resp, err := api.Call("POST", "/api/loads/upsert", map[string]interface{}{
//	    "name": "Test Load",
//	    "effort": 5,
//	})
//
//	// DELETE request
//	resp, err := api.Call("DELETE", "/api/my-capacity/override/2024-01-15", nil)
//
// Returns an error if the request fails to complete. HTTP error status codes
// (4xx, 5xx) are NOT treated as errors - check resp.StatusCode instead.
func (c *APIClient) Call(method, path string, body interface{}) (*Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}
