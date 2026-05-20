package workflows

import (
	"fmt"
	"time"
)

// WorkflowConfig is the YAML-defined state machine for a resource type.
// It is passed to workflow functions at start time so the config is baked
// into Temporal's durable execution history alongside the workflow input.
type WorkflowConfig struct {
	ResourceType  string       `yaml:"resource_type"  json:"resource_type"`
	InitialStatus string       `yaml:"initial_status" json:"initial_status"`
	Statuses      []Status     `yaml:"statuses"       json:"statuses"`
	Transitions   []Transition `yaml:"transitions"    json:"transitions"`
}

type Status struct {
	Name            string `yaml:"name"                        json:"name"`
	Label           string `yaml:"label"                       json:"label"`
	Terminal        bool   `yaml:"terminal"                    json:"terminal"`
	AutoExpireAfter string `yaml:"auto_expire_after,omitempty" json:"auto_expire_after,omitempty"`
}

type Transition struct {
	From        string   `yaml:"from"                   json:"from"`
	To          string   `yaml:"to"                     json:"to"`
	Roles       []string `yaml:"roles"                  json:"roles"`
	NotifyEvent string   `yaml:"notify_event,omitempty" json:"notify_event,omitempty"`
}

// WorkflowInput is the input for all resource workflow functions.
type WorkflowInput struct {
	ResourceID string
	ResidentID *string
	Config     WorkflowConfig
	Deadline   *time.Time // optional: used by FOIA workflows to fire deadline notifications
}

// TransitionRequest is the Update payload sent by domain service handlers
// to validate and record a status transition.
type TransitionRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Role string `json:"role"`
}

// WithdrawSignal is the Signal payload sent by a resident to withdraw
// their own submission.
type WithdrawSignal struct {
	ResidentID string `json:"resident_id"`
}

// AutoExpireAfterDuration returns the configured auto-expiry duration for a status, or 0 if not set.
func (c WorkflowConfig) AutoExpireAfterDuration(statusName string) time.Duration {
	for _, s := range c.Statuses {
		if s.Name == statusName && s.AutoExpireAfter != "" {
			d, err := time.ParseDuration(s.AutoExpireAfter)
			if err != nil {
				return 0
			}
			return d
		}
	}
	return 0
}

// IsTerminal returns true if status is terminal in this config.
func (c WorkflowConfig) IsTerminal(status string) bool {
	for _, s := range c.Statuses {
		if s.Name == status && s.Terminal {
			return true
		}
	}
	return false
}

// ValidateTransition returns nil if the transition is allowed for the given role.
func (c WorkflowConfig) ValidateTransition(from, to, role string) error {
	for _, t := range c.Transitions {
		if t.From == from && t.To == to {
			for _, r := range t.Roles {
				if r == role {
					return nil
				}
			}
			return fmt.Errorf("role %q not allowed for transition %s→%s", role, from, to)
		}
	}
	return fmt.Errorf("no transition defined: %s→%s", from, to)
}
