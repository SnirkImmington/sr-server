package routes

import (
	"fmt"
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
			PlayerID: string(auth.PlayerID),
			Version:  auth.Version,

			PlayerName: auth.PlayerName,
		},
	)
	return token.SignedString(config.JWTSecretKey)
}

func parseAuthToken(tokenString string) (sr.Auth, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthToken{}, sr.GetJWTSecretKey)
	if err != nil {
		return sr.Auth{}, err
	}
	return authFromToken(token.Claims.(*AuthToken)), nil
}

func authFromToken(token *AuthToken) sr.Auth {
	return sr.Auth{
		GameID:     token.GameID,
		PlayerID:   sr.UID(token.PlayerID),
		Version:    token.Version,
		PlayerName: token.PlayerName,
	}
}
