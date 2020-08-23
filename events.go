package sr

// EventTypeRoll is the type of `RollEvent`s.
const EventTypeRoll = "roll"

// RollEvent is triggered when a player rolls non-edge dice.
type RollEvent struct {
	EventCore
	Title string `json:"title"`
	Roll  []int  `json:"roll"`
}

// EventTypeEdgeRoll is the type of `EdgeRollEvent`s.
const EventTypeEdgeRoll = "edgeRoll"

// EdgeRollEvent is triggered when a player uses edge before a roll.
type EdgeRollEvent struct {
	EventCore
	Title  string  `json:"title"`
	Rounds [][]int `json:"rounds"`
}

// EventTypeRerollFailures is the type of `RerollFailuresEvent`.
const EventTypeRerollFailures = "rerollFailures"

// RerollFailuresEvent is triggered when a player uses edge for Second Chance
// on a roll.
type RerollFailuresEvent struct {
	EventCore
	Title  string  `json:"title"`
	Rounds [][]int `json:"rounds"`
}

// EventTypePlayerJoin is the type of `PlayerJoinEvent`.
const EventTypePlayerJoin = "playerJoin"

// PlayerJoinEvent is triggered when a new player joins a game.
type PlayerJoinEvent struct {
	EventCore
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
