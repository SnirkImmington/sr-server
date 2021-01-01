package sr

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
)

// ErrNotAuthorized is an error for when a user cannot perform an action
var ErrNotAuthorized = errors.New("not authorized")

// LogPlayerIn checks username/gameID credentials and returns the relevant
// GameInfo for the client.
//
// Returns ErrPlayerNotFound if the username is not found, ErrGameNotFound if
// the game is not found. These should not be distinguished to users.
func LogPlayerIn(gameID string, username string, conn redis.Conn) (*GameInfo, *Player, error) {
	player, err := GetPlayerByUsername(username, conn)
	if errors.Is(err, ErrPlayerNotFound) {
		return nil, nil, fmt.Errorf("%w (%v logging into %v)", err, username, gameID)
	} else if err != nil {
		return nil, nil, fmt.Errorf("redis error getting %v: %w", username, err)
	}

	info, err := GetGameInfo(gameID, conn)
	if errors.Is(err, ErrGameNotFound) {
		return nil, nil, fmt.Errorf("when logging %v in to %v: %w", username, gameID, err)
	} else if err != nil {
		return nil, nil, fmt.Errorf("redis error fetching game info for %v: %w", gameID, err)
	}

	// Ensure player is in the game
	if _, found := info.Players[string(player.ID)]; !found {
		return nil, nil, fmt.Errorf(
			"%w: player %v (%v) to %v",
			ErrNotAuthorized, player.ID, username, gameID,
		)
	}
	return info, player, nil
}
