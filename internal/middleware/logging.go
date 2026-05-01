package middleware

import (
	"net/http"
	"time"

	"bedrock-gateway/internal/admin"
	"bedrock-gateway/internal/convert"
)

// responseCapture captures the status code while preserving http.Flusher.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher by delegating to the wrapped ResponseWriter.
func (rc *responseCapture) Flush() {
	if f, ok := rc.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logging returns a middleware that logs requests to the RequestLogger.
func Logging(rl *admin.RequestLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			capture := &responseCapture{ResponseWriter: w, statusCode: 200}

			next.ServeHTTP(capture, r)

			entry := admin.RequestLog{
				ID:         convert.GenerateMessageID(),
				Timestamp:  start,
				Method:     r.Method,
				Path:       r.URL.Path,
				ClientIP:   r.RemoteAddr,
				StatusCode: capture.statusCode,
				Latency:    time.Since(start),
			}

			rl.Log(entry)
		})
	}
}
