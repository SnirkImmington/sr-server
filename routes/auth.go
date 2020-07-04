package routes

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"sr"
	"sr/config"
	"strings"
)

var authRouter = router.PathPrefix("/auth").Subrouter()

type loginResponse struct {
	playerID     string      `json:"playerID"`
	gameInfo     sr.GameInfo `json:"game"`
	authToken    string      `json:"authToken"`
	sessionToken string      `json:"sessionToken"`
}

// POST /auth/login { gameID, playerName } -> auth token, session token
var _ = gameRouter.HandleFunc("/login", login).Methods("POST")

func login(response Response, request *Request) {
	logRequest(request)
	var loginRequest struct {
		gameID     string `json:"gameID"`
		playerName string `json:"playerName"`
	}
	err := readBodyJSON(request, &loginRequest)
	if err != nil && httpInvalidRequest(response, request, err) {
		return
	}
	playerName := loginRequest.playerName
	gameID := loginRequest.gameID

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	// Check for permission to join (if game ID exists)
	if !sr.GameExists(loginRequest.gameID, conn) {
		httpUnauthorized(response, request, errGameNotFound)
		return
	}

	// Create player
	auth, session := sr.AuthenticatePlayer(playerName)
	eventID := sr.AddNewPlayerToGame(auth.PlayerID, playerName, gameID, conn)
	logf(request, "%v (%v) has joined %v", playerID, playerName, gameID)

	if err != nil && httpInternalError(response, request, err) {
		return
	}

	// Create session and auth
	session, err := sr.MakeSession(playerID, gameID, playerName, conn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	// Create response
	sessionToken, err := createSessionToken(sessionID, conn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	authToken, err := createAuthToken(auth, conn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}

	// Response
	loggedIn := loginResponse{
		playerID:     playerID,
		gameInfo:     gameInfo,
		authToken:    authToken,
		sessionToken: sessionToken,
	}
	err = writeBodyJSON(response, loggedIn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	httpSuccess(response, request)
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
	if err != nil && httpInternalError(response, request, err) {
		return
	}

	// Check the auth
	auth, err := authFromToken(authRequest.token)
	if err != nil && httpInvalidRequest(response, request, "Invalid auth") {
		return
	}

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	authValid, err := sr.CheckAuth(auth, conn)
	if !authValid {
		if errors.Is(err, sr.AuthVersionError) {
			httpUnauthorized(response, request, err)
		} else {
			httpInternalError(response, request, err)
		}
		return
	}

	// Make the session
	session, err := sr.MakeSession(auth.playerID, auth.gameID, auth.playerName, conn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	sessionToken, err := makeSessionToken(session)
	if err != nil && httpInternalError(response, request, err) {
		return
	}

	reauthedResponse := reauthResponse{
		sessionToken: sessionToken,
	}
	err = writeBodyJSON(response, reauthedResponse)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	httpSuccess(response, request, "reauth ", auth.playerName, " to ", auth.gameID)
}
