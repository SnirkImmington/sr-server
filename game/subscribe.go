package game

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

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

func subscribeTask(cleanup func(), messages chan Message, errs chan error, sub redis.PubSubConn, ctx context.Context) {
	defer cleanup()
	for {
		// Check if we have received done yet
		select {
		case <-ctx.Done():
			errs <- fmt.Errorf("received done from context: err = %w", ctx.Err())
			return
		default:
			log.Print("No Done() this ping")
		}
		const pollInterval = time.Duration(4) * time.Second
		// Receive an event or update from the game
		switch msg := sub.ReceiveWithTimeout(pollInterval).(type) {
		case error:
			log.Printf("Received error %#v", msg)
			var netError net.Error
			if errors.As(msg, &netError) {
				if netError.Timeout() {
					log.Print("Subscription helper loop")
					continue
				}
			}
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
			log.Printf("Helper: %#v", msg)
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
