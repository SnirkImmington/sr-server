package game

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"sr/id"
	"sr/update"
)

// UpdatePlayer updates a player in the database.
// It does not allow for username updates. It only publishes the update to the given game.
func UpdatePlayer(gameID string, playerID id.UID, update update.Player, conn redis.Conn) error {
	playerData := update.MakeRedisArgs()
	updateBytes, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("unable to marshal update to JSON :%w", err)
	}

	// MULTI: update player, publish update
	if err := conn.Send("MULTI"); err != nil {
		return fmt.Errorf("redis error sending MULTI for player update: %w", err)
	}
	if err = conn.Send("HSET", playerData...); err != nil {
		return fmt.Errorf("redis error sending HSET for player update: %w", err)
	}
	if err = conn.Send("PUBLISH", "update:"+gameID, updateBytes); err != nil {
		return fmt.Errorf("redis error sending event publish: %w", err)
	}
	// EXEC: [#updated>0, #players]
	results, err := redis.Ints(conn.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("redis error sending EXEC: %w", err)
	}
	if len(results) != 2 {
		return fmt.Errorf("redis error updating player, expected 2 results got %v", results)
	}
	if results[0] <= 0 {
		return fmt.Errorf("redis error updating player, expected [1, *] got %v", results)
	}
	return nil
}
