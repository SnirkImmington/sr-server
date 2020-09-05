package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/janberktold/sse"
	"log"
	"net/http"
	"sr/config"
	"time"
)

var sseUpgrader = sse.Upgrader{
	RetryTime: time.Duration(config.SSEClientRetrySecs) * time.Second,
}

// Request is an alias for http.Request
type Request = http.Request

// Response is an alias for http.ResponseWriter
type Response = http.ResponseWriter

var errExtraBody = errors.New("encountered additional data after end of JSON body")

func requestRemoteAddr(request *Request) string {
	if config.ClientIPHeader != "" {
		res := request.Header.Get(config.ClientIPHeader)
		if res != "" {
			return res
		}
	}
	return request.RemoteAddr
}

func cacheIndefinitely(request *Request, response Response) {
	rawLog(1, request, "Caching for 4 hours")
	response.Header().Set("Cache-Control", "max-age=14400")
}

func readBodyJSON(request *Request, value interface{}) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(value)
	if err != nil {
		return err
	}
	if decoder.More() {
		return errExtraBody
	}
	return nil
}

func writeBodyJSON(response Response, value interface{}) error {
	response.Header().Set("Content-Type", "text/json")
	return json.NewEncoder(response).Encode(value)
}

func logRequest(request *Request, values ...string) {
	if config.IsProduction {
		rawLog(1, request, fmt.Sprintf(
			"<< %v %v %v %v",
			requestRemoteAddr(request), request.Proto, request.Method, request.URL,
		))
	} else {
		rawLog(1, request, fmt.Sprintf("<< %v %v",
			request.Method, request.URL,
		))
	}
}

func logf(request *Request, format string, values ...interface{}) {
	rawLog(1, request, format, values...)
}

func rawLog(stack int, request *Request, format string, values ...interface{}) {
	id := requestID(request.Context())
	message := fmt.Sprintf(format, values...)
	err := log.Output(2+stack, id+" "+message)
	if err != nil {
		log.Print(id, " [Output Error] ", message)
	}
}

func httpSuccess(response Response, request *Request, message ...interface{}) {
	if len(message) == 0 {
		message = []interface{}{"OK"}
	}
	dur := displayRequestDuration(request.Context())
	rawLog(1, request, fmt.Sprintf(">> 200 %v (%v)", fmt.Sprint(message...), dur))
}
