package task

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"log"
	"sr/event"
	"sr/id"
)

const bufferSize = 1000

func streamReadEvents(gameID string, callback func([]event.Event, int), conn redis.Conn) error {
	count := 1
	newestID := fmt.Sprintf("%v", id.NewEventID())
	for {
		log.Printf("> %v read events older than %v", count, newestID)
		events, err := event.GetOlderThan(gameID, newestID, bufferSize, conn)
		if err != nil {
		}
		log.Printf("> %v found %v / %v events", count, len(events), bufferSize)
		foundEvents := make([]event.Event, len(events))
		for i, eventText := range events {
			evt, err := event.Parse([]byte(eventText))
			if err != nil {
				log.Printf("> ! Error parsing event #%v '%v': %v",
					i, eventText, err,
				)
			}
		}

	}
}
