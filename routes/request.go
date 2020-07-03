// Request middleware(s) for all requests.
package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sr/config"
	"time"
)

type Request = http.Request
type Response = http.ResponseWriter

type HandlerFunc = func(Response, *Request)

func readBodyJSON(request *Request, value interface{}) error {
	return json.NewDecoder(request.Body).Decode(value)
}

func writeBodyJSON(response Response, value interface{}) error {
	response.Header().Set("Content-Type", "text/json")
	return json.NewEncoder(response).Encode(value)
}

func logRequest(request *Request, values ...string) {
	if config.IsProduction {
		logf(request, "<- %v %v %v %v",
			request.RemoteAddr, request.Proto, request.Method, request.URL,
		)
	} else {
		logf(request, "<- %v %v",
			request.Method, request.URL,
		)
	}
}

func logf(request *Request, format string, values ...interface{}) {
	id := requestID(request)
	err := log.Output(2, fmt.Sprintf(id+" "+format, values...))
	if err != nil {
		log.Printf(id+" [Output Error] "+format, values...)
	}
}

func httpNotFound(response Response, request *Request) bool {
	logf(request, "-> 404 not found")
	http.Error(response, "Not found", http.StatusNotFound)
	return true
}

func httpUnauthorized(response Response, request *Request, err error) bool {
	logf(request, "-> 401 Unauthorized: %v", err)
	http.Error(response, "Unauthorized", http.StatusUnauthorized)
	return true
}

func httpInternalError(response Response, request *Request, err error) bool {
	if config.IsProduction {
		logf(request, "Internal error handling %v %v %v %v: %v",
			request.RemoteAddr, request.Proto, request.Method, request.URL, err,
		)
		logf(request, "-> 500 Internal Error")
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
	} else {
		message := fmt.Sprintf("Internal error handling %v %v: %v",
			request.Method, request.URL, err,
		)
		logf(request, message)
		logf(request, " -> 500 Internal error: %v", err)
		http.Error(response, message, http.StatusInternalServerError)
	}
	return true
}

func httpInternalErrorMessage(response Response, request *Request, message interface{}) bool {
	if config.IsProduction {
		logf(request, "Internal error handling %v %v %v %v: %v",
			request.RemoteAddr, request.Proto, request.Method, request.URL, message,
		)
		logf(request, "-> 500 Internal Server Error")
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
	} else {
		logMessage := fmt.Sprintf("Internal error handling %v %v %v: %v",
			request.Method, request.URL, message,
		)
		logf(request, "-> 500 %v", logMessage)
		http.Error(response, logMessage, http.StatusInternalServerError)
	}
	return true
}

func httpInvalidRequest(response Response, request *Request, message string) {
	logf(request, "-> 400 Bad Request: ", message)
	http.Error(response, message, http.StatusBadRequest)
	return true
}

func httpSuccess(response Response, request *Request, message ...interface{}) {
	if len(message) == 0 {
		message = []interface{}{"OK"}
	}
	logf(request, "-> 200 %v", fmt.Sprint(message...))
}
