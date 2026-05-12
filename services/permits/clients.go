package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fieldstone/fieldstone/internal/validate"
)

// ─── Workflow client ──────────────────────────────────────────────────────────

type workflowClient struct {
	baseURL    string
	httpClient *http.Client
}

func newWorkflowClient(baseURL string) *workflowClient {
	return &workflowClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type validateRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Role string `json:"role"`
}

type validateResponse struct {
	Error string `json:"error"`
}

func (c *workflowClient) ValidateTransition(ctx context.Context, resourceType, from, to, role string) error {
	body, _ := json.Marshal(validateRequest{From: from, To: to, Role: role})
	url := fmt.Sprintf("%s/v1/workflow/%s/validate", c.baseURL, resourceType)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call workflow service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var vr validateResponse
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return fmt.Errorf("transition not allowed (status %d)", resp.StatusCode)
	}
	return fmt.Errorf("%s", vr.Error)
}

type initialStatusResponse struct {
	InitialStatus string `json:"initial_status"`
}

func (c *workflowClient) GetInitialStatus(ctx context.Context, resourceType string) (string, error) {
	url := fmt.Sprintf("%s/v1/workflow/%s/initial", c.baseURL, resourceType)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call workflow service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("workflow service returned %d", resp.StatusCode)
	}

	var r initialStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return r.InitialStatus, nil
}

// ─── Schema validator ─────────────────────────────────────────────────────────

type schemaValidator struct {
	identityBaseURL string
	httpClient      *http.Client
	validator       *validate.Cache

	mu     sync.RWMutex
	cached map[string]schemaEntry
}

type schemaEntry struct {
	schema    json.RawMessage
	expiresAt time.Time
}

type schemaResponse struct {
	Schema json.RawMessage `json:"schema"`
}

func newSchemaValidator(identityBaseURL string) *schemaValidator {
	return &schemaValidator{
		identityBaseURL: identityBaseURL,
		httpClient:      &http.Client{Timeout: 5 * time.Second},
		validator:       validate.NewCache(),
		cached:          make(map[string]schemaEntry),
	}
}

func (s *schemaValidator) ValidateMetadata(ctx context.Context, resourceType string, metadata json.RawMessage) ([]string, error) {
	schema, err := s.fetchSchema(ctx, resourceType)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, nil // no schema registered — skip validation
	}
	return s.validator.Validate(resourceType, schema, metadata)
}

func (s *schemaValidator) fetchSchema(ctx context.Context, resourceType string) (json.RawMessage, error) {
	s.mu.RLock()
	if e, ok := s.cached[resourceType]; ok && time.Now().Before(e.expiresAt) {
		schema := e.schema
		s.mu.RUnlock()
		return schema, nil
	}
	s.mu.RUnlock()

	url := fmt.Sprintf("%s/v1/config/schemas/%s", s.identityBaseURL, resourceType)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch schema: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		s.cacheSchema(resourceType, nil)
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identity service returned %d", resp.StatusCode)
	}

	var sr schemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode schema response: %w", err)
	}

	s.cacheSchema(resourceType, sr.Schema)
	return sr.Schema, nil
}

func (s *schemaValidator) cacheSchema(resourceType string, schema json.RawMessage) {
	s.mu.Lock()
	s.cached[resourceType] = schemaEntry{
		schema:    schema,
		expiresAt: time.Now().Add(60 * time.Second),
	}
	s.mu.Unlock()
}
