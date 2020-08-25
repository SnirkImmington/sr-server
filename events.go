package sr

import (
	"encoding/json"
	"fmt"
)

//
// Roll
//

// EventTypeRoll is the type of `RollEvent`s.
const EventTypeRoll = "roll"

// RollEvent is triggered when a player rolls non-edge dice.
type RollEvent struct {
	EventCore
	Title string `json:"title"`
	Roll  []int  `json:"roll"`
}

// RollEventCore makes the EventCore of a RollEvent.
func RollEventCore(session *Session) EventCore {
	return MakeEventCore(EventTypeRoll, session)
}

//
// Edge Roll
//

// EventTypeEdgeRoll is the type of `EdgeRollEvent`s.
const EventTypeEdgeRoll = "edgeRoll"

// EdgeRollEvent is triggered when a player uses edge before a roll.
type EdgeRollEvent struct {
	EventCore
	Title  string  `json:"title"`
	Rounds [][]int `json:"rounds"`
}

// EdgeRollEventCore makes the EventCore of an EdgeRollEvent.
func EdgeRollEventCore(session *Session) EventCore {
	return MakeEventCore("edgeRoll", session)
}

//
// Reroll Failures
//

// EventTypeRerollFailures is the type of `RerollFailuresEvent`.
const EventTypeRerollFailures = "rerollFailures"

// RerollFailuresEvent is triggered when a player uses edge for Second Chance
// on a roll.
type RerollFailuresEvent struct {
	EventCore
	PrevID float64 `json:"prevID"`
	Title  string  `json:"title"`
	Rounds [][]int `json:"rounds"`
}

// RerollFailuresEventCore makes the EventCore of a RerollFailuresEvent.
func RerollFailuresEventCore(session *Session) EventCore {
	return MakeEventCore(EventTypeRerollFailures, session)
}

//
// Player Join
//

// EventTypePlayerJoin is the type of `PlayerJoinEvent`.
const EventTypePlayerJoin = "playerJoin"

// PlayerJoinEvent is triggered when a new player joins a game.
type PlayerJoinEvent struct {
	EventCore
}

// PlayerJoinEventCore makes the EventCore of a PlayerJoinEvent.
func PlayerJoinEventCore(session *Session) EventCore {
	return MakeEventCore(EventTypePlayerJoin, session)
}

//
// Event Definition
//

// EventCore is the basic values put into events.
type EventCore struct {
	ID         float64 `json:"id"`    // ID of the event
	Type       string  `json:"ty"`    // Type of the event
	PlayerID   UID     `json:"pID"`   // ID of the player who posted the event
	PlayerName string  `json:"pName"` // Name of the player who posted the event
}

// Event is the common interface of all events.
type Event interface {
	GetID() float64
	GetType() string
	GetPlayerID() UID
	GetPlayerName() string
}

// GetID returns the timestamp ID of the event.
func (core *EventCore) GetID() float64 {
	return core.ID
}

// GetType returns the type of the event.
func (core *EventCore) GetType() string {
	return core.Type
}

// GetPlayerID returns the PlayerID of the player who triggered the event.
func (core *EventCore) GetPlayerID() UID {
	return core.PlayerID
}

// GetPlayerName returns the name of the player who triggered the event
// at the time that it happened.
func (core *EventCore) GetPlayerName() string {
	return core.PlayerName
}

// ParseEvent parses an event from JSON
func ParseEvent(input []byte) (Event, error) {
	var data map[string]interface{}
	err := json.Unmarshal(input, &data)
	if err != nil {
		return nil, fmt.Errorf("Could not parse event object: %w", err)
	}
	tyVal, ok := data["ty"]
	if !ok {
		return nil, fmt.Errorf("Parsed input did not contain a ty field")
	}
	ty, ok := tyVal.(string)
	if !ok {
		return nil, fmt.Errorf("Error retrieving type info for event: got %v", data)
	}

	switch ty {
	case EventTypeRoll:
		var roll RollEvent
		err = json.Unmarshal(input, &roll)
		return &roll, err

	case EventTypeEdgeRoll:
		var edgeRoll EdgeRollEvent
		err = json.Unmarshal(input, &edgeRoll)
		return &edgeRoll, err

	case EventTypeRerollFailures:
		var rerollFailures RerollFailuresEvent
		err = json.Unmarshal(input, &rerollFailures)
		return &rerollFailures, err

	case EventTypePlayerJoin:
		var playerJoin PlayerJoinEvent
		err = json.Unmarshal(input, &playerJoin)
		return &playerJoin, err
	default:
		return nil, fmt.Errorf("Unknown roll type %v", ty)
	}
}

// MakeEventCore produces an EventCore of the given type using the given session.
func MakeEventCore(ty string, session *Session) EventCore {
	return EventCore{
		ID:         NewEventID(),
		Type:       ty,
		PlayerID:   session.PlayerID,
		PlayerName: session.PlayerName,
	}
}
