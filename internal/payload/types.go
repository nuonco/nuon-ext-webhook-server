package payload

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TransitionBefore = "before"
	TransitionAfter  = "after"
)

type StepLifecycleWebhookPayload struct {
	Version    string               `json:"version"`
	Transition string               `json:"transition"`
	SentAt     time.Time            `json:"sent_at"`
	Event      StepPhaseEvent       `json:"event"`
	Outcome    *StepPhaseOutcome    `json:"outcome,omitempty"`
	Source     WebhookPayloadSource `json:"source"`
}

type WebhookPayloadSource struct {
	Service string `json:"service"`
	Kind    string `json:"kind"`
}

type StepPhaseEvent struct {
	WorkflowID   string `json:"workflow_id"`
	WorkflowType string `json:"workflow_type"`
	OrgID        string `json:"org_id"`
	InstallID    string `json:"install_id"`
	PlanOnly     bool   `json:"plan_only"`

	StepID   string `json:"step_id"`
	StepName string `json:"step_name"`
	Phase    string `json:"phase"`

	Descriptor StepDescriptor `json:"descriptor"`
}

type StepDescriptor struct {
	StepIdx        int    `json:"step_idx"`
	GroupIdx       int    `json:"group_idx"`
	GroupRetryIdx  int    `json:"group_retry_idx"`
	ExecutionType  string `json:"execution_type"`
	SignalType     string `json:"signal_type"`
	StepTargetType string `json:"step_target_type"`
	StepTargetID   string `json:"step_target_id"`
	Kind           string `json:"kind"`

	ComponentID       *string `json:"component_id,omitempty"`
	ActionTriggerType *string `json:"action_trigger_type,omitempty"`
}

type StepPhaseOutcome struct {
	Status     string         `json:"status"`
	ErrMessage string         `json:"err_message,omitempty"`
	Duration   time.Duration  `json:"duration,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

func (p *StepLifecycleWebhookPayload) Validate() error {
	if p == nil {
		return errors.New("payload is nil")
	}

	if strings.TrimSpace(p.Version) == "" {
		return errors.New("version is required")
	}

	switch p.Transition {
	case TransitionBefore, TransitionAfter:
	default:
		return fmt.Errorf("invalid transition %q", p.Transition)
	}

	if strings.TrimSpace(p.Event.WorkflowID) == "" {
		return errors.New("event.workflow_id is required")
	}

	if strings.TrimSpace(p.Event.StepID) == "" {
		return errors.New("event.step_id is required")
	}

	if strings.TrimSpace(p.Event.Phase) == "" {
		return errors.New("event.phase is required")
	}

	return nil
}
