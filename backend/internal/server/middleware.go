package server

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type apiRateLimiter struct {
	mutex  sync.Mutex
	visits map[string]rateLimitVisit
}

type rateLimitVisit struct {
	windowStart time.Time
	count       int
}

func newAPIRateLimiter() *apiRateLimiter {
	return &apiRateLimiter{
		visits: make(map[string]rateLimitVisit),
	}
}

func (limiter *apiRateLimiter) middleware(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if !limiter.allow(request) {
			responseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
			responseWriter.WriteHeader(http.StatusTooManyRequests)
			_, _ = responseWriter.Write([]byte(`{"error":"rate limit exceeded"}` + "\n"))
			return
		}
		nextHandler.ServeHTTP(responseWriter, request)
	})
}

func (limiter *apiRateLimiter) allow(request *http.Request) bool {
	const maxRequestsPerMinute = 120
	clientIP := clientAddress(request)
	now := time.Now()

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	visit := limiter.visits[clientIP]
	if now.Sub(visit.windowStart) >= time.Minute {
		limiter.visits[clientIP] = rateLimitVisit{windowStart: now, count: 1}
		return true
	}
	visit.count++
	limiter.visits[clientIP] = visit
	return visit.count <= maxRequestsPerMinute
}

func apiAccessLog(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		nextHandler.ServeHTTP(responseWriter, request)
		log.Printf("api %s %s from %s in %s", request.Method, request.URL.Path, clientAddress(request), time.Since(startedAt).Truncate(time.Millisecond))
	})
}

func clientAddress(request *http.Request) string {
	forwardedFor := request.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		firstAddress := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		if firstAddress != "" {
			return firstAddress
		}
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return request.RemoteAddr
}
