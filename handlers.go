package srserver

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"log"
	"net/http"
	"runtime/debug"
	"srserver/config"
	"time"
)

type Handler = http.Handler
type Request = http.Request
type Response = http.ResponseWriter

type HandlerFunc = func(Response, *Request)

func recoveryMiddleware(next Handler) Handler {
	return http.HandlerFunc(func(response Response, request *Request) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("Panic serving", request.Method, request.URL, "to", request.Host)
				log.Println(message)
				log.Println(string(debug.Stack()))
				http.Error(response, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(response, request)
	})
}

func loggedMiddleware(next Handler) http.HandlerFunc {
	return http.HandlerFunc(func(response Response, request *Request) {
		if config.IsProduction {
			log.Println(request.Proto, request.Method, request.RequestURI)
		} else {
			log.Println(request.Method, request.URL)
		}
		next.ServeHTTP(response, request)
	})
}

func hstsMiddleware(next Handler) http.HandlerFunc {
	return http.HandlerFunc(func(response Response, request *Request) {
		response.Header().Add(
			"Strict-Transport-Security",
			"max-age=63072000; preload",
		)
		next.ServeHTTP(response, request)
	})
}

// Taken from https://redis.io/commands/incr#pattern-rate-limiter-1
func rateLimitedMiddleware(next Handler) http.HandlerFunc {
	return http.HandlerFunc(func(response Response, request *Request) {
		conn := redisPool.Get()
		defer conn.Close()

		// Rate limit on a per-minute basis
		ts := time.Now().Unix() % 60
		rateLimitKey := "ratelimit:" + request.RemoteAddr + ":" + string(ts)

		current, err := redis.Int(conn.Do("get", rateLimitKey))
		if err != nil {
			httpInternalError(response, request, err)
			return // Don't serve the page if our rate limit isn't working.
		}

		if current > config.MaxRequstsPerMinute {
			log.Print("Rate limit for", request.RemoteAddr, "hit")
			http.Error(response, "Rate limited", 400)
			return
		} else if current == 0 {
			conn.Send("multi")
			conn.Send("incr", rateLimitKey)
			conn.Send("expire", rateLimitKey, 60)
			err := conn.Send("exec")
			if err != nil {
				httpInternalError(response, request, err)
				return
			}
			// read incr
			// read expire
		} else {
			conn.Do("incr", rateLimitKey)
		}

		next.ServeHTTP(response, request)
	})
}
