package sr

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"regexp"
)

// EventRenameUpdate is posted when an event is renamed by a player.
func EventRenameUpdate(eventID int64, newTitle string) Update {
	return Update{
		Type:  "event.title",
		Key:   string(eventID),
		Value: newTitle,
	}
}

// EventDeleteUpdate is posted when an event is deleted by a player.
func EventDeleteUpdate(eventID int64) Update {
	return Update{
		Type:  "event.delete",
		Key:   string(eventID),
		Value: nil,
	}
}

// PlayerRenameUpdate is posted when a player is renamed
func PlayerRenameUpdate(playerID UID, newName string) Update {
	return Update{
		Type:  "player.name",
		Key:   string(playerID),
		Value: newName,
	}
}

// Update is the type of updates sent over SSE to update players of the state of the game.
// Any change in state should be sent over an update, along with a corresponding event if
// needed.
type Update struct {
	Type  string
	Key   string
	Value interface{}
}

// MarshalJSON converts the update to JSON. They're formatted as a 3-element list.
func (update *Update) MarshalJSON() ([]byte, error) {
	fields := []interface{}{update.Type, update.Key, update.Value}
	return json.Marshal(fields)
}

// UnmarshalJSON parses an update from JSON.
func (update *Update) UnmarshalJSON(input []byte) error {
	var fields []interface{}
	err := json.Unmarshal(input, fields)
	if err != nil {
		return err
	}
	if len(fields) != 3 {
		return fmt.Errorf("Expected [Ty, Key, Val], got %v", fields)
	}
	ty, ok := fields[0].(string)
	if !ok {
		return fmt.Errorf("Expected type string, got %v", fields[0])
	}
	key, ok := fields[1].(string)
	if !ok {
		return fmt.Errorf("Expected key string, got %v", fields[1])
	}
	update.Type = ty
	update.Key = key
	update.Value = fields[3]
	return nil
}

// PostUpdate posts an update to the given game.
func PostUpdate(gameID string, update *Update, conn redis.Conn) error {
	bytes, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("unable to marshal update to JSON: %w", err)
	}

	_, err = conn.Do("PUBLISH", "update:"+gameID, bytes)
	if err != nil {
		return fmt.Errorf("unable to post update to Redis: %w", err)
	}
	return nil
}

var updateTyParse = regexp.MustCompile(`$\["([^"]+)`)

func ParseUpdateTy(update string) string {
	match := updateTyParse.FindStringSubmatch(update)
	if len(match) != 2 {
		return "??"
	}
	return match[1]
}
