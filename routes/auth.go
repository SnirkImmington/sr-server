package routes

import (
	"errors"
	"sr/game"
	"sr/player"
)

var authRouter = restRouter.PathPrefix("/auth").Subrouter()

type loginResponse struct {
	Player   *sr.Player   `json:"player"`
	GameInfo *sr.GameInfo `json:"game"`
	Session  string       `json:"session"`
}

// POST /auth/login { gameID, playerName } -> auth token, session token
var _ = authRouter.HandleFunc("/login", handleLogin).Methods("POST")

func handleLogin(response Response, request *Request) {
	logRequest(request)
	var login struct {
		GameID   string `json:"gameID"`
		Username string `json:"username"`
		Persist  bool   `json:"persist"`
	}
	err := readBodyJSON(request, &login)
	httpBadRequestIf(response, request, err)

	status := "persist"
	if !login.Persist {
		status = "temp"
	}
	logf(request,
		"Login request: %v to join %v (%v)",
		login.Username, login.GameID, status,
	)

	conn := sr.RedisPool.Get()
	defer closeRedis(request, conn)

	gameInfo, player, err := sr.LogPlayerIn(login.GameID, login.Username, conn)
	if err != nil {
		logf(request, "Login response: %v", err)
	}
	if errors.Is(err, sr.ErrPlayerNotFound) ||
		errors.Is(err, sr.ErrGameNotFound) ||
		errors.Is(err, sr.ErrNotAuthorized) {
		httpForbiddenIf(response, request, err)
	} else if err != nil {
		logf(request, "Error with redis operation: %v", err)
		httpInternalErrorIf(response, request, err)
	}
	logf(request, "Found %v in %v", player.ID, login.GameID)

	logf(request, "Creating session %s for %v", status, player.ID)
	session, err := sr.NewPlayerSession(login.GameID, player, login.Persist, conn)
	httpInternalErrorIf(response, request, err)
	logf(request, "Created session %v for %v", session.ID, player.ID)
	logf(request, "Got game info %v", gameInfo)

	err = writeBodyJSON(response, loginResponse{
		Player:   player,
		GameInfo: gameInfo,
		Session:  string(session.ID),
	})
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		session.Type(), " ", session.ID, " for ", session.PlayerID,
		" in ", login.GameID,
	)
}

// POST /auth/reauth { session } -> { login response }
var _ = authRouter.HandleFunc("/reauth", handleReauth).Methods("POST")

func handleReauth(response Response, request *Request) {
	logRequest(request)
	var reauthRequest struct {
		Session string `json:"session"`
	}
	err := readBodyJSON(request, &reauthRequest)
	httpBadRequestIf(response, request, err)

	requestSession := reauthRequest.Session

	logf(request,
		"Relogin request for session %v", requestSession,
	)

	conn := sr.RedisPool.Get()
	defer closeRedis(request, conn)

	session, err := sr.GetSessionByID(requestSession, conn)
	httpUnauthorizedIf(response, request, err)
	logf(request, "Found session %s", session.String())

	// Double check that the relevant items exist.
	gameExists, err := sr.GameExists(session.GameID, conn)
	httpInternalErrorIf(response, request, err)
	if !gameExists {
		logf(request, "Game %v does not exist", session.GameID)
		err = sr.RemoveSession(&session, conn)
		httpInternalErrorIf(response, request, err)
		logf(request, "Removed session %v for deleted game %v", session.ID, session.PlayerID)
		httpUnauthorized(response, request, "Your session is now invalid")
	}
	logf(request, "Confirmed game %s exists", session.GameID)

	player, err := sr.GetPlayerByID(string(session.PlayerID), conn)
	if errors.Is(err, sr.ErrPlayerNotFound) {
		logf(request, "Player %v does not exist", session.PlayerID)
		err = sr.RemoveSession(&session, conn)
		httpInternalErrorIf(response, request, err)
		logf(request, "Removed session %v for deleted player %v", session.ID, session.PlayerID)
		httpUnauthorized(response, request, "Your session is now invalid")
	} else if err != nil {
		httpInternalErrorIf(response, request, err)
	}
	logf(request, "Confirmed player %s exists", player.ID)

	gameInfo, err := sr.GetGameInfo(session.GameID, conn)
	httpInternalErrorIf(response, request, err)

	reauthed := loginResponse{
		Player:   player,
		GameInfo: gameInfo,
		Session:  string(session.ID),
	}
	err = writeBodyJSON(response, reauthed)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		session.PlayerID, " reauthed for ", session.GameID,
	)
}

// POST auth/logout { session } -> OK

var _ = authRouter.HandleFunc("/logout", handleLogout).Methods("POST")

func handleLogout(response Response, request *Request) {
	logRequest(request)
	sess, conn, err := requestSession(request)
	defer closeRedis(request, conn)
	httpUnauthorizedIf(response, request, err)

	err = sr.RemoveSession(&sess, conn)
	httpInternalErrorIf(response, request, err)
	logf(request, "Logged out %v", sess.PlayerInfo())

	httpSuccess(response, request, "logged out")
}
