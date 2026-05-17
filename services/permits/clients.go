package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
	enumspb "go.temporal.io/api/enums/v1"

	temporalclient "github.com/fieldstone/fieldstone/internal/temporal"
	"github.com/fieldstone/fieldstone/internal/validate"
	wftypes "github.com/fieldstone/fieldstone/internal/workflows"
)

// ─── Workflow client ──────────────────────────────────────────────────────────

// temporalWorkflowClient implements handlers.WorkflowClient using Temporal for
// durable transition validation and HTTP for initial-status queries and fallback.
type temporalWorkflowClient struct {
	temporal   client.Client
	httpClient *http.Client
	baseURL    string // workflow-worker HTTP base URL
}

func newWorkflowClient(temporalHost, workerBaseURL string) (*temporalWorkflowClient, error) {
	tc, err := temporalclient.NewClient(temporalHost)
	if err != nil {
		return nil, fmt.Errorf("connect to temporal: %w", err)
	}
	return &temporalWorkflowClient{
		temporal:   tc,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    workerBaseURL,
	}, nil
}

// GetInitialStatus calls the workflow-worker HTTP endpoint.
func (c *temporalWorkflowClient) GetInitialStatus(ctx context.Context, resourceType string) (string, error) {
	url := fmt.Sprintf("%s/v1/workflow/%s/initial", c.baseURL, resourceType)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call workflow worker: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("workflow worker returned %d", resp.StatusCode)
	}
	var r struct {
		InitialStatus string `json:"initial_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return r.InitialStatus, nil
}

// StartWorkflow starts a durable Temporal workflow for a newly created resource.
// Fetches the YAML config from the workflow-worker so it is baked into the
// workflow's durable input. Non-fatal: if Temporal is unavailable the resource
// still exists in the DB and status updates fall back to HTTP validation.
func (c *temporalWorkflowClient) StartWorkflow(ctx context.Context, resourceType, resourceID string, residentID *string) error {
	cfg, err := c.fetchConfig(ctx, resourceType)
	if err != nil {
		return fmt.Errorf("fetch workflow config: %w", err)
	}
	input := wftypes.WorkflowInput{
		ResourceID: resourceID,
		ResidentID: residentID,
		Config:     *cfg,
	}
	opts := client.StartWorkflowOptions{
		ID:                       temporalclient.WorkflowID(resourceType, resourceID),
		TaskQueue:                temporalclient.TaskQueue,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionTimeout: 365 * 24 * time.Hour,
	}
	_, err = c.temporal.ExecuteWorkflow(ctx, opts, workflowNameFor(resourceType), input)
	return err
}

func (c *temporalWorkflowClient) fetchConfig(ctx context.Context, resourceType string) (*wftypes.WorkflowConfig, error) {
	url := fmt.Sprintf("%s/v1/workflow/%s/config", c.baseURL, resourceType)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workflow worker returned %d", resp.StatusCode)
	}
	var cfg wftypes.WorkflowConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &cfg, nil
}

// ValidateTransition sends a Temporal Update to the running workflow.
// If no workflow execution exists (records that predate Temporal), it falls back
// to the HTTP validation endpoint on the workflow-worker.
func (c *temporalWorkflowClient) ValidateTransition(ctx context.Context, resourceType, resourceID, from, to, role string) error {
	workflowID := temporalclient.WorkflowID(resourceType, resourceID)
	handle, err := c.temporal.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   "validate-transition",
		Args:         []interface{}{wftypes.TransitionRequest{From: from, To: to, Role: role}},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		if isWorkflowNotFound(err) {
			return c.validateHTTP(ctx, resourceType, from, to, role)
		}
		return fmt.Errorf("temporal update: %w", err)
	}
	return handle.Get(ctx, nil)
}

func (c *temporalWorkflowClient) validateHTTP(ctx context.Context, resourceType, from, to, role string) error {
	body, _ := json.Marshal(wftypes.TransitionRequest{From: from, To: to, Role: role})
	url := fmt.Sprintf("%s/v1/workflow/%s/validate", c.baseURL, resourceType)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call workflow worker: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	var r struct{ Error string `json:"error"` }
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("transition not allowed (status %d)", resp.StatusCode)
	}
	return fmt.Errorf("%s", r.Error)
}

func isWorkflowNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "workflow not found") ||
		strings.Contains(s, "no current workflow") ||
		strings.Contains(s, "WORKFLOW_NOT_FOUND")
}

// workflowNameFor returns the Temporal workflow function name for a resource type.
// Must match the names registered in the worker's RegisterWorkflow calls.
func workflowNameFor(resourceType string) string {
	m := map[string]string{
		"permit":          "PermitWorkflow",
		"service_request": "ServiceRequestWorkflow",
		"foia_request":    "FOIARequestWorkflow",
	}
	return m[resourceType]
}

// ─── Schema validator ─────────────────────────────────────────────────────────

type schemaValidator struct {
	identityBaseURL string
	httpClient      *http.Client
	validator       *validate.Cache
	mu              sync.RWMutex
	cached          map[string]schemaEntry
}

type schemaEntry struct {
	schema    json.RawMessage
	expiresAt time.Time
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
		return nil, nil
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
	var sr struct{ Schema json.RawMessage `json:"schema"` }
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode schema response: %w", err)
	}
	s.cacheSchema(resourceType, sr.Schema)
	return sr.Schema, nil
}

func (s *schemaValidator) cacheSchema(resourceType string, schema json.RawMessage) {
	s.mu.Lock()
	s.cached[resourceType] = schemaEntry{schema: schema, expiresAt: time.Now().Add(60 * time.Second)}
	s.mu.Unlock()
}
