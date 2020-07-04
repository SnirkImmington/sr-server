package routes

import (
	"context"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"log"
	"math/rand"
	"net/http"
	"runtime/debug"
	"sr"
	"sr/config"
	"time"
)

func headersMiddleware(wrapped http.Handler) http.Handler {
	return http.HandlerFunc(func(response Response, request *Request) {
		if config.TlsEnable {
			response.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		response.Header().Set("Cache-Control", "no-cache")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		wrapped.ServeHTTP(response, request)
	})
}

type requestIDKeyType int

var requestIDKey requestIDKeyType

func withRequestID(ctx context.Context) context.Context {
	id := fmt.Sprintf("%02x", rand.Intn(257))
	return context.WithValue(ctx, requestIDKey, id)
}

func requestID(request *Request) string {
	return request.Context().Value(requestIDKey).(string)
}

func requestIDMiddleware(wrapped http.Handler) http.Handler {
	return http.HandlerFunc(func(response Response, request *Request) {
		requestCtx := request.Context()
		requestCtx = withRequestID(requestCtx)
		requestWithID := request.WithContext(requestCtx)

		wrapped.ServeHTTP(response, requestWithID)
	})
}

func recoveryMiddleware(wrapped http.Handler) http.Handler {
	return http.HandlerFunc(func(response Response, request *Request) {
		defer func() {
			if err := recover(); err != nil {
				if err == abortedRequestPanicMessage {
					logf(request, "aborted request")
					return
				}
				message := fmt.Sprintf("Panic serving %s %s to %s", request.Method, request.RequestURI, request.Host)
				log.Println(message)
				log.Println(err)
				log.Println(string(debug.Stack()))
				http.Error(response, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		wrapped.ServeHTTP(response, request)
	})
}

func rateLimitedMiddleware(wrapped http.Handler) http.Handler {
	return http.HandlerFunc(func(response Response, request *Request) {
		conn := sr.RedisPool.Get()
		defer sr.CloseRedis(conn)

		// Taken from https://redis.io/commands/incr#pattern-rate-limiter-1

		remoteAddr := strings.Split(request.RemoteAddr, ":")[0]

		_, _, sec := time.Now().Clock()
		rateLimitKey := fmt.Sprintf("ratelimit:%v:%v", remoteAddr, sec%10)

		current, err := redis.Int(conn.Do("get", rateLimitKey))
		if err == redis.ErrNil {
			current = 0
		} else if err != nil {
			httpInternalError(response, request, err)
			return
		}

		limited := false
		if current > config.MaxRequestsPer10Secs {
			log.Printf("Rate limit for %v hit", request.RemoteAddr)
			http.Error(response, "Rate limited", http.StatusTooManyRequests)
			limited = true
		}

		err = conn.Send("multi")
		if err != nil {
			httpInternalError(response, request, err)
			return
		}
		err = conn.Send("incr", rateLimitKey)
		if err != nil {
			httpInternalError(response, request, err)
			return
		}
		// Always push back expire time: forces client to back off for 10s
		err = conn.Send("expire", rateLimitKey, "10")
		if err != nil {
			httpInternalError(response, request, err)
			return
		}
		err = conn.Send("exec")
		if err != nil {
			httpInternalError(response, request, err)
			return
		}

		if !limited {
			wrapped.ServeHTTP(response, request)
		}
	})
}
