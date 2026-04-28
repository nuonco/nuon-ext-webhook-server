package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/nuonco/nuon-ext-webhook-server/internal/payload"
)

const testWebhookPath = "/webhooks/operation/lifecycle"

func TestWebhookHandlerAcceptsValidPayload(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Logger:      zap.NewNop(),
		WebhookPath: testWebhookPath,
		HealthPath:  "/healthz",
	})

	body, err := json.Marshal(validCloudEvent())
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, testWebhookPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/cloudevents+json; charset=utf-8")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusOK, rec.Code, rec.Body.String())
	}

	var ack ackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &ack); err != nil {
		t.Fatalf("decode ack: %v", err)
	}

	if !ack.OK {
		t.Fatalf("expected ok=true")
	}
	if ack.Event != payload.EventPlanFinished {
		t.Fatalf("expected event=%q, got %q", payload.EventPlanFinished, ack.Event)
	}
	if ack.Operation != "component-deploy" {
		t.Fatalf("expected operation=component-deploy, got %q", ack.Operation)
	}
	if ack.Stage != payload.StagePlan {
		t.Fatalf("expected stage=plan, got %q", ack.Stage)
	}
	if ack.Status != payload.StatusSucceeded {
		t.Fatalf("expected status=succeeded, got %q", ack.Status)
	}
}

func TestWebhookHandlerRejectsInvalidType(t *testing.T) {
	t.Parallel()

	srv := New(Config{Logger: zap.NewNop(), WebhookPath: testWebhookPath})

	ce := validCloudEvent()
	ce.Type = "com.nuon.signal.lifecycle.v1"

	body, err := json.Marshal(ce)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, testWebhookPath, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWebhookHandlerRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	srv := New(Config{Logger: zap.NewNop(), WebhookPath: testWebhookPath})

	invalidPayload := map[string]any{
		"specversion": "1.0",
		"id":          "evt_123",
		"type":        payload.CloudEventType,
		"source":      "//nuon.co/ctl-api",
		"data": map[string]any{
			"event":     "plan.started",
			"operation": "",
			"status":    "started",
			"context":   map[string]any{"org_id": "org_123"},
		},
	}

	body, err := json.Marshal(invalidPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, testWebhookPath, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWebhookHandlerAcceptsValidSignature(t *testing.T) {
	t.Parallel()

	const secret = "shh"

	srv := New(Config{
		Logger:         zap.NewNop(),
		WebhookPath:    testWebhookPath,
		ExpectedSecret: secret,
	})

	body, err := json.Marshal(validCloudEvent())
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, testWebhookPath, bytes.NewReader(body))
	req.Header.Set("X-Nuon-Signature", sig)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestWebhookHandlerRejectsSecretMismatch(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Logger:         zap.NewNop(),
		WebhookPath:    testWebhookPath,
		ExpectedSecret: "expected",
	})

	body, err := json.Marshal(validCloudEvent())
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, testWebhookPath, bytes.NewReader(body))
	req.Header.Set("X-Nuon-Signature", "deadbeef")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWebhookHandlerRejectsMissingSignature(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Logger:         zap.NewNop(),
		WebhookPath:    testWebhookPath,
		ExpectedSecret: "expected",
	})

	body, err := json.Marshal(validCloudEvent())
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, testWebhookPath, bytes.NewReader(body))
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

func validCloudEvent() payload.CloudEvent {
	installID := "ins_123"
	componentID := "cmp_123"

	return payload.CloudEvent{
		SpecVersion:     "1.0",
		ID:              "evt_123",
		Type:            payload.CloudEventType,
		Source:          "//nuon.co/ctl-api",
		Time:            "2026-04-28T00:00:00Z",
		Subject:         "org_123/component-deploy/plan/plan.finished",
		DataContentType: "application/json",
		NuonOrgID:       "org_123",
		NuonOperation:   "component-deploy",
		NuonStage:       payload.StagePlan,
		NuonStatus:      payload.StatusSucceeded,
		Data: &payload.OperationLifecycleEventData{
			Event:      payload.EventPlanFinished,
			Operation:  "component-deploy",
			Stage:      payload.StagePlan,
			Status:     payload.StatusSucceeded,
			DurationMs: 1234,
			Context: payload.OperationContext{
				OrgID:       "org_123",
				InstallID:   &installID,
				ComponentID: &componentID,
			},
		},
	}
}
