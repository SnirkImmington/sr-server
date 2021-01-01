package sr

import (
	"encoding/json"
	"fmt"
	"regexp"
)

//
// Roll
//

// EventTypeRoll is the type of `RollEvent`s.
const EventTypeRoll = "roll"

// RollEvent is triggered when a player rolls non-edge dice.
type RollEvent struct {
	EventCore
	Title   string `json:"title"`
	Dice    []int  `json:"dice"`
	Glitchy int    `json:"glitchy"`
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
	Title   string  `json:"title"`
	Rounds  [][]int `json:"rounds"`
	Glitchy int     `json:"glitchy"`
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
	PrevID  int64   `json:"prevID"`
	Title   string  `json:"title"`
	Rounds  [][]int `json:"rounds"`
	Glitchy int     `json:"glitchy"`
}

// RerollFailuresEventCore makes the EventCore of a RerollFailuresEvent.
func RerollFailuresEventCore(session *Session) EventCore {
	return MakeEventCore(EventTypeRerollFailures, session)
}

// EventTypeInitiativeRoll is the type of `InitiativeRollEvent`.
const EventTypeInitiativeRoll = "initiativeRoll"

// InitiativeRollEvent is an event for a player's initiative roll.
type InitiativeRollEvent struct {
	EventCore
	Title string `json:"title"`
	Base  int    `json:"base"`
	Dice  []int  `json:"dice"`
}

// InitiativeRollEventCore makes the EventCore of an InitiativeRollEvent.
func InitiativeRollEventCore(session *Session) EventCore {
	return MakeEventCore(EventTypeInitiativeRoll, session)
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
	ID         int64  `json:"id"`             // ID of the event
	Type       string `json:"ty"`             // Type of the event
	Edit       int64  `json:"edit,omitempty"` // Edit time of the event
	PlayerID   UID    `json:"pID"`            // ID of the player who posted the event
	PlayerName string `json:"pName"`          // Name of the player who posted the event
}

// Event is the common interface of all events.
type Event interface {
	GetID() int64
	GetType() string
	GetPlayerID() UID
	GetPlayerName() string
	GetEdit() int64
	SetEdit(edited int64)
}

// GetID returns the timestamp ID of the event.
func (core *EventCore) GetID() int64 {
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

// GetEdit gets the event's edit time
func (core *EventCore) GetEdit() int64 {
	return core.Edit
}

// SetEdit updates the event's edit time
func (core *EventCore) SetEdit(edited int64) {
	core.Edit = edited
}

// ParseEvent parses an event from JSON
func ParseEvent(input []byte) (Event, error) {
	var data map[string]interface{}
	err := json.Unmarshal(input, &data)
	if err != nil {
		return nil, fmt.Errorf("could not parse event object: %w", err)
	}
	tyVal, ok := data["ty"]
	if !ok {
		return nil, fmt.Errorf("parsed input did not contain a ty field")
	}
	ty, ok := tyVal.(string)
	if !ok {
		return nil, fmt.Errorf("error retrieving type info for event: got %v", data)
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

	case EventTypeInitiativeRoll:
		var initiativeRoll InitiativeRollEvent
		err = json.Unmarshal(input, &initiativeRoll)
		return &initiativeRoll, err

	case EventTypePlayerJoin:
		var playerJoin PlayerJoinEvent
		err = json.Unmarshal(input, &playerJoin)
		return &playerJoin, err

	default:
		return nil, fmt.Errorf("unknown event type %v", ty)
	}
}

// MakeEventCore produces an EventCore of the given type using the given session.
func MakeEventCore(ty string, session *Session) EventCore {
	return EventCore{
		ID:         NewEventID(),
		Type:       ty,
		Edit:       0,
		PlayerID:   session.PlayerID,
		PlayerName: "", // TODO
	}
}

// Hacky workaround for logs to show event type.
// A user couldn't actually write "ty":"foo" in the field, though,
// as it'd come back escaped.
var eventTyParse = regexp.MustCompile(`"ty":"([^"]+)"`)
var eventIDParse = regexp.MustCompile(`"id":(\d+)`)

// ParseEventTy gives the `ty` field for an event string.
// This should only be used for logging.
func ParseEventTy(event string) string {
	match := eventTyParse.FindStringSubmatch(event)
	if len(match) != 2 {
		return "??"
	}
	return match[1]
}

// ParseEventID gives the `id` field for an event ID as a string.
// This should only be used for logging.
func ParseEventID(event string) string {
	match := eventIDParse.FindStringSubmatch(event)
	if len(match) != 2 {
		return "????????"
	}
	return match[1]
}
