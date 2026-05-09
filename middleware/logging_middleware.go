package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Ludo/utils"
)

// LoggingMiddleware logs all incoming HTTP requests and responses with Prometheus metrics
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip verbose logging for polling endpoints
		// Skip logging for specific endpoints
		skipLogging := strings.Contains(r.URL.Path, "/state") ||
			strings.Contains(r.URL.Path, "/metrics/prometheus")

		// Only log detailed info for non-polling requests
		if !skipLogging {
			var requestBody string
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					requestBody = string(bodyBytes)
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			// Use utils.InfoLogger instead of log.Printf
			utils.InfoLogger.Println("═══════════════════════════════════════════════════════════")
			utils.InfoLogger.Printf("📥 INCOMING REQUEST")
			utils.InfoLogger.Printf("Method: %s", r.Method)
			utils.InfoLogger.Printf("Path: %s", r.URL.Path)
			utils.InfoLogger.Printf("IP: %s", r.RemoteAddr)
			utils.InfoLogger.Printf("User-Agent: %s", r.Header.Get("User-Agent"))

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				if len(authHeader) > 20 {
					utils.InfoLogger.Printf("Auth: %s...%s", authHeader[:10], authHeader[len(authHeader)-5:])
				} else {
					utils.InfoLogger.Printf("Auth: %s", authHeader)
				}
			}

			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				if requestBody != "" {
					maskedBody := maskSensitiveData(requestBody)
					if len(maskedBody) > 500 {
						utils.InfoLogger.Printf("Body: %s... (truncated)", maskedBody[:500])
					} else {
						utils.InfoLogger.Printf("Body: %s", maskedBody)
					}
				} else {
					utils.InfoLogger.Printf("Body: (empty)")
				}
			}
		} else {
			// Simple log for polling requests
			utils.InfoLogger.Printf("🔄 [%s] %s", r.Method, r.URL.Path)
		}

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// ===== PROMETHEUS METRICS =====
		normalizedPath := normalizePath(r.URL.Path)

		utils.HttpRequestsTotal.WithLabelValues(
			r.Method,
			normalizedPath,
			strconv.Itoa(rw.statusCode),
		).Inc()

		utils.HttpRequestDuration.WithLabelValues(
			r.Method,
			normalizedPath,
		).Observe(duration.Seconds())

		if rw.statusCode >= 400 {
			errorType := "client_error"
			if rw.statusCode >= 500 {
				errorType = "server_error"
			}
			utils.ApiErrorsTotal.WithLabelValues(normalizedPath, errorType).Inc()
		}
		// ===== END PROMETHEUS METRICS =====

		if !skipLogging {
			statusEmoji := getStatusEmoji(rw.statusCode)
			utils.InfoLogger.Println("───────────────────────────────────────────────────────────")
			utils.InfoLogger.Printf("📤 RESPONSE")
			utils.InfoLogger.Printf("Status: %s %d", statusEmoji, rw.statusCode)
			utils.InfoLogger.Printf("Duration: %v", duration)

			//responseBody := rw.body.String()
			//if responseBody != "" {
			//	if len(responseBody) > 1000 {
			//		utils.InfoLogger.Printf("Body: %s... (truncated)", responseBody[:1000])
			//	} else {
			//		utils.InfoLogger.Printf("Body: %s", responseBody)
			//	}
			//}

			//utils.InfoLogger.Println("═══════════════════════════════════════════════════════════")
		}
	})
}

// normalizePath removes dynamic segments (IDs) from path for better grouping
func normalizePath(path string) string {
	// Replace UUIDs with {id}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Check if part looks like a UUID or ID
		if len(part) == 36 && strings.Count(part, "-") == 4 {
			parts[i] = "{id}"
		} else if len(part) > 0 && isAlphanumeric(part) && !isKnownSegment(part) {
			// Replace other potential IDs
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

func isKnownSegment(s string) bool {
	known := []string{
		"api", "game", "user", "auth", "health", "metrics", "swagger",
		"create", "join", "start", "leave", "state", "active", "history",
		"roll-dice", "move-token", "skip-turn", "profile", "signup", "login",
	}
	for _, k := range known {
		if s == k {
			return true
		}
	}
	return false
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func getStatusEmoji(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "✅"
	case statusCode >= 300 && statusCode < 400:
		return "🔄"
	case statusCode >= 400 && statusCode < 500:
		return "⚠️"
	case statusCode >= 500:
		return "❌"
	default:
		return "❓"
	}
}

func maskSensitiveData(body string) string {
	sensitiveFields := []string{
		"password", "token", "secret", "api_key", "apikey", "authorization", "credit_card",
	}

	maskedBody := body
	for _, field := range sensitiveFields {
		searchPattern := `"` + field + `"`
		if strings.Contains(strings.ToLower(maskedBody), strings.ToLower(searchPattern)) {
			lowerBody := strings.ToLower(maskedBody)
			idx := strings.Index(lowerBody, strings.ToLower(searchPattern))
			if idx != -1 {
				colonIdx := strings.Index(maskedBody[idx:], ":")
				if colonIdx != -1 {
					colonIdx += idx
					valueStart := colonIdx + 1
					for valueStart < len(maskedBody) && (maskedBody[valueStart] == ' ' || maskedBody[valueStart] == '\t') {
						valueStart++
					}

					if valueStart < len(maskedBody) && maskedBody[valueStart] == '"' {
						valueEnd := strings.Index(maskedBody[valueStart+1:], `"`)
						if valueEnd != -1 {
							valueEnd += valueStart + 1
							maskedBody = maskedBody[:valueStart+1] + "***MASKED***" + maskedBody[valueEnd:]
						}
					}
				}
			}
		}
	}

	return maskedBody
}
