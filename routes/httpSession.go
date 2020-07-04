package routes

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	"github.com/gomodule/redigo/redis"
	"sr"
	"sr/config"
	"strings"
)

// SessionToken is a JWT sent to clients containing their Session ID.
type SessionToken struct {
	sessionID string `json:"sid"`
	jwt.StandardClaims
}

func (sess *SessionToken) String() string {
	return fmt.Sprintf("%s", sess.sessionID)
}

func createSessionToken(session *sr.Session) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		SessionToken{sessionID: string(session.ID)},
	)
	return token.SignedString(config.JWTSecretKey)
}

// requestSession retrieves the authenticated session for the request.
// It does not open redis if an invalid session is supplied.
func requestSession(request *Request) (*Session, redis.Conn, error) {
	token, err := requestSessionToken(request)
	if err != nil {
		return nil, err
	}
	conn := sr.RedisPool.Get()
	session, err := sr.GetSessionByID(token.sessionID, conn)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return session, conn, nil
}

// tokenExtractor gets a JWT which is given in an Authentication: Bearer header.
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
func requestSessionToken(request *Request) (SessionToken, error) {
	return jwtRequest.ParseFromRequest(
		request,
		&tokenExtractor{},
		sr.GetJWTSecretKey,
		jwtRequest.WithClaims(&SessionToken{}),
	)
}
