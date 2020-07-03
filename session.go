package sr

import ()

type Session struct {
	ID       string `redis:"-"`
	GameID   string `redis:"gameID"`
	PlayerID string `redis:"playerID"`

	Playername string `redis:"playerName"`
}

type SessionToken struct {
	SessionID string `json:"sr.sid"`
	Version   string `json:"sr.v"`
	jwt.StandardClaims
}
