package srserver

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"srserver/config"
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
