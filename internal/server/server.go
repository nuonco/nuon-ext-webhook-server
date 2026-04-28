package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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
	OK         bool   `json:"ok"`
	ReceivedAt string `json:"received_at"`
	EventID    string `json:"event_id,omitempty"`
	Event      string `json:"event,omitempty"`
	Operation  string `json:"operation,omitempty"`
	Stage      string `json:"stage,omitempty"`
	Status     string `json:"status,omitempty"`
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
		webhookPath = "/webhooks/operation/lifecycle"
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

	writeJSON(w, http.StatusOK, ackResponse{OK: true, ReceivedAt: time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.maxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Warn("unable to read webhook body",
			zap.String("remote_addr", requestIP(r)),
			zap.Error(err))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unable to read request body"})
		return
	}

	if s.expectedSecret != "" {
		providedSig := strings.TrimSpace(r.Header.Get("X-Nuon-Signature"))
		if providedSig == "" {
			s.logger.Warn("rejecting webhook: missing X-Nuon-Signature",
				zap.String("remote_addr", requestIP(r)))
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing signature"})
			return
		}

		mac := hmac.New(sha256.New, []byte(s.expectedSecret))
		mac.Write(body)
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
			s.logger.Warn("rejecting webhook: signature mismatch",
				zap.String("remote_addr", requestIP(r)),
				zap.String("webhook_path", s.webhookPath))
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid signature"})
			return
		}
	}

	var event payload.CloudEvent
	if err := json.Unmarshal(body, &event); err != nil {
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

	prettyJSON, _ := json.MarshalIndent(event, "", "  ")
	s.logger.Info("received operation lifecycle webhook",
		zap.String("remote_addr", requestIP(r)),
		zap.String("user_agent", r.UserAgent()),
		zap.String("event", event.Data.Event),
		zap.String("operation", event.Data.Operation),
		zap.String("stage", event.Data.Stage),
		zap.String("status", event.Data.Status),
	)
	fmt.Fprintf(os.Stdout, "\n%s\n\n", prettyJSON)

	writeJSON(w, http.StatusOK, ackResponse{
		OK:         true,
		ReceivedAt: time.Now().UTC().Format(time.RFC3339),
		EventID:    event.ID,
		Event:      event.Data.Event,
		Operation:  event.Data.Operation,
		Stage:      event.Data.Stage,
		Status:     event.Data.Status,
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
