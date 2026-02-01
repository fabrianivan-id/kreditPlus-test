package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"kreditplus-test/pkg/config"
)

type Middleware struct {
	apiKey         string
	bodyLimitBytes int64
	allowedOrigins map[string]struct{}
	rateLimiter    *rateLimiter
	logger         *log.Logger
}

func NewMiddleware(cfg config.Config, logger *log.Logger) *Middleware {
	origins := make(map[string]struct{})
	for _, o := range cfg.AllowedOrigins {
		origins[o] = struct{}{}
	}

	return &Middleware{
		apiKey:         cfg.APIKey,
		bodyLimitBytes: cfg.BodyLimitBytes,
		allowedOrigins: origins,
		rateLimiter:    newRateLimiter(cfg.RateLimitPerMin, time.Minute),
		logger:         logger,
	}
}

func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func (m *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				m.logger.Printf("panic: %v", rec)
				writeError(w, http.StatusInternalServerError, errInternal)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		m.logger.Printf("%s %s %d %dB %s", r.Method, r.URL.Path, rec.status, rec.bytes, time.Since(start))
	})
}

func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && m.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ip != "" && !m.rateLimiter.Allow(ip) {
			writeError(w, http.StatusTooManyRequests, errRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.bodyLimitBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, m.bodyLimitBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) APIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.apiKey == "" || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-API-Key") != m.apiKey {
			writeError(w, http.StatusUnauthorized, errUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

type requestIDKey struct{}

type rateLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	clients     map[string]*rateClient
}

type rateClient struct {
	count       int
	windowStart time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 60
	}
	return &rateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]*rateClient),
	}
}

func (rl *rateLimiter) Allow(key string) bool {
	if key == "" {
		return true
	}
	if rl.limit <= 0 {
		return true
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, ok := rl.clients[key]
	if !ok {
		rl.clients[key] = &rateClient{count: 1, windowStart: time.Now()}
		return true
	}

	now := time.Now()
	if now.Sub(client.windowStart) >= rl.window {
		client.windowStart = now
		client.count = 1
		return true
	}

	if client.count >= rl.limit {
		return false
	}
	client.count++
	return true
}

func newRequestID() string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func clientIP(r *http.Request) string {
	ip := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (m *Middleware) isOriginAllowed(origin string) bool {
	if len(m.allowedOrigins) == 0 {
		return false
	}
	if _, ok := m.allowedOrigins["*"]; ok {
		return true
	}
	_, ok := m.allowedOrigins[origin]
	return ok
}

func isPublicPath(path string) bool {
	switch path {
	case "/healthz", "/readyz":
		return true
	default:
		return false
	}
}
