package routes

// Server routing

import (
	"fmt"
	gorillaMux "github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"sr/config"
	"strings"
)

var router = gorillaMux.NewRouter()

func RegisterDefaultGames() {
	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	gameNames := strings.Split(config.HardcodedGameNames, ",")

	for _, game := range gameNames {
		_, err := conn.Do("hmset", "game:"+game, "event_id", 0)
		if err != nil {
			panic(fmt.Sprintf("Unable to connect to redis: ", err))
		}
	}

	log.Print("Registered ", len(gameNames), " hardcoded game IDs.")
}

func MakeServerMux() *gorillaMux.Router {
	mux := gorillaMux.NewRouter()
	mux.Use(recoveryMiddleware)
	mux.Use(requestIDMiddleware)
	mux.Use(headersMiddleware)
	mux.Use(rateLimitedMiddleware)

	mux.HandleFunc("/join-game", handleJoinGame).Methods("POST")

	mux.HandleFunc("/events", handleEvents).Methods("GET")

	mux.HandleFunc("/event-range", handleEventRange).Methods("POST")

	mux.HandleFunc("/players", handleGetPlayers).Methods("GET")

	mux.HandleFunc("/roll", handleRoll).Methods("POST")

	mux.HandleFunc("/", func(response Response, request *Request) {
		logRequest(request)
		if config.IsProduction {
			log.Printf("%v -> snirkimmington.github.io/shadowroller", http.StatusSeeOther)
			http.Redirect(
				response, request,
				"https://snirkimmington.github.io/shadowroller", http.StatusSeeOther,
			)
		} else {
			_, err := io.WriteString(response,
				`<a href="`+config.FrontendAddress+`">Frontend</a>`)
			if err != nil {
				httpInternalError(response, request, err)
				return
			}
			httpSuccess(response, request, "page with link to frontend")
		}
	}).Methods("GET")

	mux.HandleFunc("/health-check", func(response Response, request *Request) {
		logRequest(request)
		httpSuccess(response, request)
	}).Methods("GET")

	return mux
}