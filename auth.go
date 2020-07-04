package sr

// Authentication

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
type Auth struct {
	GameID   string `json:"gameID"`
	PlayerID string `json:"playerID"`
	Version  int    `json:"v"`

	// Player name is tied to auth concept for now.
	PlayerName string `json:"playerName"`
}

func (auth *Auth) String() string {
	return fmt.Sprintf("%v (%v) in %v",
		auth.PlayerID, auth.PlayerName, auth.GameID,
	)
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

func CreateAuth(gameID string, playerID string, playerName string, conn redis.Conn) (*Auth, error) {
	version, err := AuthVersion(conn)
	if err != nil {
		log.Printf("Unable to get auth version from redis: %v", err)
		return nil, err
	}

	return &Auth{
		GameID:     gameID,
		PlayerID:   playerID,
		Version:    version,
		PlayerName: playerName,
	}, nil
}

var AuthVersionError = errors.New("auth has been revoked")

func CheckAuth(auth *Auth, conn redis.Conn) (bool, error) {
	version, err := AuthVersion(conn)
	if err != nil {
		return false, err
	}
	if version != auth.Version {
		return false, AuthVersionError
	}
	return true, nil
}

func IncrAuthVersion(conn redis.Conn) (int, error) {
	return redis.Int(conn.Do("incr", "auth_version"))
}
