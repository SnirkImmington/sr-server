package routes

import (
	"github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
)

// AuthToken is a representation of sr.Auth in JWT token form.
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
		logf(request, "Request has no 'Authentication' header")
		return "", ErrNoAuthHeader
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		logf(request, "Invalid authentication header: '%v'", auth)
		return "", ErrInvalidAuthHeader
	}
	return auth[8:], nil
}

func requestSession(request *Request, redis *redis.Redis) (*Session, error) {
	sessionID, err := requestSessionID(request)
	if err != nil {
		return nil, err
	}

	result, err := redis.Do("HGET", "session:"+sessionID)
	if err != nil {
		logf(request, "Could not find session %v", sessionID)
		return nil, err
	}
	var session *Session
	err = redis.ScanStruct(result, session)
	if err != nil {
		logf(request, "Error paring session struct: %v", err)
		return nil, err
	}

	return session, nil
}
