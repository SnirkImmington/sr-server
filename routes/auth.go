package routes

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	"github.com/gomodule/redigo/redis"
	"sr"
	"sr/config"
	"strings"
)

var authRouter = router.PathPrefix("/auth").Subrouter()

type loginResponse struct {
	playerID      string      `json:"playerID"`
	game          sr.GameInfo `json:"game"`
	newestEventID string      `json:"newestEventID"`
	token         string      `json:"authToken"`
	sessionToken  string      `json:"sessionToken"`
}

// POST /auth/login { gameID, playerID, playerName }
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

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	// Check for permission to join (if game ID exists)
	if !sr.GameExists(loginRequest.gameID, conn) {
		httpUnauthorized(response, request, errGameNotFound)
		return
	}

	// Create player
	auth := sr.AddPlayerToGame(loginRequest.playerName, conn)

	// Post event
	event := sr.PlayerJoinEvent{
		EventCore:  sr.EventCore{Type: "playerJoin"},
		PlayerID:   auth.playerID,
		PlayerName: auth.PlayerName,
	}
	newestEventID, err := sr.PostEvent(join.GameID, event, conn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	logf(request, "%v (%v) has joined %v",
		playerID, loginRequest.playerName, loginRequest.gameID,
	)

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

	loggedIn := loginResponse{
		playerID:     playerID,
		game:         gameInfo,
		lastEventID:  newestEventID,
		authToken:    token,
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
	auth, err := sr.AuthFromToken(authRequest.token)
	if err != nil && httpInvalidRequest(response, request, err) {
		return
	}

}

type AuthToken struct {
	GameID     string `json:"sr.gid"`
	PlayerID   string `json:"sr.pid"`
	PlayerName string `json:"sr.pname"`
	jwt.StandardClaims
}

func (auth *AuthToken) String() string {
	return fmt.Sprintf("%v (%v) in %v",
		auth.PlayerID, auth.PlayerName, auth.GameID,
	)
}

func createAuthToken(request *loginRequest) (string, error) {
	expireTime := time.Now()
	claims := AuthToken{
		GameID:         request.gameID,
		PlayerID:       request.playerID,
		PlayerName:     request.playerName,
		StandardClaims: jwt.StandardClaims{},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.JWTSecretKey)
}

func createSessionToken(session *Session, conn *redis.Redis) (string, error) {
	version, err := sr.SessionVersion(conn)
	if err != nil {
		return nil, err
	}
	expireTime := time.Now()
	claims := SessionToken{
		Version: version,
		StandardClaims: jwt.StandardClaims{
			ID:        session.ID,
			ExpiresAt: expireTime,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.JWTSecretKey)
}

var ErrNoAuthHeader = errors.New("No Authentication header")
var ErrInvalidAuthHeader = errors.New("Invalid Authentication header")

type tokenExtractor struct{}

func (t *tokenExtractor) ExtractToken(request *Request) (string, error) {
	auth := request.Header.Get("Authentication")
	if auth == "" {
		return "", ErrNoAuthHeader
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", ErrInvalidAuthHeader
	}

	logf(request, "Have header %v, got auth %v", auth, auth[8:])

	return auth[8:], nil
}

func requestAuthToken(request *Request) (*AuthToken, error) {
	token, err := jwtRequest.ParseFromRequest(
		request,
		&tokenExtractor{},
		sr.GetJWTSecretKey,
		jwtRequest.WithClaims(&AuthToken{}),
	)
	if err != nil {
		return nil, err
	}
	auth, ok := token.Claims.(*AuthToken)
	if !ok || !token.Valid {
		return nil, jwt.ErrInvalidKeyType
	}
	return auth, nil
}

func requestSessionID(request *Request) (string, error) {
	auth := request.Header.Get("Authentication")
	if auth == "" {
		return "", ErrNoAuthHeader
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", ErrInvalidAuthHeader
	}

	logf(request, "Have header %v, got auth %v", auth, auth[8:])

	return auth[8:], nil
}

func requestSession(request *Request, redis *redis.Redis) (*Session, error) {
	sessionID, err := requestSessionID(request)
	if err != nil {
		return nil, err
	}

	var session *Session
	result, err := redis.Do("HGET", "session:"+sessionID)
	if err != nil {
		return nil, err
	}
	err = redis.ScanStruct(result, session)
	if err != nil {
		logf(request, "Error paring struct: %v", err)
		return nil, err
	}

	return session, nil
}
