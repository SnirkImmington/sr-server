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

// POST /auth/login { gameID, playerID, playerName }
var _ = gameRouter.HandleFunc("/login", login).Methods("POST")

type loginRequest struct {
	gameID     string `json:"gameID"`
	playerName string `json:"playerName"`
}

type loginResponse struct {
	playerID      string      `json:"playerID"`
	game          sr.GameInfo `json:"game"`
	newestEventID string      `json:"newestEventID"`
	token         string      `json:"authToken"`
	sessionID     string      `json:"sessionID"`
}

func login(response Response, request *Request) {
	logRequest(request)
	var loginRequest loginRequest
	err := readBodyJSON(request, &loginRequest)
	if err != nil {
		httpInvalidRequest(response, request, "Invalid request")
	}

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	if !sr.GameExists(loginRequest.gameID, conn) {
		httpUnauthorized(response, request, errGameNotFound)
		return
	}

	playerID := sr.GenUID()
	sr.AddPlayerToGame(playerID, loginRequest.playerName)
	sessionID := sr.MakeSession(playerID, gameID, loginRequest.playerName)
	gameInfo := sr.GameInfo(gameID)

	event := sr.PlayerJoinEvent{
		EventCore:  sr.EventCore{Type: "playerJoin"},
		PlayerID:   playerID,
		PlayerName: join.PlayerName,
	}
	newestEventID, err := sr.PostEvent(join.GameID, event, conn)
	if err != nil {
		httpInternalError(response, request, err)
		return
	}

	logf(request, "%v (%v) has joined %v",
		playerID, loginRequest.playerName, loginRequest.gameID,
	)

	token, err := createAuthToken(loginRquest)
	if err != nil {
		httpInternalError(response, request, err)
	}

	loggedIn := loginResponse{
		playerID:    playerID,
		game:        gameInfo,
		lastEventID: newestEventID,
		authToken:   token,
		sessionID:   sessionID,
	}

}

type Session struct {
	ID       string `redis:"-"`
	GameID   string `redis:"gameID"`
	PlayerID string `redis:"playerID"`

	Playername string `redis:"playerName"`
	jwt.StandardClaims
}

type SessionToken struct {
	Version string `json:"sr.v"`
	jwt.StandardClaims
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
