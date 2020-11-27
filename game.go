package sr

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
)

// ErrGameNotFound means that a specified game does not exists
var ErrGameNotFound = errors.New("game not found")

// GameExists returns whether the given game exists in Redis.
func GameExists(gameID string, conn redis.Conn) (bool, error) {
	return redis.Bool(conn.Do("exists", "game:"+gameID))
}

// GetPlayersInGame retrieves the list of players in a game
// Returns ErrGameNotFound if the game is not found.
func GetPlayersInGame(gameID string, conn redis.Conn) ([]Player, error) {
	playerIDs, err := redis.Strings(conn.Do("SMEMBERS", "players:"+gameID))
	if err != nil {
		return nil, fmt.Errorf("redis error retrieving player ID list: %w")
	}
	if len(playerIDs) == 0 {
		return nil, fmt.Errorf("game %v has no players: %w", gameID, ErrGameNotFound)
	}
	if err = conn.Send("MULTI"); err != nil {
		return nil, fmt.Errorf("redis error sending MULTI: %w", err)
	}
	for _, playerID := range playerIDs {
		if err = conn.Send("HGETALL", "player:"+playerID); err != nil {
			return nil, fmt.Errorf("redis error sending HGETALL %v: %w", playerID, err)
		}
	}
	playerMaps, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("redis error sending EXEC: %w", err)
	}
	if len(playerMaps) == 0 || len(playerMaps) < len(playerIDs) {
		return nil, fmt.Errorf(
			"insufficient list of players in %v for meaningful response: %v",
			gameID, playerMaps,
		)
	}
	players := make([]Player, len(playerMaps))
	for i, playerMap := range playerMaps {
		err = redis.ScanStruct(playerMap.([]interface{}), &players[i])
		if err != nil {
			return nil, fmt.Errorf(
				"redis error parsing %v player #%v %v: %w",
				gameID, i, playerIDs[i], err,
			)
		}
		if players[i].Username == "" {
			return nil, fmt.Errorf(
				"no data for %v player #%v %v after redis parse: %v",
				gameID, i, playerIDs[i], playerMap,
			)
		}
	}
	return players, nil
}

// GameInfo represents basic info about a game that the frontend would want
// by default, all at once.
type GameInfo struct {
	ID      string                `json:"id"`
	Players map[string]PlayerInfo `json:"players"`
}

type OldGameInfo struct {
	ID      string            `json:"id"`
	Players map[string]string `json:"players"`
}

func GetOldGameInfo(gameID string, conn redis.Conn) (*OldGameInfo, error) {
	players, err := redis.StringMap(conn.Do("hgetall", "player:"+gameID))
	if err != nil {
		return nil, err
	}
	return &OldGameInfo{gameID, players}, nil
}

// GetGameInfo retrieves `GameInfo` for the given GameID
func GetGameInfo(gameID string, conn redis.Conn) (*GameInfo, error) {
	players, err := GetPlayersInGame(gameID, conn)
	if err != nil {
		return nil, fmt.Errorf("error getting players in game: %w", err)
	}
	info := make(map[string]PlayerInfo, len(players))
	for _, player := range players {
		info[player.ID.String()] = player.Info()
	}
	return &GameInfo{ID: gameID, Players: info}, nil
}

// AddNewPlayerToKnownGame is used at login to add a newly created player to
// a game. It does not verify the GameID.
func AddNewPlayerToKnownGame(
	session *Session,
	conn redis.Conn,
) (string, error) {
	_, err := conn.Do("hset", "player:"+session.GameID, session.PlayerID, session.PlayerName)
	if err != nil {
		return "", err
	}

	event := PlayerJoinEvent{
		EventCore: EventCore{
			ID:         NewEventID(),
			Type:       EventTypePlayerJoin,
			PlayerID:   session.PlayerID,
			PlayerName: session.PlayerName,
		},
	}
	err = PostEvent(session.GameID, &event, conn)
	if err != nil {
		return "", err
	}
	return "", nil
}
