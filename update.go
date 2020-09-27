package sr

import (
	"encoding/json"
	"regexp"
)

// EventRenameUpdate is posted when an event is renamed by a player.
func EventRenameUpdate(eventID int64, newTitle string) EventUpdate {
	diff := make(map[string]interface{}, 1)
	diff["title"] = newTitle
	return &EventDiffUpdate{ID: eventID, Diff: diff}
}

// EventDeleteUpdate is posted when an event is deleted by a player.
func EventDeleteUpdate(eventID int64) EventUpdate {
	return &EventDelUpdate{ID: eventID}
}

// EventUpdate is the interface for updates to events
type EventUpdate interface {
	json.Marshaler

	GetEventID() int64
}

// EventDiffUpdate updates various fields on an event.
type EventDiffUpdate struct {
	ID   int64
	Diff map[string]interface{}
}

func MakeEventDiffUpdate(event Event) EventDiffUpdate {
	return EventDiffUpdate{ID: event.GetID(), Diff: make(map[string]interface{})}
}

func (update *EventDiffUpdate) GetEventID() int64 {
	return update.ID
}

func (update *EventDiffUpdate) AddField(field string, value interface{}) {
	update.Diff[field] = value
}

// MarshalJSON converts the update to JSON. They're formatted as a 3-element list.
func (update *EventDiffUpdate) MarshalJSON() ([]byte, error) {
	fields := []interface{}{update.ID, update.Diff}
	return json.Marshal(fields)
}

// EventDelUpdate is a specific update type for deleting events
type EventDelUpdate struct {
	ID int64
}

func (update *EventDelUpdate) GetEventID() int64 {
	return update.ID
}

func (update *EventDelUpdate) MarshalJSON() ([]byte, error) {
	fields := []interface{}{update.ID, "del"}
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
