package sr

import (
	"encoding/json"
	"regexp"
)

// UpdateTypeEvent is the "type" field that's set for event updates
const UpdateTypeEvent = "evt"

// UpdateTypePlayer is the "type" field that's set for player updates
const UpdateTypePlayer = "plr"

// EventRenameUpdate is posted when an event is renamed by a player.
func EventRenameUpdate(event Event, newTitle string) EventUpdate {
	update := MakeEventDiffUpdate(event)
	update.AddField("title", newTitle)
	return &update
}

// SecondChanceUpdate is posted when a roll is rerolled
func SecondChanceUpdate(event Event, round []int) EventUpdate {
	update := MakeEventDiffUpdate(event)
	update.AddField("reroll", round)
	return &update
}

// EventDeleteUpdate is posted when an event is deleted by a player.
func EventDeleteUpdate(eventID int64) EventUpdate {
	return &EventDelUpdate{ID: eventID}
}

// EventUpdate is the interface for updates to events
type EventUpdate interface {
	json.Marshaler

	GetEventID() int64
	GetTime() int64
}

// EventDiffUpdate updates various fields on an event.
type EventDiffUpdate struct {
	ID   int64
	Time int64
	Diff map[string]interface{}
}

func MakeEventDiffUpdate(event Event) EventDiffUpdate {
	return EventDiffUpdate{
		ID:   event.GetID(),
		Time: event.GetEdit(),
		Diff: make(map[string]interface{}),
	}
}

func (update *EventDiffUpdate) GetEventID() int64 {
	return update.ID
}

func (update *EventDiffUpdate) GetTime() int64 {
	return update.Time
}

func (update *EventDiffUpdate) AddField(field string, value interface{}) {
	update.Diff[field] = value
}

// MarshalJSON converts the update to JSON. They're formatted as a 3-element list.
func (update *EventDiffUpdate) MarshalJSON() ([]byte, error) {
	fields := []interface{}{UpdateTypeEvent, update.ID, update.Diff, update.Time}
	return json.Marshal(fields)
}

// EventDelUpdate is a specific update type for deleting events
type EventDelUpdate struct {
	ID int64
}

func (update *EventDelUpdate) GetEventID() int64 {
	return update.ID
}

func (update *EventDelUpdate) GetTime() int64 {
	return 0
}

func (update *EventDelUpdate) MarshalJSON() ([]byte, error) {
	fields := []interface{}{UpdateTypeEvent, update.ID, "del"}
	return json.Marshal(fields)
}

var updateTyParse = regexp.MustCompile(`$\["([^"]+)`)

func ParseUpdateTy(update string) string {
	match := updateTyParse.FindStringSubmatch(update)
	if len(match) != 2 {
		return "??"
	}
	return match[1]
}

// PlayerDiffUpdate is an update when an attribute of a Player has been changed
type PlayerDiffUpdate struct {
	ID   UID
	Diff map[string]interface{}
}

// MakePlayerDiffUpdate produces a PlayerDiffUpdate for the given Player
func MakePlayerDiffUpdate(playerID UID) PlayerDiffUpdate {
	return PlayerDiffUpdate{
		ID:   playerID,
		Diff: make(map[string]interface{}),
	}
}

// AddField adds a field to the given player update
func (update *PlayerDiffUpdate) AddField(field string, value interface{}) {
	update.Diff[field] = value
}

// MarshalJSON - [plr ID Diff{}]
func (update *PlayerDiffUpdate) MarshalJSON() ([]byte, error) {
	fields := []interface{}{UpdateTypePlayer, update.ID, update.Diff}
	return json.Marshal(fields)
}

// PlayerAddUpdate is sent to clients when a player is added to a game
type PlayerAddUpdate struct {
	Info PlayerInfo
}

// MakePlayerAddUpdate constructs a PlayerAddUpdate
func MakePlayerAddUpdate(player *Player) PlayerAddUpdate {
	return PlayerAddUpdate{player.Info()}
}

// MarshalJSON - [plr add PlayerInfo{}]
func (update *PlayerAddUpdate) MarshalJSON() ([]byte, error) {
	fields := []interface{}{UpdateTypePlayer, "add", update.Info}
	return json.Marshal(fields)
}
