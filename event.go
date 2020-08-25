package sr

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"log"
	"regexp"
	"strings"
)

// PostEvent posts an event to Redis and returns the generated ID.
func PostEvent(gameID string, event Event, conn redis.Conn) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("Unable to marshal event to JSON: %w", err)
	}

	err = conn.Send("MULTI")
	if err != nil {
		return fmt.Errorf("Redis error initiating event post: %w", err)
	}
	err = conn.Send("ZADD", "history:"+gameID, "NX", event.GetID(), bytes)
	if err != nil {
		return fmt.Errorf("Redis error sending add event to history: %w", err)
	}
	err = conn.Send("PUBLISH", "history:"+gameID, bytes)
	if err != nil {
		return fmt.Errorf("Redis error ending publish event to history: %w", err)
	}
	results, err := redis.Ints(conn.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("Redis error EXECing event post or parsing: %w", err)
	}
	if len(results) != 2 {
		return fmt.Errorf("Redis error posting event, expected 2 results, got %v", results)
	}
	if results[0] != 1 {
		log.Printf(
			"Unexpected result from posting event: expected [1, #activeplrs], got %v",
			results,
		)
	}

	return nil
}

// EventByID retrieves a single event from Redis via its ID.
func EventByID(gameID string, eventID string, conn redis.Conn) (string, error) {
	//data, err := conn.Do("XRANGE", "event:"+gameID, eventID, eventID)
	//events, err := ScanEvents(data.([]interface{}))
	events, err := redis.Strings(conn.Do(
		"ZRANGEBYSCORE",
		"history:"+gameID,
		eventID, eventID,
		"LIMIT", "0", "1",
	))

	if err != nil {
		return "", fmt.Errorf("Redis error finding event by ID: %w", err)
	}

	return events[0], nil
}

var idRegex = regexp.MustCompile(`^([\d]{13})-([\d]+)$`)

// ValidEventID returns whether the non-empty-string id is valid.
func ValidEventID(id string) bool {
	return idRegex.MatchString(id)
}

// SubscribeToGame starts a goroutine that reads from the given game's history
// and update channels.
// Each update is sent over the returned string channel, with a prefix "event:"
// for events and "update:" for updates.
// The given context is used for its cancellation function. Errors (such as being
// canceled) are sent over the error channel.
func SubscribeToGame(ctx context.Context, conn redis.Conn, gameID string) (<-chan string, <-chan error) {
	events := make(chan string)
	errChan := make(chan error, 1)

	sub := redis.PubSubConn{Conn: conn}
	if err := sub.Subscribe("history:"+gameID, "updates:"+gameID); err != nil {
		errChan <- fmt.Errorf("Unable to subscribe to update channels: %w", err)
		return events, errChan
	}

	go func() {
		defer func() {
			close(events)
			close(errChan)
		}()
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
			switch msg := sub.Receive().(type) {
			case error:
				errChan <- fmt.Errorf("Error from Redis Receive(): %w", msg)
				return
			case redis.Message:
				message := string(msg.Data)
				if strings.HasPrefix(msg.Channel, "history") {
					message = "event:" + message
				} else {
					message = "update:" + message
				}
				events <- message
			}
		}
	}()
	return events, errChan

}

// ReceiveEvents subscribes to the event stream for a given game
func ReceiveEvents(gameID string, requestID string) (<-chan string, chan<- bool) {
	eventsChan := make(chan string)
	okChan := make(chan bool)
	go func() {
		defer close(eventsChan)

		conn := RedisPool.Get()
		defer CloseRedis(conn)
		sub := redis.PubSubConn{Conn: conn}
		if err := sub.Subscribe("history:" + gameID); err != nil {
			log.Printf("%vE Error subscribing to game history: %v", err)
			return
		}

		log.Printf("%vE Begin reading events for %v", requestID, gameID)

		for {
			// See if we've been canceled.
			select {
			case <-okChan:
				log.Printf("%vE: Received cancel signal", requestID)
				log.Printf("%vE << close: signal", requestID)
				return
			default:
			}

			_, err := redis.Values(conn.Do(
				"XREAD", "BLOCK", 0, "STREAMS", "event:"+gameID, "$",
			))
			events := []string{}
			if err != nil {
				log.Printf(
					"%vE Error reading stream for %v: %v",
					requestID, gameID, err,
				)
				log.Printf("%vE << close error: %v", requestID, err)
				return
			}

			if err != nil {
				log.Printf("%vE Unable to deserialize event: %v", requestID, err)
				log.Printf("%vE << close: redis error: %v", requestID, err)
				return
			}
			for _, event := range events {
				reStringed, err := json.Marshal(event)
				if err != nil {
					log.Printf(
						"%vE Unable to write event back to string: %v",
						requestID, err,
					)
					continue
				}
				eventsChan <- string(reStringed)
				// We don't log sending the event on this side of the channel.
			}
		}
	}()
	return eventsChan, okChan
}
