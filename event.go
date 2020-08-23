package sr

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"log"
	"regexp"
)

// PostEvent posts an event to Redis and returns the generated ID.
func PostEvent(gameID string, event Event, conn redis.Conn) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = conn.Do("ZADD", "history:"+gameID, "NX", event.GetID(), bytes)
	//id, err := redis.String(conn.Do("XADD", "event:"+gameID, "*", "payload", bytes))
	if err != nil {
		return err
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
		return "", err
	}

	return events[0], nil
}

var idRegex = regexp.MustCompile(`^([\d]{13})-([\d]+)$`)

// ValidEventID returns whether the non-empty-string id is valid.
func ValidEventID(id string) bool {
	return idRegex.MatchString(id)
}

// ReceiveEvents subscribes to the event stream for a given game
func ReceiveEvents(gameID string, requestID string) (<-chan string, chan<- bool) {
	// Events channel is buffered: if there are too many events for our consumer,
	// we will block on channel send.
	eventsChan := make(chan string, 10)
	okChan := make(chan bool)
	go func() {
		defer close(eventsChan)

		conn := RedisPool.Get()
		defer CloseRedis(conn)

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
