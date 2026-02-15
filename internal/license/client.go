package license

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultAPIURL  = "https://api.lemonsqueezy.com/v1/licenses"
	defaultTimeout = 10 * time.Second
)

// Client is a client for the LemonSqueezy License API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new LemonSqueezy license client.
func NewClient() *Client {
	return &Client{
		baseURL: defaultAPIURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Validate validates a license key with optional instance ID.
func (c *Client) Validate(ctx context.Context, licenseKey, instanceID string) (*ValidationResponse, error) {
	data := url.Values{}
	data.Set("license_key", licenseKey)
	if instanceID != "" {
		data.Set("instance_id", instanceID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/validate", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result ValidationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}

// Activate activates a license key for a specific instance.
func (c *Client) Activate(ctx context.Context, licenseKey, instanceName string) (*ActivationResponse, error) {
	data := url.Values{}
	data.Set("license_key", licenseKey)
	data.Set("instance_name", instanceName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/activate", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result ActivationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}

// Deactivate deactivates a license key for a specific instance.
func (c *Client) Deactivate(ctx context.Context, licenseKey, instanceID string) (*DeactivationResponse, error) {
	data := url.Values{}
	data.Set("license_key", licenseKey)
	data.Set("instance_id", instanceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/deactivate", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result DeactivationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}
