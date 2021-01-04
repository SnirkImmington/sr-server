package routes

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"net/http"
	"sr/config"
	"sr/game"
	"sr/id"
	"sr/player"
	redisUtil "sr/redis"
	"sr/session"
)

// RegisterTasksViaConfig adds the /task route if it's enabled.
// It must be called after config values are loaded.
func RegisterTasksViaConfig() {
	if config.EnableTasks {
		tasksRouter.NotFoundHandler = http.HandlerFunc(notFoundHandler)
		restRouter.PathPrefix("/task").Handler(tasksRouter)
	}
	if config.TasksLocalhostOnly {
		tasksRouter.Use(localhostOnlyMiddleware)
	}
}

var tasksRouter = apiRouter().PathPrefix("/task").Subrouter()

var _ = tasksRouter.HandleFunc("/create-game", handleCreateGame).Methods("GET")

func handleCreateGame(response Response, request *Request) {
	logRequest(request)
	gameID := request.FormValue("gameID")
	if gameID == "" {
		httpBadRequest(response, request, "Invalid game ID")
	}

	conn := redisUtil.Connect()
	defer closeRedis(request, conn)

	if exists, err := game.Exists(gameID, conn); exists {
		httpInternalErrorIf(response, request, err)
		httpBadRequest(response, request, "Game already exists")
	}

	set, err := redis.Int(conn.Do(
		"HSET", "game:"+gameID,
		"created_at", id.TimestampNow(),
	))
	httpInternalErrorIf(response, request, err)
	if set != 1 {
		httpInternalError(response, request,
			fmt.Sprintf("Expected 1 new field to be updated, got %v", set),
		)
	}
	httpSuccess(response, request,
		"Game ", gameID, " created",
	)
}

var _ = tasksRouter.HandleFunc("/delete-game", handleCreateGame).Methods("GET")

func handleDeleteGame(response Response, request *Request) {
	logRequest(request)
	gameID := request.FormValue("gameID")
	if gameID == "" {
		httpBadRequest(response, request, "Invalid game ID")
	}

	conn := redisUtil.Connect()
	defer closeRedis(request, conn)

	if exists, err := game.Exists(gameID, conn); !exists {
		httpInternalErrorIf(response, request, err)
		httpBadRequest(response, request, "Game does not exist")
	}

	set, err := redis.Int(conn.Do(
		"del", "game:"+gameID,
	))
	httpInternalErrorIf(response, request, err)
	if set != 1 {
		httpInternalError(response, request,
			fmt.Sprintf("Expected 1 game to be deleted, got %v", set),
		)
	}
	httpSuccess(response, request,
		"Game ", gameID, " deleted",
	)
}

var _ = tasksRouter.HandleFunc("/create-player", handleCreatePlayer).Methods("GET")

func handleCreatePlayer(response Response, request *Request) {
	logRequest(request)

	conn := redisUtil.Connect()
	defer closeRedis(request, conn)

	username := request.FormValue("uname")
	if username == "" {
		httpBadRequest(response, request, "Username `uname` not specified")
	}
	name := request.FormValue("name")
	if name == "" {
		httpBadRequest(response, request, "Name `name` not specified")
	}
	id := id.GenUID()
	hue := player.RandomHue()

	plr := player.Player{
		ID:       id,
		Username: username,
		Name:     name,
		Hue:      hue,
	}
	logf(request, "Created %#v", plr)

	err := player.Create(&plr, conn)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		"Player ", plr.Username, " created with ID ", plr.ID,
	)
}

var _ = tasksRouter.HandleFunc("/add-to-game", handleAddToGame).Methods("GET")

func handleAddToGame(response Response, request *Request) {
	logRequest(request)

	username := request.FormValue("uname")
	if username == "" {
		httpBadRequest(response, request, "Username `uname` not specified")
	}
	gameID := request.FormValue("game")
	if gameID == "" {
		httpBadRequest(response, request, "GameID `game` not specified")
	}

	conn := redisUtil.Connect()
	defer closeRedis(request, conn)

	plr, err := player.GetByUsername(username, conn)
	if errors.Is(err, player.ErrNotFound) {
		httpBadRequest(response, request,
			fmt.Sprintf("Player with username %v not found", username),
		)
	}
	httpInternalErrorIf(response, request, err)
	logf(request, "Found %#v", plr)

	err = game.AddPlayer(gameID, plr, conn)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		"Added ", plr, " to ", gameID,
	)
}

var _ = tasksRouter.HandleFunc("/migrate-to-players", handleMigrateToPlayers).Methods("GET")

func handleMigrateToPlayers(response Response, request *Request) {
	logRequest(request)

	gameID := request.FormValue("gameID")
	if gameID == "" {
		httpBadRequest(response, request, "Invalid game ID")
	}

	conn := redisUtil.Connect()
	defer closeRedis(request, conn)

	/*
		playerMap := make(map[sr.UID]sr.UID)
		eventCount := 0
		batch := 1

		for {

		}
	*/
}

var _ = tasksRouter.HandleFunc("/trim-players", handleTrimPlayers).Methods("GET")

func handleTrimPlayers(response Response, request *Request) {
	logRequest(request)
	_, conn, err := requestSession(request)
	defer closeRedis(request, conn)
	httpUnauthorizedIf(response, request, err)

	gameID := request.URL.Query().Get("gameID")
	if gameID == "" {
		httpBadRequest(response, request, "No game ID given")
	}
	logf(request, "Trimming players in %v", gameID)
	exists, err := game.Exists(gameID, conn)
	httpInternalErrorIf(response, request, err)
	if !exists {
		httpBadRequest(response, request, "No game '"+gameID+"' found")
	}

	sessionKeys, err := redis.Strings(conn.Do("keys", "session:*"))
	httpInternalErrorIf(response, request, err)
	logf(request, "Found %v sessions", len(sessionKeys))
	if len(sessionKeys) == 0 {
		httpInternalError(response, request, "There are no sessions")
	}
	err = conn.Send("MULTI")
	httpInternalErrorIf(response, request, err)

	// var foundPlayers map[string]bool

	for _, key := range sessionKeys {
		sessionID := key[8:]
		logf(request, "Checking for session %v in %v", sessionID, gameID)
		sess, err := session.GetByID(sessionID, conn)
		logf(request, "Found %v", sess)
		httpInternalErrorIf(response, request, err)
	}
}
