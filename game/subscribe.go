package game

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gomodule/redigo/redis"
	redisUtil "sr/redis"
)

type MessageType int

const MessageTypeEvent MessageType = MessageType(1)
const MessageTypeUpdate MessageType = MessageType(2)

type Message struct {
	Type MessageType
	Body string
}

// This task becomes a memory leak as we cannot signal for it to close effectively.
// The goroutine will remain open after a client disconnects, until the next event
// or update is sent in the game. The underlying connection may eventually be reclaimed by the redis pool.
// This is due to a limitation in the redigo library - we cannot check for an
// external close message while also listening on the connection without entering a reconnect loop.
// Because this leak is not indefinite, and can only grow trivally large given current traffic,
// and because I'd like to replace the library with one that can handle pipelining anyway,
// I'm going to ignore it for now and switch libraries by next major release.
func subscribeTask(cleanup func(), messages chan Message, errs chan error, sub redis.PubSubConn, ctx context.Context) {
	defer cleanup()
	for {
		// Check if we have received done yet
		select {
		case <-ctx.Done():
			errs <- fmt.Errorf("received done from context: err = %w", ctx.Err())
			return
		default:
		}
		// Receive an event or update from the game
		switch msg := sub.Receive().(type) {
		case error:
			log.Printf("Received error %#v", msg)
			errs <- fmt.Errorf("from redis Receive(): %w", msg)
			return
		case redis.Message:
			var message Message
			messageText := string(msg.Data)
			if strings.HasPrefix(msg.Channel, "history") {
				message = Message{Type: MessageTypeEvent, Body: messageText}
			} else {
				message = Message{Type: MessageTypeUpdate, Body: messageText}
			}
			messages <- message
		case redis.Subscription:
			// okay; ignore
		default:
			errs <- fmt.Errorf("unexpected value for Receive(): %#v", msg)
		}
	}
}

func Subscribe(gameID string, messages chan Message, errors chan error, ctx context.Context) error {
	conn, err := redisUtil.ConnectWithContext(ctx)
	if err != nil {
		close(errors)
		close(messages)
		return fmt.Errorf("dialing redis with context: %w", err)
	}
	sub := redis.PubSubConn{Conn: conn}

	cleanup := func() {
		log.Print("Cleaning up subscribe task")
		sub.Unsubscribe()
		redisUtil.Close(conn)
		close(errors)
		close(messages)
	}

	if err := sub.Subscribe("history:"+gameID, "update:"+gameID); err != nil {
		cleanup()
		return fmt.Errorf("subscribing to events and history: %w", err)
	}
	go subscribeTask(cleanup, messages, errors, sub, ctx)
	return nil
}
