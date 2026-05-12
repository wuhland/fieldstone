package main

import "fmt"

// Engine validates workflow transitions and answers queries about valid states.
type Engine struct {
	workflows map[string]*WorkflowConfig
}

func NewEngine(workflows map[string]*WorkflowConfig) *Engine {
	return &Engine{workflows: workflows}
}

type ValidationRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Role string `json:"role"`
}

type ValidationError struct {
	Message string `json:"message"`
}

// Validate checks whether a transition is allowed for the given role.
// Returns nil if allowed, *ValidationError if not.
func (e *Engine) Validate(resourceType string, req ValidationRequest) *ValidationError {
	wf, ok := e.workflows[resourceType]
	if !ok {
		return &ValidationError{Message: fmt.Sprintf("unknown resource type: %s", resourceType)}
	}
	for _, t := range wf.Transitions {
		if t.From == req.From && t.To == req.To {
			for _, r := range t.Roles {
				if r == req.Role || r == "system" {
					return nil
				}
			}
			return &ValidationError{Message: fmt.Sprintf("role %q not allowed for transition %s→%s", req.Role, req.From, req.To)}
		}
	}
	return &ValidationError{Message: fmt.Sprintf("no transition defined: %s→%s", req.From, req.To)}
}

func (e *Engine) Statuses(resourceType string) ([]Status, error) {
	wf, ok := e.workflows[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return wf.Statuses, nil
}

func (e *Engine) Transitions(resourceType string) ([]Transition, error) {
	wf, ok := e.workflows[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return wf.Transitions, nil
}

func (e *Engine) InitialStatus(resourceType string) (string, error) {
	wf, ok := e.workflows[resourceType]
	if !ok {
		return "", fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return wf.InitialStatus, nil
}
