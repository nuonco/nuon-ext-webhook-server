package payload

import (
	"errors"
	"fmt"
	"strings"
)

const (
	TransitionBefore = "before"
	TransitionAfter  = "after"
)

// CloudEvent represents a CloudEvents v1.0 envelope for Nuon webhook deliveries.
type CloudEvent struct {
	SpecVersion     string                    `json:"specversion"`
	ID              string                    `json:"id"`
	Type            string                    `json:"type"`
	Source          string                    `json:"source"`
	Time            string                    `json:"time,omitempty"`
	Subject         string                    `json:"subject,omitempty"`
	DataContentType string                    `json:"datacontenttype,omitempty"`
	NuonOrgID       string                    `json:"nuonorgid,omitempty"`
	NuonOperation   string                    `json:"nuonoperation,omitempty"`
	NuonTransition  string                    `json:"nuontransition"`
	Data            *SignalLifecycleEventData `json:"data"`
}

type SignalLifecycleEventData struct {
	Signal    SignalIdentity `json:"signal"`
	Phase     string         `json:"phase"`
	Operation string         `json:"operation"`
	Context   SignalContext  `json:"context"`
	Outcome   *SignalOutcome `json:"outcome"`
}

type SignalIdentity struct {
	ID       string  `json:"id"`
	QueueID  string  `json:"queue_id"`
	Type     string  `json:"type"`
	ParentID *string `json:"parent_id"`
}

type SignalContext struct {
	OrgID       string  `json:"org_id"`
	AppID       *string `json:"app_id"`
	InstallID   *string `json:"install_id"`
	ComponentID *string `json:"component_id"`
	SandboxID   *string `json:"sandbox_id"`
	RunnerID    *string `json:"runner_id"`
}

type SignalOutcome struct {
	Status     string         `json:"status"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Metadata   map[string]any `json:"metadata,omitempty"`
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

	if strings.TrimSpace(ce.Source) == "" {
		return errors.New("source is required")
	}

	switch ce.NuonTransition {
	case TransitionBefore, TransitionAfter:
	default:
		return fmt.Errorf("invalid nuontransition %q", ce.NuonTransition)
	}

	if ce.Data == nil {
		return errors.New("data is required")
	}

	if strings.TrimSpace(ce.Data.Signal.ID) == "" {
		return errors.New("data.signal.id is required")
	}

	if strings.TrimSpace(ce.Data.Phase) == "" {
		return errors.New("data.phase is required")
	}

	return nil
}
