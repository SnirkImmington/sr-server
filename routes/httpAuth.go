package routes

import (
	"github.com/dgrijalva/jwt-go"
	"sr"
	"sr/config"
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

func makeAuthToken(auth *sr.Auth) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		AuthToken{
			GameID:   auth.GameID,
			PlayerID: auth.PlayerID,
			Version:  auth.Version,

			PlayerName: auth.PlayerName,
		},
	)
	return token.SignedString(config.JWTSecretKey)
}

func authFromToken(token string) (sr.Auth, error) {

}
