package sr

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
)

// Char is a character created by a char.
//
// Chars may be char characters or minions (spirits, drones, etc.)
// They are intended to be units which have stats.
type Char struct {
	ID     UID    `redis:"-"`
	CharID UID    `redis:"pID"`
	Name   string `redis:"name"`
	Hue    byte   `redis:"hue"`
}

func (c *Char) String() string {
	return fmt.Sprintf(
		"Char %v (%v)", c.ID, c.Name,
	)
}

func (c *Char) redisKey() string {
	if c == nil || c.ID == "" {
		panic("Attempted to call redisKey() on nil char")
	}
	return "char:" + string(c.ID)
}

// NewChar creates a Char and generates an ID for it
func NewChar(charID UID, name string, hue byte) Char {
	return Char{
		ID:     GenUID(),
		CharID: charID,
		Name:   name,
		Hue:    hue,
	}
}

var errNilChar = errors.New("nil CharID requested")

// CharExists determines if a char with the given ID exists in the database
func CharExists(charID string, conn redis.Conn) (bool, error) {
	if charID == "" {
		return false, fmt.Errorf(
			"empty CharID passed to CharExists: %w", errNilChar,
		)
	}
	return redis.Bool(conn.Do("exists", "char:"+charID))
}

// GetCharByID retrieves a char from Redis
func GetCharByID(charID string, conn redis.Conn) (*Char, error) {
	if charID == "" {
		return nil, fmt.Errorf(
			"empty CharID passed to GetCharByID: %w", errNilChar,
		)
	}
	var char Char
	data, err := conn.Do("HGETALL", "char:"+charID)
	if err != nil {
		return nil, fmt.Errorf(
			"redis error retrieving data for %v: %w", charID, err,
		)
	}
	if data == nil || len(data.([]interface{})) == 0 {
		return nil, fmt.Errorf(
			"empty data from redis for %v", charID,
		)
	}
	err = redis.ScanStruct(data.([]interface{}), &char)
	if err != nil {
		return nil, fmt.Errorf(
			"redis error parsing char %v: %w", charID, err,
		)
	}
	if char.Name == "" {
		return nil, fmt.Errorf(
			"no data for %v after redis parse", charID,
		)
	}
	char.ID = UID(charID)
	return &char, nil
}

// CreateChar adds the given Char to the database
func CreateChar(playerID UID, char *Char, conn redis.Conn) error {
	err := conn.Send("MULTI")
	if err != nil {
		return fmt.Errorf("redis error sending `MULTI`: %w", err)
	}

	err = conn.Send("SADD", "chars:"+string(playerID), char.ID)
	if err != nil {
		return fmt.Errorf("redis error sending `SADD`: %w", err)
	}

	charData := redis.Args{}.Add(char.redisKey()).AddFlat(char)
	err = conn.Send("HSET", charData...)
	if err != nil {
		return fmt.Errorf("redis error sending `HSET`: %w", err)
	}

	data, err := redis.Ints(conn.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("redis error sending `EXEC`: %w", err)
	}

	if len(data) != 2 || data[0] != 1 || data[1] <= 3 {
		return fmt.Errorf("redis error with multi: expected [1, >1], got %v", data)
	}
	return nil
}
