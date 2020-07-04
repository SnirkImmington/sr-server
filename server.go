package sr

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"log"
	"sr/config"
	"strings"
	"time"
)

// RedisPool is the pool of redis connections
var RedisPool = &redis.Pool{
	MaxIdle:     10,
	IdleTimeout: time.Duration(60) * time.Second,
	Dial: func() (redis.Conn, error) {
		return redis.DialURL(config.RedisURL)
	},
}

// CloseRedis closes a redis connection and logs errors if they occur
func CloseRedis(conn redis.Conn) {
	err := conn.Close()
	if err != nil {
		log.Printf("Error closing redis connection: %v", err)
	}
}

func RegisterDefaultGames() {
	conn := RedisPool.Get()
	defer CloseRedis(conn)

	gameNames := strings.Split(config.HardcodedGameNames, ",")

	for _, game := range gameNames {
		_, err := conn.Do("hmset", "game:"+game, "event_id", 0)
		if err != nil {
			panic(fmt.Sprintf("Unable to connect to redis: ", err))
		}
	}

	log.Print("Registered ", len(gameNames), " hardcoded game IDs.")
}
