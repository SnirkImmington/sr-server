package sr

// Authentication Credentials

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gomodule/redigo/redis"
	"log"
	"sr/config"
)

// Auth represents a user's credentials of a player in a game.
// Auth additionally has a `version` tag which allows us to unauthorize
// everyone.
// We trust AuthTokens from clients only because they're signed by us.
type Auth struct {
	GameID   string `json:"gameID"`
	PlayerID UID    `json:"playerID"`
	Version  int    `json:"v"`

	// Player name is tied to auth concept for now.
	PlayerName string `json:"playerName"`
}

func (auth *Auth) String() string {
	return fmt.Sprintf("%v (%v) in %v",
		auth.PlayerID, auth.PlayerName, auth.GameID,
	)
}

// CreateAuthedPlayer constructs
func CreateAuthedPlayer(gameID string, playerName string, conn redis.Conn) (Auth, Session, error) {
	auth, err := createAuth(gameID, playerName, conn)
	if err != nil {
		return Auth{}, Session{}, err
	}
	sess, err := MakeSession(gameID, playerName, auth.PlayerID, conn)
	if err != nil {
		return Auth{}, Session{}, err
	}
	return auth, sess, nil
}

func GetJWTSecretKey(token *jwt.Token) (interface{}, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, jwt.ErrInvalidKeyType
	}
	return config.JWTSecretKey, nil
}

func AuthVersion(conn redis.Conn) (int, error) {
	return redis.Int(conn.Do("get", "auth_version"))
}

func createAuth(gameID string, playerName string, conn redis.Conn) (Auth, error) {
	version, err := AuthVersion(conn)
	if err != nil {
		log.Printf("Unable to get auth version from redis: %v", err)
		return Auth{}, err
	}

	playerID := GenUID()

	return Auth{
		GameID:     gameID,
		PlayerID:   playerID,
		Version:    version,
		PlayerName: playerName,
	}, nil
}

var ErrOldAuthVersion = errors.New("auth has been revoked")

func CheckAuth(auth *Auth, conn redis.Conn) (bool, error) {
	version, err := AuthVersion(conn)
	if err != nil {
		return false, err
	}
	if version != auth.Version {
		return false, ErrOldAuthVersion
	}
	return true, nil
}

func IncrAuthVersion(conn redis.Conn) (int, error) {
	return redis.Int(conn.Do("incr", "auth_version"))
}
