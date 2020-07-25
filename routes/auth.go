package routes

import (
	"errors"
	//"io"
	"sr"
	// "strings"
)

var authRouter = restRouter.PathPrefix("/auth").Subrouter()

type loginResponse struct {
	PlayerID     string      `json:"playerID"`
	GameInfo     sr.GameInfo `json:"game"`
	AuthToken    string      `json:"authToken"`
	SessionID string      `json:"session"`
	LastEvent    string      `json:"lastEvent"`
}

// POST /auth/login { gameID, playerName } -> auth token, session token
var _ = authRouter.HandleFunc("/login", handleLogin).Methods("POST")

func handleLogin(response Response, request *Request) {
	logRequest(request)
	// var builder strings.Builder
	//written, err := io.Copy(&builder, request.Body)
	// logf(request, "Written: %v, err: %v", written, err)
	// logf(request, "Body: %s", builder.String())
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
		gameID, playerName, persist,
	)

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	// Check for permission to join (if game ID exists)
	gameExists, err := sr.GameExists(gameID, conn)
	httpInternalErrorIf(response, request, err)
	if !gameExists {
		httpNotFound(response, request, "Game not found")
	}

	// Create player
	auth, session, err := sr.CreateAuthedPlayer(gameID, playerName, conn)
	httpInternalErrorIf(response, request, err)

	eventID, err := sr.AddNewPlayerToKnownGame(&auth, conn)
	httpInternalErrorIf(response, request, err)

	logf(request, "Granted %s", auth.LogInfo())

	// Get game info
	gameInfo, err := sr.GetGameInfo(auth.GameID, conn)
	httpInternalErrorIf(response, request, err)

	var authToken string
	if persist {
		authToken, err = makeAuthToken(&auth)
		httpInternalErrorIf(response, request, err)
	}

	// Response
	loggedIn := loginResponse{
		PlayerID:     string(auth.PlayerID),
		GameInfo:     gameInfo,
		AuthToken:    authToken,
		SessionID: string(session.ID),
		LastEvent:    eventID,
	}
	err = writeBodyJSON(response, loggedIn)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		auth.PlayerID, " joined ", gameID,
	)
}

type reauthResponse struct {
	SessionID string `json:"session"`
}

// POST auth/reauth { authToken } -> session token
var _ = authRouter.HandleFunc("/reauth", handleReauth).Methods("POST")

func handleReauth(response Response, request *Request) {
	logRequest(request)
	var authRequest struct {
		token string `json:"token"`
	}
	err := readBodyJSON(request, &authRequest)
	httpInternalErrorIf(response, request, err)

	// Check the auth
	auth, err := parseAuthToken(authRequest.token)
	httpBadRequestIf(response, request, err)

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	authValid, err := sr.CheckAuth(&auth, conn)
	if !authValid {
		if errors.Is(err, sr.ErrOldAuthVersion) {
			httpUnauthorized(response, request, "You have been logged out")
		} else {
			httpInternalErrorIf(response, request, err)
		}
	}

	// Make the session
	session, err := sr.MakeSession(auth.GameID, auth.PlayerName, auth.PlayerID, conn)
	httpInternalErrorIf(response, request, err)

	reauthedResponse := reauthResponse{
		SessionID: string(session.ID),
	}
	err = writeBodyJSON(response, reauthedResponse)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		"reauth ", auth.PlayerName, " to ", auth.GameID,
	)
}

// POST auth/logout { session } -> OK

var _ = authRouter.HandleFunc("/logout", handleLogout).Methods("POST")

func handleLogout(response Response, request *Request) {
	logRequest(request)
	sess, conn, err := requestSession(request)
	httpUnauthorizedIf(response, request, err)
	defer sr.CloseRedis(conn)

	ok, err := sr.RemoveSession(&sess, conn)
	httpInternalErrorIf(response, request, err)
	if !ok {
		logf(request, "Possibly-timed-out session %v", sess)
	} else {
	    logf(request, "Logged out %v", sess)
    }

	httpSuccess(response, request, "logged out")
}
