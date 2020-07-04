package routes

import (
	"net/http"
	"sr"
)

var abortedRequestPanicMessage = string(sr.GenUID())

func abortRequest() {
	panic(abortedRequestPanicMessage)
}

func httpBadRequestIf(response Response, request *Request, err error) {
	if err != nil {
		logf(request, "-> 400 Bad Request: %v", err)
		http.Error(response, "Bad Request", http.StatusBadRequest)
		abortRequest()
	}
}

func httpBadRequest(response Response, request *Request, message string) {
	if message == "" {
		message = "Bad Request"
	}
	logf(request, "-> 400 %s", message)
	http.Error(response, message, http.StatusBadRequest)
	abortRequest()
}

func httpForbiddenIf(response Response, request *Request, err error) {
	if err != nil {
		logf(request, "-> 403 Forbidden: %v", err)
		http.Error(response, "Forbidden", http.StatusForbidden)
		abortRequest()
	}
}

func httpForbidden(response Response, request *Request, message string) {
	if message == "" {
		message = "Forbidden"
	}
	logf(request, "-> 403 %s", message)
	http.Error(response, message, http.StatusForbidden)
	abortRequest()
}

func httpNotFoundIf(response Response, request *Request, err error) {
	if err != nil {
		logf(request, "-> 404 Not Found: %v", err)
		http.Error(response, "Not Found", http.StatusNotFound)
		abortRequest()
	}
}

func httpNotFound(response Response, request *Request, message string) {
	if message == "" {
		message = "Not Found"
	}
	logf(request, "-> 404 %s", message)
	http.Error(response, message, http.StatusNotFound)
	abortRequest()
}

func httpUnauthorizedIf(response Response, request *Request, err error) {
	if err != nil {
		logf(request, "-> 401 Unauthorized: %v", err)
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		abortRequest()
	}
}

func httpUnauthorized(response Response, request *Request, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	logf(request, "-> 401 %s", message)
	http.Error(response, message, http.StatusUnauthorized)
	abortRequest()
}

func httpInternalErrorIf(response Response, request *Request, err error) {
	if err != nil {
		logf(request, "-> 500 Internal Server Error: %v", err)
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		abortRequest()
	}
}

func httpInternalError(response Response, request *Request, message string) {
	if message == "" {
		message = "Internal Server Error"
	}
	logf(request, "-> 500 %s", message)
	http.Error(response, message, http.StatusInternalServerError)
	abortRequest()
}
