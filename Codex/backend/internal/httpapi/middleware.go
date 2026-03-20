package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type contextKey string

const (
	ctxKeyUser    contextKey = "user"
	ctxKeySession contextKey = "session"
)

func withUser(ctx context.Context, user any) context.Context {
	return context.WithValue(ctx, ctxKeyUser, user)
}

func withSession(ctx context.Context, session any) context.Context {
	return context.WithValue(ctx, ctxKeySession, session)
}

func userFromContext(ctx context.Context) any {
	return ctx.Value(ctxKeyUser)
}

func sessionFromContext(ctx context.Context) any {
	return ctx.Value(ctxKeySession)
}

// requestLogger writes structured request logs.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)
			logger.Info("http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", r.Header.Get("X-Request-ID"),
				"remote", r.RemoteAddr,
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
