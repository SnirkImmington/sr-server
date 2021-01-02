package update

import (
	"encoding/json"
	"sr"
)

// Event is the interface for updates to events
type Event interface {
	Update

	EventID() int64
	Time() int64
}

// eventDiff updates various fields on an event.
type eventDiff struct {
	id   int64
	time int64
	diff map[string]interface{}
}

// Type gets the type of the update
func (update *eventDiff) Type() string {
	return UpdateTypeEvent
}

// EventID gets the ID of the update's event
func (update *eventDiff) EventID() int64 {
	return update.id
}

// Time gets the time of the update's event
func (update *eventDiff) Time() int64 {
	return update.time
}

// MarshalJSON converts the update to JSON.
func (update *eventDiff) MarshalJSON() ([]byte, error) {
	fields := []interface{}{
		UpdateTypeEvent, update.id, update.diff, update.time,
	}
	return json.Marshal(fields)
}

func makeEventDiff(event sr.Event) eventDiff {
	return &eventDiff{
		id:   event.GetID(),
		time: event.GetEdit(),
		diff: make(map[string]interface{}),
	}
}

// ForEventDiff constructs an update for an event changing
func ForEventDiff(event sr.Event, diff map[string]interface{}) Event {
	return &eventDiff{
		id:   event.GetID(),
		time: event.GetEdit(),
		eiff: diff,
	}
}

// ForEventRename constructs an update for renaming an event
func ForEventRename(event sr.Event, newTitle string) Event {
	update := makeEventDiff(event)
	update.diff["title"] = newTitle
	return &update
}

// ForSecondChance constructs an update for a second chance roll.
func ForSecondChance(event sr.Event, round []int) Event {
	update := makeEventDiff(event)
	update.diff["reroll"] = round
	return &update
}

// eventDelete is a specific update type for deleting events
type eventDelete struct {
	id int64
}

func (update *eventDelete) Type() string {
	return UpdateTypeEvent
}

func (update *eventDelete) GetEventID() int64 {
	return update.id
}

func (update *eventDelete) GetTime() int64 {
	return 0
}

func (update *eventDelete) MarshalJSON() ([]byte, error) {
	fields := []interface{}{UpdateTypeEvent, update.id, "del"}
	return json.Marshal(fields)
}

// ForEventDelete constructs an update for deleting an event
func ForEventDelete(eventID int64) Event {
	return &EventDelUpdate{eventID}
}
