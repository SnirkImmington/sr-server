package routes

import (
	"errors"
	"sr"
)

var authRouter = router.PathPrefix("/auth").Subrouter()

type loginResponse struct {
	playerID     string      `json:"playerID"`
	gameInfo     sr.GameInfo `json:"game"`
	authToken    string      `json:"authToken"`
	sessionToken string      `json:"sessionToken"`
}

// POST /auth/login { gameID, playerName } -> auth token, session token
var _ = gameRouter.HandleFunc("/login", handleLogin).Methods("POST")

func handleLogin(response Response, request *Request) {
	logRequest(request)
	var loginRequest struct {
		gameID     string `json:"gameID"`
		playerName string `json:"playerName"`
	}
	err := readBodyJSON(request, &loginRequest)
	httpBadRequestIf(response, request, err)

	playerName := loginRequest.playerName
	gameID := loginRequest.gameID

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	// Check for permission to join (if game ID exists)
	gameExists, err := sr.GameExists(loginRequest.gameID, conn)
	httpInternalErrorIf(response, request, err)
	if !gameExists {
		httpNotFound(response, request, "Game not found")
	}

	// Create player
	auth, session, err := sr.AuthenticatePlayer(playerName)
	httpInternalErrorIf(response, request, err)

	eventID, err := sr.AddNewPlayerToKnownGame(
		auth.PlayerID, playerName, gameID, conn,
	)
	httpInternalErrorIf(response, request, err)

	logf(request, "Granted %s", auth)

	// Get game info
	gameInfo, err := sr.GetGameInfo(auth.GameID, conn)
	httpInternalErrorIf(response, request, err)

	// Create response
	sessionToken, err := makeSessionToken(session)
	httpInternalErrorIf(response, request, err)
	authToken, err := makeAuthToken(auth, conn)
	httpInternalErrorIf(response, request, err)

	// Response
	loggedIn := loginResponse{
		playerID:     auth.PlayerID,
		gameInfo:     gameInfo,
		authToken:    authToken,
		sessionToken: sessionToken,
	}
	err = writeBodyJSON(response, loggedIn)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		auth.PlayerID, " joined ", gameID,
	)
}

type reauthResponse struct {
	sessionToken string `json:"sessionToken"`
}

// POST auth/reauth { authToken } -> session token
var _ = authRouter.HandleFunc("/reauth", reauth).Methods("POST")

func reauth(response Response, request *Request) {
	logRequest(request)
	var authRequest struct {
		token string `json:"token"`
	}
	err := readBodyJSON(request, &authRequest)
	httpInternalErrorIf(response, request, err)

	// Check the auth
	auth, err := authFromToken(authRequest.token)
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
	session, err := sr.MakeSession(auth.PlayerID, auth.GameID, auth.PlayerName, conn)
	httpInternalErrorIf(response, request, err)
	sessionToken, err := makeSessionToken(session)
	httpInternalErrorIf(response, request, err)

	reauthedResponse := reauthResponse{
		sessionToken: sessionToken,
	}
	err = writeBodyJSON(response, reauthedResponse)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		"reauth ", auth.PlayerName, " to ", auth.GameID,
	)
}
