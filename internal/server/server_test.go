package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/nuonco/nuon-ext-webhook-server/internal/payload"
)

func TestWebhookHandlerAcceptsValidPayload(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Logger:      zap.NewNop(),
		WebhookPath: "/webhooks/flow/step-lifecycle",
		HealthPath:  "/healthz",
	})

	body, err := json.Marshal(validPayload())
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/flow/step-lifecycle", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWebhookHandlerRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	srv := New(Config{Logger: zap.NewNop()})

	invalidPayload := map[string]any{
		"version":    "v1",
		"transition": "before",
		"event": map[string]any{
			"workflow_id": "",
			"step_id":     "step_123",
			"phase":       "execute",
		},
	}

	body, err := json.Marshal(invalidPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/flow/step-lifecycle", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWebhookHandlerRejectsSecretMismatch(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Logger:         zap.NewNop(),
		ExpectedSecret: "expected",
	})

	body, err := json.Marshal(validPayload())
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/flow/step-lifecycle", bytes.NewReader(body))
	req.Header.Set("X-Nuon-Webhook-Secret", "wrong")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	srv := New(Config{Logger: zap.NewNop()})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func validPayload() payload.StepLifecycleWebhookPayload {
	componentID := "cmp_123"
	actionTriggerType := "deploy"

	return payload.StepLifecycleWebhookPayload{
		Version:    "v1",
		Transition: payload.TransitionAfter,
		Event: payload.StepPhaseEvent{
			WorkflowID:   "wf_123",
			WorkflowType: "install_deploy",
			OrgID:        "org_123",
			InstallID:    "ins_123",
			StepID:       "step_123",
			StepName:     "plan",
			Phase:        "plan-policy-check",
			Descriptor: payload.StepDescriptor{
				StepIdx:           1,
				GroupIdx:          0,
				GroupRetryIdx:     0,
				ExecutionType:     "approval",
				SignalType:        "execute_deploy_component_sync_and_plan",
				StepTargetType:    "install_deploys",
				StepTargetID:      "dep_123",
				Kind:              "component-plan",
				ComponentID:       &componentID,
				ActionTriggerType: &actionTriggerType,
			},
		},
		Outcome: &payload.StepPhaseOutcome{Status: "success"},
		Source: payload.WebhookPayloadSource{
			Service: "ctl-api",
			Kind:    "step_lifecycle_hook",
		},
	}
}
