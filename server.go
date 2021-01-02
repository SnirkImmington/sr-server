package sr

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"log"
	"os"
	"sr/config"
	"strings"
	"time"
)

var redisLogger *log.Logger

// RedisPool is the pool of redis connections
var RedisPool = &redis.Pool{
	MaxIdle:     10,
	IdleTimeout: time.Duration(60) * time.Second,
	Dial: func() (redis.Conn, error) {
		conn, err := redis.DialURL(config.RedisURL)
		if config.RedisDebug && err == nil {
			return redis.NewLoggingConn(conn, redisLogger, "redis"), nil
		}
		return conn, err
	},
}

// CloseRedis closes a redis connection and logs errors if they occur
func CloseRedis(conn redis.Conn) {
	err := conn.Close()
	if err != nil {
		log.Printf("Error closing redis connection: %v", err)
	}
}

func addHardcodedGames(conn redis.Conn) error {
	gameKeys, err := redis.Strings(conn.Do("keys", "game:*"))
	if err != nil {
		return fmt.Errorf("Error reading games from redis: %w", err)
	}
	for i, gameKey := range gameKeys {
		gameKeys[i] = gameKey[5:]
	}
	log.Printf("Games (%v): %v", len(gameKeys), gameKeys)
	if !config.IsProduction && len(gameKeys) < len(config.HardcodedGameNames) {
		for i, game := range config.HardcodedGameNames {
			if _, err := conn.Do("hmset", "game:"+game, "event_id", 0); err != nil {
				return fmt.Errorf("Unable to add game #%v, %v: %w", i+1, game, err)
			}
		}
	}
	return nil
}

func addHardcodedPlayers(conn redis.Conn) error {
	playerKeys, err := redis.Strings(conn.Do("keys", "player:*"))
	if err != nil {
		return fmt.Errorf("getting player keys from redis: %w", err)
	}
	playerMapping, err := redis.StringMap(conn.Do("hgetall", "player_ids"))
	if err != nil {
		return fmt.Errorf("getting player ID mapping from redis: %w", err)
	}
	players := make([]string, len(playerKeys))
	if err != nil {
		return fmt.Errorf("Reading players from redis: %w", err)
	}
	for i, playerKey := range playerKeys {
		playerID := playerKey[7:]
		for username, mappedID := range playerMapping {
			if mappedID == playerID {
				players[i] = username
				break
			}
		}
		if players[i] == "" {
			log.Printf("Need mapping for %v -> %v", playerID)
		}
	}
	log.Printf("Players (%v): %v", len(players), players)
	if !config.IsProduction && len(players) < len(config.HardcodedUsernames) {
		log.Printf("Adding %v to all games", config.HardcodedUsernames)
		games := config.HardcodedGameNames
		for _, username := range config.HardcodedUsernames {
			player := NewPlayer(username, strings.Title(username))
			err := CreatePlayer(&player, conn)
			if err != nil {
				return fmt.Errorf("creating %v: %w", username, err)
			}
			for _, gameID := range games {
				err := AddPlayerToGame(&player, gameID, conn)
				if err != nil {
					return fmt.Errorf("adding %v to %v: %w", username, gameID)
				}
			}
		}
	}
	return nil
}

// SetupRedis adds the game names from the config to Redis
// and sets up Redis logging if enabled.
func SetupRedis() {
	if config.RedisDebug {
		redisLogger = log.New(os.Stdout, "", log.Ltime|log.Lshortfile)
	}
	conn := RedisPool.Get()
	defer CloseRedis(conn)

	if err := addHardcodedGames(conn); err != nil {
		panic(fmt.Errorf("Error adding hardcoded games: %w", err))
	}
	if err := addHardcodedPlayers(conn); err != nil {
		panic(fmt.Errorf("Error adding hardcoded players: %w", err))
	}
}
