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
	ID                 int64    `json:"id,omitempty"`
	OrganizationID     int64    `json:"organization_id,omitempty"`
	HostID             *int64   `json:"host_id,omitempty"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Host               string   `json:"host"`
	Path               *string  `json:"path,omitempty"`
	Credentials        *string  `json:"credentials,omitempty"`
	Service            string   `json:"service"`
	ServiceInterval    int      `json:"service_interval,omitempty"`
	MaxAttempts        int      `json:"max_attempts,omitempty"`
	FailureThreshold   *float64 `json:"failure_threshold,omitempty"`
	WarningThreshold   *float64 `json:"warning_threshold,omitempty"`
	CriticalThreshold  *float64 `json:"critical_threshold,omitempty"`
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

// HostGroup scopes alert correlation to a host and its peers.
type HostGroup struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name"`
	DisplayName *string `json:"display_name,omitempty"`
	IsDefault   bool    `json:"is_default,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

// MetricLabel is one label equality constraint on a custom metric series.
type MetricLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AlertRule is a threshold rule, including the custom-metric form that is the
// only way to alert on a scraped series.
type AlertRule struct {
	ID                int64         `json:"id,omitempty"`
	Name              string        `json:"name"`
	Description       *string       `json:"description,omitempty"`
	Type              string        `json:"type,omitempty"`
	TargetType        string        `json:"target_type,omitempty"`
	TargetID          *int64        `json:"target_id,omitempty"`
	Metric            string        `json:"metric,omitempty"`
	CustomMetric      *string       `json:"custom_metric,omitempty"`
	CustomLabels      []MetricLabel `json:"custom_labels,omitempty"`
	Aggregation       string        `json:"aggregation,omitempty"`
	Operator          string        `json:"operator,omitempty"`
	Threshold         float64       `json:"threshold"`
	WarningThreshold  *float64      `json:"warning_threshold,omitempty"`
	CriticalThreshold *float64      `json:"critical_threshold,omitempty"`
	RecoveryThreshold *float64      `json:"recovery_threshold,omitempty"`
	Duration          int           `json:"duration,omitempty"`
	Severity          string        `json:"severity,omitempty"`
	Enabled           bool          `json:"enabled"`
	Channels          []int64       `json:"channels,omitempty"`
	Tags              string        `json:"tags,omitempty"`
	NotifyWarning     bool          `json:"notify_warning"`
	NotifyCritical    bool          `json:"notify_critical"`
	CreatedAt         string        `json:"created_at,omitempty"`
}

// NotificationChannel is where alerts are delivered.
type NotificationChannel struct {
	ID        int64          `json:"id,omitempty"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Config    map[string]any `json:"config"`
	Enabled   bool           `json:"enabled"`
	CreatedAt string         `json:"created_at,omitempty"`
}

// MaintenanceWindow suppresses alerts on one target for a period.
type MaintenanceWindow struct {
	ID         int64  `json:"id,omitempty"`
	Name       string `json:"name"`
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// delete is the shared body-less DELETE, which every resource below needs.
func (c *Client) delete(path string) error {
	url := c.orgURL(path)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
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
		return &APIError{Method: http.MethodDelete, URL: url, StatusCode: resp.StatusCode, Body: string(raw)}
	}
	return nil
}

// ---- Host groups. The API lists and mutates them; there is no read-by-id, so
// Read finds the row in the list.

func (c *Client) ListHostGroups() ([]HostGroup, error) {
	var out []HostGroup
	_, err := c.doJSON(http.MethodGet, c.orgURL("/host-groups"), nil, &out)
	return out, err
}

func (c *Client) GetHostGroup(id int64) (*HostGroup, error) {
	groups, err := c.ListHostGroups()
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].ID == id {
			return &groups[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateHostGroup(g HostGroup) (*HostGroup, error) {
	var out HostGroup
	_, err := c.doJSON(http.MethodPost, c.orgURL("/host-groups"), g, &out)
	return &out, err
}

func (c *Client) UpdateHostGroup(id int64, g HostGroup) (*HostGroup, error) {
	var out HostGroup
	_, err := c.doJSON(http.MethodPut, c.orgURL(fmt.Sprintf("/host-groups/%d", id)), g, &out)
	return &out, err
}

func (c *Client) DeleteHostGroup(id int64) error {
	return c.delete(fmt.Sprintf("/host-groups/%d", id))
}

// ---- Alert rules

func (c *Client) ListAlertRules() ([]AlertRule, error) {
	var out []AlertRule
	_, err := c.doJSON(http.MethodGet, c.orgURL("/alert-rules"), nil, &out)
	return out, err
}

func (c *Client) GetAlertRule(id int64) (*AlertRule, error) {
	rules, err := c.ListAlertRules()
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateAlertRule(r AlertRule) (*AlertRule, error) {
	var out AlertRule
	_, err := c.doJSON(http.MethodPost, c.orgURL("/alert-rules"), r, &out)
	return &out, err
}

func (c *Client) UpdateAlertRule(id int64, r AlertRule) (*AlertRule, error) {
	var out AlertRule
	_, err := c.doJSON(http.MethodPut, c.orgURL(fmt.Sprintf("/alert-rules/%d", id)), r, &out)
	return &out, err
}

func (c *Client) DeleteAlertRule(id int64) error {
	return c.delete(fmt.Sprintf("/alert-rules/%d", id))
}

// ---- Notification channels

func (c *Client) ListNotificationChannels() ([]NotificationChannel, error) {
	var out []NotificationChannel
	_, err := c.doJSON(http.MethodGet, c.orgURL("/notification-channels"), nil, &out)
	return out, err
}

func (c *Client) GetNotificationChannel(id int64) (*NotificationChannel, error) {
	channels, err := c.ListNotificationChannels()
	if err != nil {
		return nil, err
	}
	for i := range channels {
		if channels[i].ID == id {
			return &channels[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateNotificationChannel(ch NotificationChannel) (*NotificationChannel, error) {
	var out NotificationChannel
	_, err := c.doJSON(http.MethodPost, c.orgURL("/notification-channels"), ch, &out)
	return &out, err
}

func (c *Client) UpdateNotificationChannel(id int64, ch NotificationChannel) (*NotificationChannel, error) {
	var out NotificationChannel
	_, err := c.doJSON(http.MethodPut, c.orgURL(fmt.Sprintf("/notification-channels/%d", id)), ch, &out)
	return &out, err
}

func (c *Client) DeleteNotificationChannel(id int64) error {
	return c.delete(fmt.Sprintf("/notification-channels/%d", id))
}

// ---- Maintenance windows. The API has no update: a change replaces the window.

func (c *Client) ListMaintenanceWindows() ([]MaintenanceWindow, error) {
	var out []MaintenanceWindow
	_, err := c.doJSON(http.MethodGet, c.orgURL("/maintenance-windows"), nil, &out)
	return out, err
}

func (c *Client) GetMaintenanceWindow(id int64) (*MaintenanceWindow, error) {
	windows, err := c.ListMaintenanceWindows()
	if err != nil {
		return nil, err
	}
	for i := range windows {
		if windows[i].ID == id {
			return &windows[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateMaintenanceWindow(w MaintenanceWindow) (*MaintenanceWindow, error) {
	var out MaintenanceWindow
	_, err := c.doJSON(http.MethodPost, c.orgURL("/maintenance-windows"), w, &out)
	return &out, err
}

func (c *Client) DeleteMaintenanceWindow(id int64) error {
	return c.delete(fmt.Sprintf("/maintenance-windows/%d", id))
}
