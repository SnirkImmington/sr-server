package event

import (
	"sr"
)

// EventTypeRoll is the type of `RollEvent`s.
const EventTypeRoll = "roll"

// Roll is triggered when a player rolls non-edge dice.
type Roll struct {
	EventCore
	Title   string `json:"title"`
	Dice    []int  `json:"dice"`
	Glitchy int    `json:"glitchy"`
}

// RollEventCore makes the EventCore of a RollEvent.
func RollEventCore(player *sr.Player) EventCore {
	return MakeEventCore(EventTypeRoll, player)
}

//
// Edge Roll
//

// EventTypeEdgeRoll is the type of `EdgeRollEvent`s.
const EventTypeEdgeRoll = "edgeRoll"

// EdgeRoll is triggered when a player uses edge before a roll.
type EdgeRoll struct {
	EventCore
	Title   string  `json:"title"`
	Rounds  [][]int `json:"rounds"`
	Glitchy int     `json:"glitchy"`
}

// EdgeRollEventCore makes the EventCore of an EdgeRollEvent.
func EdgeRollEventCore(player *sr.Player) EventCore {
	return MakeEventCore("edgeRoll", player)
}

//
// Reroll Failures
//

// EventTypeRerollFailures is the type of `RerollFailuresEvent`.
const EventTypeRerollFailures = "rerollFailures"

// RerollFailures is triggered when a player uses edge for Second Chance
// on a roll.
type RerollFailures struct {
	EventCore
	PrevID  int64   `json:"prevID"`
	Title   string  `json:"title"`
	Rounds  [][]int `json:"rounds"`
	Glitchy int     `json:"glitchy"`
}

// RerollFailuresEventCore makes the EventCore of a RerollFailuresEvent.
func RerollFailuresEventCore(player *sr.Player) EventCore {
	return MakeEventCore(EventTypeRerollFailures, player)
}
