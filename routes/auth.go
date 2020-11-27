package routes

import (
	"errors"
	"sr"
)

var authRouter = restRouter.PathPrefix("/auth").Subrouter()

type loginResponse struct {
	PlayerID   string          `json:"playerID"`
	PlayerName string          `json:"playerName"`
	GameInfo   *sr.OldGameInfo `json:"game"`
	Session    string          `json:"session"`
	LastEvent  string          `json:"lastEvent"`
}

// POST /auth/login { gameID, playerName } -> auth token, session token
var _ = authRouter.HandleFunc("/login", handleLogin).Methods("POST")

func handleLogin(response Response, request *Request) {
	logRequest(request)
	var loginRequest struct {
		GameID     string `json:"gameID"`
		PlayerName string `json:"playerName"`
		Persist    bool   `json:"persist"`
	}
	err := readBodyJSON(request, &loginRequest)
	httpBadRequestIf(response, request, err)

	playerName := loginRequest.PlayerName
	gameID := loginRequest.GameID
	persist := loginRequest.Persist
	logf(request,
		"Login request: %v joining %v (persist: %v)",
		playerName, gameID, persist,
	)

	conn := sr.RedisPool.Get()
	defer closeRedis(request, conn)

	// Check for permission to join (if game ID exists)
	gameExists, err := sr.GameExists(gameID, conn)
	httpInternalErrorIf(response, request, err)
	if !gameExists {
		httpNotFound(response, request, "Game not found")
	}

	// Create player
	session, err := sr.NewPlayerSession(gameID, playerName, persist, conn)
	httpInternalErrorIf(response, request, err)

	eventID, err := sr.AddNewPlayerToKnownGame(&session, conn)
	httpInternalErrorIf(response, request, err)

	logf(request, "Authenticated: %s", session.LogInfo())

	// Get game info
	gameInfo, err := sr.GetOldGameInfo(session.GameID, conn)
	httpInternalErrorIf(response, request, err)

	// Response
	loggedIn := loginResponse{
		PlayerID:   string(session.PlayerID),
		PlayerName: session.PlayerName,
		GameInfo:   gameInfo,
		Session:    string(session.ID),
		LastEvent:  eventID,
	}
	err = writeBodyJSON(response, loggedIn)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		session.Type(), " ", session.ID, " for ", session.PlayerID, " in ", gameID,
	)
}

func newHandleLogin(response Response, request *Request) {
	logRequest(request)
	var login struct {
		GameID   string `json:"gameID"`
		Username string `json:"user"`
		Persist  bool   `json:"persist"`
	}
	err := readBodyJSON(request, &login)
	httpBadRequestIf(response, request, err)

	status := "persist"
	if !login.Persist {
		status = "temp"
	}
	logf(request,
		"Login request: %v to join %v %v",
		login.Username, login.GameID, status,
	)

	conn := sr.RedisPool.Get()
	defer closeRedis(request, conn)

	gameInfo, playerID, err := sr.LogPlayerIn(login.GameID, login.Username, conn)
	if errors.Is(err, sr.ErrPlayerNotFound) {
		logf(request, "Login response: game %v not found", login.GameID)
		httpForbiddenIf(response, request, err)
	} else if errors.Is(err, sr.ErrGameNotFound) {
		logf(request, "Login response: player %v not found", login.Username)
		httpForbiddenIf(response, request, err)
	} else if err != nil {
		httpInternalErrorIf(response, request, err)
	}
	logf(request, "found %v in %v", playerID, login.GameID)

	session, err := sr.NewPlayerSession(login.GameID, login.Username, login.Persist, conn)
	httpInternalErrorIf(response, request, err)
	logf(request, "Created session %v for %v", session.ID, playerID)
	logf(request, "Got game info %v", gameInfo)

	err = writeBodyJSON(response, loginResponse{
		PlayerID: string(playerID),
		//GameInfo: gameInfo, // TODO update
		Session: string(session.ID),
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
		"Relogin request for %v", requestSession,
	)

	conn := sr.RedisPool.Get()
	defer closeRedis(request, conn)

	session, err := sr.GetSessionByID(requestSession, conn)
	httpUnauthorizedIf(response, request, err)

	// Session could have been compromised since last login.

	gameExists, err := sr.GameExists(session.GameID, conn)
	httpInternalErrorIf(response, request, err)
	if !gameExists {
		logf(request, "Game %v does not exist", session.GameID)
		err = sr.RemoveSession(&session, conn)
		httpInternalErrorIf(response, request, err)
		logf(request, "Removed session for deleted game %v", session.GameID)
		httpUnauthorized(response, request, "Your session is now invalid")
	}

	// We skip showing a "player has joined" message here.

	logf(request, "Confirmed %s", session.LogInfo())

	gameInfo, err := sr.GetOldGameInfo(session.GameID, conn)
	httpInternalErrorIf(response, request, err)

	reauthed := loginResponse{
		PlayerID:   string(session.PlayerID),
		PlayerName: session.PlayerName,
		GameInfo:   gameInfo,
		Session:    string(session.ID),
		LastEvent:  "",
	}
	err = writeBodyJSON(response, reauthed)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		session.PlayerID, " reauthed for ", session.GameID,
	)
}

func newHandleReauth(response Response, request *Request) {
	logRequest(request)
	var reauth struct {
		Session string `json:"session"`
	}
	err := readBodyJSON(request, &reauth)
	httpBadRequestIf(response, request, err)
	logf(request, "Relogin request for %v", reauth.Session)

	conn := sr.RedisPool.Get()
	defer closeRedis(request, conn)

	sess, err := sr.GetSessionByID(reauth.Session, conn)
	httpUnauthorizedIf(response, request, err)
	logf(request, "Found session %v for player %v", sess.ID, sess.PlayerID)

	// Confirm game still exists for the session
	gameExists, err := sr.GameExists(sess.GameID, conn)
	httpInternalErrorIf(response, request, err)
	if !gameExists {
		logf(request, "Game %v for session %v does not exist", sess.GameID, sess)
		err = sr.RemoveSession(&sess, conn)
		logf(request, "Removed session %v for deleted game %v", sess.ID, sess.GameID)
		httpUnauthorized(response, request, "Your session is now invalid")
	}
	logf(request, "Confirming authorization for %v in %v", sess.PlayerID, sess.GameID)

	gameInfo, err := sr.GetGameInfo(sess.GameID, conn)
	httpInternalErrorIf(response, request, err)
	logf(request, "Got game info %v", gameInfo)

	// TODO sr.MakeLoginResponse(*Session, *GameInfo)
	err = writeBodyJSON(response, loginResponse{
		PlayerID:   string(sess.PlayerID),
		PlayerName: sess.PlayerName,
		//GameInfo:   gameInfo,
		Session: string(sess.ID),
	})
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		sess.Type(), " ", sess.ID, " reauth for ", sess.PlayerID,
		" in ", sess.GameID,
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
	logf(request, "Logged out %v", sess.LogInfo())

	httpSuccess(response, request, "logged out")
}
