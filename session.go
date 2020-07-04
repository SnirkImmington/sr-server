package sr

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"sr/config"
)

// Session represents a temporary Auth of a user's credentials.
//
// Sessions are used in order to track authenticated requests over
// remote calls. In the REST API, sessions are kept as JWTs of session IDs.
//
// Sessions are checked by redis for each authenticated endpoint hit.
// In addition to IDs of the auth'd player and game, sessions contain
// commonly-used data, such as the player name. This data must be kept up to
// date.
//
// Sessions are expired at a configurable interval after their owners drop
// from the game subscription. Web clients are told to drop session tokens.
type Session struct {
	ID       UID    `redis:"-"`
	GameID   string `redis:"gameID"`
	PlayerID string `redis:"playerID"`

	PlayerName string `redis:"playerName"`
}

func (s *Session) LogInfo() string {
	return fmt.Sprintf(
		"%v (%v) in %v",
		s.PlayerID, s.PlayerName, s.GameID,
	)
}

func (s *Session) String() string {
	return fmt.Sprintf(
		"%v (%v) in %v (%s)",
		s.PlayerID, s.PlayerName, s.GameID, s.ID,
	)
}

func (s Session) redisKey() string {
	return "session:" + string(s.ID)
}

// MakeSession adds a session for the given player in the given game
func MakeSession(gameID, playerID, playerName string, conn redis.Conn) (*Session, error) {
	sessionID := GenUID()
	session := Session{
		ID:       sessionID,
		GameID:   gameID,
		PlayerID: playerID,

		PlayerName: playerName,
	}

	sessionArgs := redis.Args{}.AddFlat(&session)
	_, err := redis.Int(conn.Do("HMSET", session.redisKey(), sessionArgs))
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// SessionExists returns whether the session exists in Redis.
func SessionExists(sessionID string, conn redis.Conn) (bool, error) {
	return redis.Bool(conn.Do("exists", "session:"+sessionID))
}

// GetSessionByID retrieves a session from redis.
func GetSessionByID(sessionID string, conn redis.Conn) (*Session, error) {
	var session Session
	data, err := redis.Values(conn.Do("HGETALL", "session:"+sessionID))
	if err != nil {
		return nil, err
	}
	err = redis.ScanStruct(data, &session)
	if err != nil {
		return nil, err
	}
	session.ID = UID(sessionID)

	return &session, nil
}

// ExpireSession sets the session to expire in `config.SesssionExpirySecs`.
func ExpireSession(session *Session, conn redis.Conn) (bool, error) {
	return redis.Bool(conn.Do(
		"expire", session.redisKey(), config.SessionExpirySecs,
	))
}

// UnexpireSession prevents the session from exipiring.
func UnexpireSession(session *Session, conn redis.Conn) (bool, error) {
	return redis.Bool(conn.Do("persist", session.redisKey()))
}
