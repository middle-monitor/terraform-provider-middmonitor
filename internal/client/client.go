// Package client is a minimal HTTP client for the Middle Monitor dashboard API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls /api/v1/organizations/{org_slug}/...
type Client struct {
	BaseURL    string
	Token      string
	OrgSlug    string
	HTTPClient *http.Client
}

func New(baseURL, token, orgSlug string) *Client {
	b := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{
		BaseURL: b,
		Token:   strings.TrimSpace(token),
		OrgSlug: strings.TrimSpace(orgSlug),
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) orgURL(path string) string {
	if path == "" {
		return fmt.Sprintf("%s/api/v1/organizations/%s", c.BaseURL, c.OrgSlug)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("%s/api/v1/organizations/%s%s", c.BaseURL, c.OrgSlug, path)
}

func (c *Client) doJSON(method, url string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return resp.StatusCode, &APIError{Method: method, URL: url, StatusCode: resp.StatusCode, Body: string(raw)}
	}

	if out != nil && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, &DecodeError{Cause: err, Body: truncate(string(raw), 500)}
		}
	}
	return resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// --- Models (subset of API JSON) ---

type Organization struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Plan      string `json:"plan"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Host struct {
	ID             int64   `json:"id,omitempty"`
	OrganizationID int64   `json:"organization_id,omitempty"`
	Name           string  `json:"name"`
	DisplayName    *string `json:"display_name,omitempty"`
	Host           string  `json:"host"`
	Service        string  `json:"service"`
	CreatedAt      string  `json:"created_at,omitempty"`
}

type Service struct {
	ID               int64    `json:"id,omitempty"`
	OrganizationID   int64    `json:"organization_id,omitempty"`
	HostID           *int64   `json:"host_id,omitempty"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Host             string   `json:"host"`
	Path             *string  `json:"path,omitempty"`
	Credentials      *string  `json:"credentials,omitempty"`
	Service          string   `json:"service"`
	ServiceInterval    int      `json:"service_interval,omitempty"`
	MaxAttempts        int      `json:"max_attempts,omitempty"`
	FailureThreshold   *float64 `json:"failure_threshold,omitempty"`
	ExpectedStatusCode *int     `json:"expected_status_code,omitempty"`
	CreatedAt          string   `json:"created_at,omitempty"`
}

type InstallToken struct {
	ID             int64   `json:"id"`
	OrganizationID int64   `json:"organization_id"`
	Token          string  `json:"token,omitempty"`
	TokenPrefix    string  `json:"token_prefix,omitempty"`
	Name           string  `json:"name"`
	CreatedAt      string  `json:"created_at"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
}

func (c *Client) GetOrganization() (*Organization, error) {
	var o Organization
	_, err := c.doJSON(http.MethodGet, c.orgURL(""), nil, &o)
	return &o, err
}

func (c *Client) CreateHost(h Host) (*Host, error) {
	var out Host
	_, err := c.doJSON(http.MethodPost, c.orgURL("/hosts"), h, &out)
	return &out, err
}

func (c *Client) GetHost(id int64) (*Host, error) {
	var out Host
	_, err := c.doJSON(http.MethodGet, c.orgURL(fmt.Sprintf("/hosts/%d", id)), nil, &out)
	return &out, err
}

func (c *Client) UpdateHost(id int64, h Host) (*Host, error) {
	var out Host
	_, err := c.doJSON(http.MethodPut, c.orgURL(fmt.Sprintf("/hosts/%d", id)), h, &out)
	return &out, err
}

func (c *Client) DeleteHost(id int64) error {
	req, err := http.NewRequest(http.MethodDelete, c.orgURL(fmt.Sprintf("/hosts/%d", id)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return &APIError{Method: http.MethodDelete, URL: c.orgURL(fmt.Sprintf("/hosts/%d", id)), StatusCode: resp.StatusCode, Body: string(raw)}
	}
	return nil
}

func (c *Client) CreateService(s Service) (*Service, error) {
	var out Service
	_, err := c.doJSON(http.MethodPost, c.orgURL("/services"), s, &out)
	return &out, err
}

func (c *Client) GetService(id int64) (*Service, error) {
	var out Service
	_, err := c.doJSON(http.MethodGet, c.orgURL(fmt.Sprintf("/services/%d", id)), nil, &out)
	return &out, err
}

func (c *Client) UpdateService(id int64, s Service) (*Service, error) {
	var out Service
	_, err := c.doJSON(http.MethodPut, c.orgURL(fmt.Sprintf("/services/%d", id)), s, &out)
	return &out, err
}

func (c *Client) DeleteService(id int64) error {
	req, err := http.NewRequest(http.MethodDelete, c.orgURL(fmt.Sprintf("/services/%d", id)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return &APIError{Method: http.MethodDelete, URL: c.orgURL(fmt.Sprintf("/services/%d", id)), StatusCode: resp.StatusCode, Body: string(raw)}
	}
	return nil
}

type createInstallTokenBody struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func (c *Client) CreateInstallToken(name string, expiresAt *string) (*InstallToken, error) {
	body := createInstallTokenBody{Name: name, ExpiresAt: expiresAt}
	var out InstallToken
	_, err := c.doJSON(http.MethodPost, c.orgURL("/install-tokens"), body, &out)
	return &out, err
}

func (c *Client) DeleteInstallToken(id int64) error {
	req, err := http.NewRequest(http.MethodDelete, c.orgURL(fmt.Sprintf("/install-tokens/%d", id)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return &APIError{Method: http.MethodDelete, URL: c.orgURL(fmt.Sprintf("/install-tokens/%d", id)), StatusCode: resp.StatusCode, Body: string(raw)}
	}
	return nil
}
