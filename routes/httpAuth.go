package routes

import (
	"github.com/dgrijalva/jwt-go"
)

// AuthToken is a representation of sr.Auth in JWT token form.
type AuthToken struct {
	GameID     string `json:"sr.gid"`
	PlayerID   string `json:"sr.pid"`
	PlayerName string `json:"sr.pname"`
	Version    int    `json:"sr.v"`
	jwt.StandardClaims
}

func (auth *AuthToken) String() string {
	return fmt.Sprintf("%v (%v) in %v",
		auth.PlayerID, auth.PlayerName, auth.GameID,
	)
}
