package event

import (
	"sr/player"
)

// EventTypeInitiativeRoll is the type of `InitiativeRollEvent`.
const EventTypeInitiativeRoll = "initiativeRoll"

// InitiativeRoll is an event for a player's initiative roll.
type InitiativeRoll struct {
	EventCore
	Title string `json:"title"`
	Base  int    `json:"base"`
	Dice  []int  `json:"dice"`
}

// InitiativeRollEventCore makes the EventCore of an InitiativeRollEvent.
func InitiativeRollEventCore(player *player.Player) EventCore {
	return MakeEventCore(EventTypeInitiativeRoll, player)
}
