package payload

import (
	"errors"
	"fmt"
	"strings"
)

// CloudEventType is the only event type emitted by ctl-api operation
// lifecycle webhooks.
const CloudEventType = "com.nuon.operation.lifecycle.v1"

// Status values surfaced in operation lifecycle events.
const (
	StatusStarted   = "started"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"
)

// Stage values. Empty stage means a single-phase operation.
const (
	StagePlan  = "plan"
	StageApply = "apply"
)

// Event names emitted in data.event.
const (
	EventPlanStarted       = "plan.started"
	EventPlanFinished      = "plan.finished"
	EventApplyStarted      = "apply.started"
	EventApplyFinished     = "apply.finished"
	EventOperationStarted  = "operation.started"
	EventOperationFinished = "operation.finished"
)

// Failure reasons surfaced when status is "failed".
const (
	FailureReasonValidationFailed = "validation_failed"
)

// CloudEvent represents a CloudEvents v1.0 envelope for Nuon operation
// lifecycle webhook deliveries.
type CloudEvent struct {
	SpecVersion     string `json:"specversion"`
	ID              string `json:"id"`
	Type            string `json:"type"`
	Source          string `json:"source"`
	Time            string `json:"time,omitempty"`
	Subject         string `json:"subject,omitempty"`
	DataContentType string `json:"datacontenttype,omitempty"`

	NuonOrgID     string `json:"nuonorgid,omitempty"`
	NuonOperation string `json:"nuonoperation,omitempty"`
	NuonStage     string `json:"nuonstage,omitempty"`
	NuonStatus    string `json:"nuonstatus,omitempty"`

	Data *OperationLifecycleEventData `json:"data"`
}

type OperationLifecycleEventData struct {
	Event         string           `json:"event"`
	Operation     string           `json:"operation"`
	Stage         string           `json:"stage,omitempty"`
	Status        string           `json:"status"`
	FailureReason string           `json:"failure_reason,omitempty"`
	Error         string           `json:"error,omitempty"`
	DurationMs    int64            `json:"duration_ms,omitempty"`
	Context       OperationContext `json:"context"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
}

type OperationContext struct {
	OrgID       string  `json:"org_id"`
	InstallID   *string `json:"install_id,omitempty"`
	ComponentID *string `json:"component_id,omitempty"`
	SandboxID   *string `json:"sandbox_id,omitempty"`
}

func (ce *CloudEvent) Validate() error {
	if ce == nil {
		return errors.New("payload is nil")
	}

	if ce.SpecVersion != "1.0" {
		return fmt.Errorf("unsupported specversion %q", ce.SpecVersion)
	}

	if strings.TrimSpace(ce.ID) == "" {
		return errors.New("id is required")
	}

	if strings.TrimSpace(ce.Type) == "" {
		return errors.New("type is required")
	}

	if ce.Type != CloudEventType {
		return fmt.Errorf("unsupported type %q (expected %q)", ce.Type, CloudEventType)
	}

	if strings.TrimSpace(ce.Source) == "" {
		return errors.New("source is required")
	}

	if ce.Data == nil {
		return errors.New("data is required")
	}

	if strings.TrimSpace(ce.Data.Event) == "" {
		return errors.New("data.event is required")
	}

	if strings.TrimSpace(ce.Data.Operation) == "" {
		return errors.New("data.operation is required")
	}

	switch ce.Data.Status {
	case StatusStarted, StatusSucceeded, StatusFailed, StatusCanceled:
	default:
		return fmt.Errorf("invalid data.status %q", ce.Data.Status)
	}

	switch ce.Data.Event {
	case EventPlanStarted, EventPlanFinished,
		EventApplyStarted, EventApplyFinished,
		EventOperationStarted, EventOperationFinished:
	default:
		return fmt.Errorf("invalid data.event %q", ce.Data.Event)
	}

	if strings.TrimSpace(ce.Data.Context.OrgID) == "" {
		return errors.New("data.context.org_id is required")
	}

	return nil
}
