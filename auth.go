package sr

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
)

// LogPlayerIn checks username/gameID credentials and returns the relevant
// GameInfo for the client.
//
// Returns ErrPlayerNotFound if the username is not found, ErrGameNotFound if
// the game is not found. These should not be distinguished to users.
func LogPlayerIn(username string, gameID string, conn redis.Conn) (*GameInfo, *Player, error) {
	player, err := GetPlayerByUsername(username, conn)
	if errors.Is(err, ErrPlayerNotFound) {
		return nil, nil, fmt.Errorf("when logging %v in to %v: %w", username, gameID, err)
	} else if err != nil {
		return nil, nil, fmt.Errorf("redis error getting player: %w", err)
	}

	info, err := GetGameInfo(gameID, conn)
	if errors.Is(err, ErrGameNotFound) {
		return nil, nil, fmt.Errorf("when logging %v in to %v: %w", username, gameID, err)
	} else if err != nil {
		return nil, nil, fmt.Errorf("redis error getting game info: %w", err)
	}

	// Ensure player is in the game
	if _, found := info.Players[string(player.ID)]; !found {
		return nil, nil, fmt.Errorf(
			"could not find %v (%v) in %v",
			player.ID, username, gameID, info,
		)
	}
	return info, player, nil
}
