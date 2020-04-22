package srserver

// Server routing

import (
	"fmt"
	gorillaMux "github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"srserver/config"
	"strings"
)

func RegisterDefaultGames() {
	conn := redisPool.Get()
	defer conn.Close()

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

	mux.HandleFunc("/join-game", loggedHandler(handleJoinGame)).Methods("POST")

	mux.HandleFunc("/events", loggedHandler(handleEvents)).Methods("GET")

	mux.HandleFunc("/players", loggedHandler(handleGetPlayers)).Methods("GET")

	mux.HandleFunc("/roll", loggedHandler(handleRoll)).Methods("POST")

	mux.HandleFunc("/", loggedHandler(func(response Response, request *Request) {
		if config.IsProduction {
			log.Print("-> 307 Moved Temporarily shadowroller")
			http.Redirect(response, request, config.FrontendAddress, http.StatusTemporaryRedirect)
		} else {
			io.WriteString(response, "Hello world!")
		}
	})).Methods("GET")

	mux.HandleFunc("/health-check", loggedHandler(func(response Response, request *Request) {
		io.WriteString(response, "ok")
	}))

	return mux
}
