package task

import (
	"fmt"
	"log"
	"reflect"
	"sr/event"
	"sr/game"
	"sr/id"
	"sr/player"
	"strings"

	"github.com/gomodule/redigo/redis"
)

func handleGameMigrationTask(gameID string, conn redis.Conn) error {
	players := make(map[id.UID]id.UID)
	playersByUsername := make(map[string]id.UID)
	selectAPlayer := func() (id.UID, string) {
		for {
			log.Printf("Select a username:")
			var nameParts []string
			if _, err := fmt.Scanln(nameParts); err != nil {
				panic(fmt.Sprintf("Error scanning from stdin: %v", err))
			}
			playerName := strings.Join(nameParts, " ")
			if playerName == "" {
				log.Printf("Empty player name detected.")
				continue
			}
			if playerID, found := playersByUsername[playerName]; found {
				return playerID, playerName
			}
			log.Printf("Could not find player %v", playerName)
		}
	}
	gamePlayers, err := game.GetPlayersIn(gameID, conn)
	gamePlayersByID := player.MapByID(gamePlayers)
	if err != nil {
		return fmt.Errorf("getting game info: %w", err)
	}
	log.Printf("Game %v:", gameID)
	for _, plr := range gamePlayers {
		log.Printf("+ %v -> %v", plr.Username, plr.ID)
		playersByUsername[plr.Username] = plr.ID
	}

	// Operate on events in batches
	err = streamReadEvents(gameID, func(batch []event.Event, iter int) error {
		log.Printf("> Round %v, %v events", iter, len(batch))
		for ix, evt := range batch {
			// Get existing data from event
			playerID := evt.GetPlayerID()
			playerName := evt.GetPlayerName()
			var newPlayerID id.UID
			// If we already have a known player, set them
			if foundID, found := players[playerID]; found {
				foundPlayer := gamePlayersByID[playerID]
				log.Printf("Found %v (%v) -> %v (%v)",
					playerName, playerID, foundPlayer.Username, foundID,
				)
				newPlayerID = foundID
			} else { // Prompt for username to use
				log.Printf("New player %v (%v) in event %v", playerName, playerID, ix)
				log.Printf(" - %v", printEvent(evt))
				foundID, foundName := selectAPlayer()
				log.Printf("New %v (%v) -> %v (%v)",
					playerName, playerID, foundName, foundID,
				)
				players[playerID] = foundID
				newPlayerID = foundID
			}
			// Update the event value
			playerIDValue := reflect.Indirect(reflect.ValueOf(evt)).FieldByName("PlayerID")
			if !playerIDValue.CanSet() {
				return fmt.Errorf("cannot set %#v of %#v", playerIDValue, evt)
			}
			playerIDValue.Set(reflect.ValueOf(newPlayerID))
		}
		err = event.BulkUpdate(gameID, batch, conn)
		if err != nil {
			return fmt.Errorf("bulk updating #%v: %w", iter, err)
		}
		return nil
	}, conn)
	if err != nil {
		return fmt.Errorf("Received error from streamRead: %w", err)
	}
	return nil
}
