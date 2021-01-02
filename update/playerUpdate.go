package update

import (
	"encoding/json"
	"sr"
)

// Player is an update for players changing
type Player interface {
	Update

	PlayerID() UID
}

type playerDiff struct {
	id   UID
	diff map[string]interface{}
}

func (update *playerDiff) Type() string {
	return UpdateTypePlayer
}

func (update *playerDiff) PlayerID() UID {
	return update.id
}

func (update *playerDiff) MarshalJSON() ([]byte, error) {
	fields := []interface{}{
		UpdateTypePlayer, update.id, update.diff,
	}
	return json.Marshal(fields)
}

// ForPlayerDiff constructs an update for a player's info changing
func ForPlayerDiff(playerID UID, diff map[string]interface{}) Player {
	return &playerDiff{
		id:   playerID,
		diff: diff,
	}
}

type playerAdd struct {
	info PlayerInfo
}

func (update *playerAdd) Type() string {
	return UpdateTypePlayer
}

func (update *playerAdd) PlayerID() UID {
	return update.Info.ID
}

func (update *playerAdd) MarshalJSON() ([]byte, error) {
	fields := []interface{}{UpdateTypePlayer, "add", update.Info}
	return json.Marshal(fields)
}

// ForPlayerAdd constructs an update for adding a player to a game
func ForPlayerAdd(info sr.PlayerInfo) Player {
	return &playerAdd{info}
}
