package redis

import (
	"github.com/gomodule/redigo/redis"
	"log"
	"os"
	"sr/config"
	"time"
)

var logger *log.Logger

// RedisPool is the pool of redis connections
var pool = &redis.Pool{
	MaxIdle:     10,
	IdleTimeout: time.Duration(60) * time.Second,
	Dial: func() (redis.Conn, error) {
		conn, err := redis.DialURL(config.RedisURL)
		if config.RedisDebug && err == nil {
			return redis.NewLoggingConn(conn, logger, "redis"), nil
		}
		return conn, err
	},
}

// FromPool opens a connection from the redis pool
func FromPool() redis.Conn {
	return pool.Get()
}

// Close closes a redis connection and logs errors if they occur
func Close(conn redis.Conn) {
	err := conn.Close()
	if err != nil {
		log.Printf("Error closing redis connection: %v", err)
	}
}

func SetupWithConfig() {
	if config.RedisDebug {
		logger = log.New(os.Stdout, "", log.Ltime|log.Lshortfile)
	}
}
