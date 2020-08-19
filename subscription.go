package sr

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"log"
	"regexp"
)

// EventCore is the basic values put into events.
type EventCore struct {
	ID         int64  `json:"id"`    // ID of the event
	Type       string `json:"ty"`    // Type of the event
	PlayerID   string `json:"pID"`   // ID of the player who posted the event
	PlayerName string `json:"pName"` // Name of the player who posted the event
}

// Event is the common interface of all events.
type Event interface {
	GetID() string
	GetType() string
	GetPlayerID() string
	GetPlayerName() string
}

func (core *EventCore) GetID() int64 {
	return core.ID
}
func (core *EventCore) GetType() string {
	return core.Type
}
func (core *EventCore) GetPlayerID() string {
	return core.PlayerID
}
func (core *EventCore) GetPlayerName() string {
	return core.PlayerName
}

const EventTypeRoll = "roll"

type RollEvent struct {
	EventCore
	Title string `json:"title"`
	Roll  []int  `json:"roll"`
}

const EventTypeEdgeRoll = "edgeRoll"

type EdgeRollEvent struct {
	EventCore
	Title  string  `json:"title"`
	Rounds [][]int `json:"rounds"`
}

const EventTypeRerollFailures = "rerollFailures"

type RerollFailuresEvent struct {
	EventCore
	Title  string  `json:"title"`
	Rounds [][]int `json:"rounds"`
}

const EventTypePlayerJoin = "playerJoin"

type PlayerJoinEvent struct {
	EventCore
}
