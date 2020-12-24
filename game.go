package sr

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"sr/config"
)

// ErrGameNotFound means that a specified game does not exists
var ErrGameNotFound = errors.New("game not found")

// ErrTransactionAborted means that a transaction was aborted and should be retried
var ErrTransactionAborted = errors.New("transaction aborted")

// GameExists returns whether the given game exists in Redis.
func GameExists(gameID string, conn redis.Conn) (bool, error) {
	return redis.Bool(conn.Do("exists", "game:"+gameID))
}

// GetPlayersInGame retrieves the list of players in a game.
// Returns ErrGameNotFound if the game is not found OR if it has no players.
func GetPlayersInGame(gameID string, conn redis.Conn) ([]Player, error) {
	getPlayerMaps := func() ([]string, []interface{}, error) {
		if _, err := conn.Do("WATCH", "players:"+gameID); err != nil {
			return nil, nil, fmt.Errorf("redis error sending `WATCH`: %w", err)
		}
		playerIDs, err := redis.Strings(conn.Do("SMEMBERS", "players:"+gameID))
		if err != nil {
			return nil, nil, fmt.Errorf("redis error retrieving player ID list: %w")
		}
		if playerIDs == nil || len(playerIDs) == 0 {
			if _, err := conn.Do("UNWATCH", "players:"+gameID); err != nil {
				return nil, nil, fmt.Errorf("redis error sending `UNWATCH`: %w", err)
			}
			return nil, nil, fmt.Errorf("game %v has no players: %w", gameID, ErrGameNotFound)
		}

		if err = conn.Send("MULTI"); err != nil {
			return nil, nil, fmt.Errorf("redis error sending MULTI: %w", err)
		}
		for _, playerID := range playerIDs {
			if err = conn.Send("HGETALL", "player:"+playerID); err != nil {
				return nil, nil, fmt.Errorf("redis error sending HGETALL %v: %w", playerID, err)
			}
		}

		playerMaps, err := redis.Values(conn.Do("EXEC"))
		if err != nil {
			return nil, nil, fmt.Errorf("redis error sending EXEC: %w", err)
		}
		if playerMaps == nil || len(playerMaps) == 0 {
			return nil, nil, ErrTransactionAborted
		}
		if len(playerMaps) < len(playerIDs) {
			return nil, nil, fmt.Errorf(
				"insufficient list of players in %v for meaningful response: %v",
				gameID, playerMaps,
			)
		}
		return playerIDs, playerMaps, nil
	}
	var err error
	var playerIDs []string
	var playerMaps []interface{}
	for i := 0; i < config.RedisRetries; i++ {
		playerIDs, playerMaps, err = getPlayerMaps()
		if errors.Is(err, ErrTransactionAborted) {
			continue
		} else if errors.Is(err, ErrGameNotFound) {
			return nil, err
		} else if err != nil {
			return nil, fmt.Errorf("After %s attempt(s): %w", i+1, err)
		}
		break
	}
	if err != nil {
		return nil, fmt.Errorf("Error after max attempts: %w", err)
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

// AddPlayerToGame adds the given player to the given game
func AddPlayerToGame(player *Player, gameID string, conn redis.Conn) error {
	updateBytes, err := json.Marshal(MakePlayerAddUpdate(player))
	if err != nil {
		return fmt.Errorf("error creating add player update for %v: %w", player, err)
	}

	if err := conn.Send("MULTI"); err != nil {
		return fmt.Errorf("redis error sending MULTI: %w", err)
	}
	if err := conn.Send("SADD", "players:"+gameID, player.ID); err != nil {
		return fmt.Errorf("redis error sending SADD: %w", err)
	}
	if err := conn.Send("PUBLISH", "update:"+gameID, updateBytes); err != nil {
		return fmt.Errorf("redis error sending PUBLISH: %w", err)
	}
	// EXEC: [#added=1, #updated]
	results, err := redis.Ints(conn.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("redis error sending EXEC: %w", err)
	}
	if len(results) != 2 || results[0] != 1 {
		return fmt.Errorf(
			"redis invalid adding %v to %v: expected [1, *], got %v",
			player, gameID, results,
		)
	}
	return nil
}
