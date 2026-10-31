package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	TenantIDKey  contextKey = "tenant_id"
	UserIDKey    contextKey = "user_id"
)

// Handler wraps slog.JSONHandler to automatically enrich log records with context values.
type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
			r.AddAttrs(slog.String("request_id", reqID))
		}
		if tenantID, ok := ctx.Value(TenantIDKey).(string); ok && tenantID != "" {
			r.AddAttrs(slog.String("tenant_id", tenantID))
		}
		if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
			r.AddAttrs(slog.String("user_id", userID))
		}
	}
	return h.Handler.Handle(ctx, r)
}

// New creates a new structured JSON logger configured with log level.
func New(env string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}

	var level slog.Level
	if strings.ToLower(env) == "production" {
		level = slog.LevelInfo
	} else {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Redact sensitive keys
			keyLower := strings.ToLower(a.Key)
			if strings.Contains(keyLower, "password") ||
				strings.Contains(keyLower, "secret") ||
				strings.Contains(keyLower, "token") ||
				strings.Contains(keyLower, "authorization") {
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	}

	baseHandler := slog.NewJSONHandler(out, opts)
	return slog.New(&contextHandler{Handler: baseHandler})
}

// WithRequestID adds a request ID to the given context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context if present.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

