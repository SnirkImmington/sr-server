package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sr/config"
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

func httpSuccess(response Response, request *Request, message ...interface{}) {
	if len(message) == 0 {
		message = []interface{}{"OK"}
	}
	logf(request, "-> 200 %v", fmt.Sprint(message...))
}
