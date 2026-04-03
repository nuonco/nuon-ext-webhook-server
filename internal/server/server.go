package server

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/nuonco/nuon-ext-webhook-server/internal/payload"
)

const defaultMaxBodyBytes int64 = 1024 * 1024

type Config struct {
	Logger         *zap.Logger
	WebhookPath    string
	HealthPath     string
	ExpectedSecret string
	MaxBodyBytes   int64
}

type Server struct {
	logger         *zap.Logger
	webhookPath    string
	healthPath     string
	expectedSecret string
	maxBodyBytes   int64
	handler        http.Handler
}

type ackResponse struct {
	OK         bool      `json:"ok"`
	ReceivedAt time.Time `json:"received_at"`
	Transition string    `json:"transition,omitempty"`
	WorkflowID string    `json:"workflow_id,omitempty"`
	StepID     string    `json:"step_id,omitempty"`
	Phase      string    `json:"phase,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(cfg Config) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	webhookPath := normalizePath(cfg.WebhookPath)
	healthPath := normalizePath(cfg.HealthPath)
	if webhookPath == "/" {
		webhookPath = "/webhooks/flow/step-lifecycle"
	}
	if healthPath == "/" {
		healthPath = "/healthz"
	}

	maxBodyBytes := cfg.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultMaxBodyBytes
	}

	s := &Server{
		logger:         logger,
		webhookPath:    webhookPath,
		healthPath:     healthPath,
		expectedSecret: strings.TrimSpace(cfg.ExpectedSecret),
		maxBodyBytes:   maxBodyBytes,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.webhookPath, s.handleWebhook)
	mux.HandleFunc(s.healthPath, s.handleHealth)
	s.handler = mux

	return s
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, ackResponse{OK: true, ReceivedAt: time.Now().UTC()})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if s.expectedSecret != "" {
		providedSecret := strings.TrimSpace(r.Header.Get("X-Nuon-Webhook-Secret"))
		if providedSecret != s.expectedSecret {
			s.logger.Warn("rejecting webhook due to secret mismatch",
				zap.String("remote_addr", requestIP(r)),
				zap.String("webhook_path", s.webhookPath))
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid webhook secret"})
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.maxBodyBytes)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	var event payload.StepLifecycleWebhookPayload
	if err := decoder.Decode(&event); err != nil {
		s.logger.Warn("unable to decode webhook payload",
			zap.String("remote_addr", requestIP(r)),
			zap.Error(err))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON payload"})
		return
	}

	if err := event.Validate(); err != nil {
		s.logger.Warn("invalid webhook payload",
			zap.String("remote_addr", requestIP(r)),
			zap.Error(err))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	logFields := []zap.Field{
		zap.String("transition", event.Transition),
		zap.String("workflow_id", event.Event.WorkflowID),
		zap.String("workflow_type", event.Event.WorkflowType),
		zap.String("org_id", event.Event.OrgID),
		zap.String("install_id", event.Event.InstallID),
		zap.Bool("plan_only", event.Event.PlanOnly),
		zap.String("step_id", event.Event.StepID),
		zap.String("step_name", event.Event.StepName),
		zap.String("phase", event.Event.Phase),
		zap.Int("step_idx", event.Event.Descriptor.StepIdx),
		zap.Int("group_idx", event.Event.Descriptor.GroupIdx),
		zap.Int("group_retry_idx", event.Event.Descriptor.GroupRetryIdx),
		zap.String("execution_type", event.Event.Descriptor.ExecutionType),
		zap.String("signal_type", event.Event.Descriptor.SignalType),
		zap.String("step_target_type", event.Event.Descriptor.StepTargetType),
		zap.String("step_target_id", event.Event.Descriptor.StepTargetID),
		zap.String("step_kind", event.Event.Descriptor.Kind),
		zap.String("source_service", event.Source.Service),
		zap.String("source_kind", event.Source.Kind),
		zap.String("remote_addr", requestIP(r)),
		zap.String("user_agent", r.UserAgent()),
	}

	if event.Event.Descriptor.ComponentID != nil {
		logFields = append(logFields, zap.String("component_id", *event.Event.Descriptor.ComponentID))
	}
	if event.Event.Descriptor.ActionTriggerType != nil {
		logFields = append(logFields, zap.String("action_trigger_type", *event.Event.Descriptor.ActionTriggerType))
	}

	if event.Outcome != nil {
		logFields = append(logFields,
			zap.String("status", event.Outcome.Status),
			zap.String("err_message", event.Outcome.ErrMessage),
			zap.Duration("duration", event.Outcome.Duration),
			zap.Int("outcome_metadata_keys", len(event.Outcome.Metadata)),
		)
	}

	s.logger.Info("received step lifecycle webhook", logFields...)

	writeJSON(w, http.StatusOK, ackResponse{
		OK:         true,
		ReceivedAt: time.Now().UTC(),
		Transition: event.Transition,
		WorkflowID: event.Event.WorkflowID,
		StepID:     event.Event.StepID,
		Phase:      event.Event.Phase,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(v)
}

func requestIP(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}

	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	return "/" + trimmed
}
