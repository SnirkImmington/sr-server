package sr

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
)

// Player is a user of Shadowroller.
//
// Players may be registered for a number of games.
// Within those games, they may have a number of chars.
type Player struct {
	ID       UID    `redis:"-"`
	Username string `redis:"uname"`
}

func (p *Player) String() string {
	return fmt.Sprintf(
		"%v (%v)", p.ID, p.Username,
	)
}

func (p *Player) redisKey() string {
	if p == nil || p.ID == "" {
		panic("Attempted to call redisKey() on nil player")
	}
	return "player:" + string(p.ID)
}

// NewPlayer constructs a new Player object, giving it a UID
func NewPlayer(username string) Player {
	return Player{
		ID:       GenUID(),
		Username: username,
	}
}

var errNilPlayer = errors.New("nil PlayerID requested")
var errNoPlayer = errors.New("player not found")

// PlayerExists determines if a player with the given ID exists in the database
func PlayerExists(playerID string, conn redis.Conn) (bool, error) {
	if playerID == "" {
		return false, fmt.Errorf(
			"empty PlayerID passed to PlayerExists: %w", errNilPlayer,
		)
	}
	return redis.Bool(conn.Do("exists", "player:"+playerID))
}

// GetPlayerByID retrieves a player from Redis
func GetPlayerByID(playerID string, conn redis.Conn) (*Player, error) {
	if playerID == "" {
		return nil, fmt.Errorf(
			"empty PlayerID passed to GetPlayerByID: %w", errNilPlayer,
		)
	}
	var player Player
	data, err := conn.Do("hgetall", "player:"+playerID)
	if err != nil {
		return nil, fmt.Errorf(
			"redis error retrieving data for %v: %w", playerID, err,
		)
	}
	if data == nil || len(data.([]interface{})) == 0 {
		return nil, fmt.Errorf(
			"empty data from redis for %v: %w", playerID, errNoPlayer,
		)
	}
	err = redis.ScanStruct(data.([]interface{}), &player)
	if err != nil {
		return nil, fmt.Errorf(
			"redis error parsing player %v: %w", playerID, err,
		)
	}
	if player.Username == "" {
		return nil, fmt.Errorf(
			"no data for %v after redis parse: %w", playerID, errNoPlayer,
		)
	}
	player.ID = UID(playerID)
	return &player, nil
}

// GetPlayerByUsername retrieves a player based on the username given.
// Returns nil if no player with that username was found!
func GetPlayerByUsername(username string, conn redis.Conn) (*Player, error) {
	if username == "" {
		return nil, fmt.Errorf("empty username passed to GetPlayerByUsername")
	}

	playerID, err := redis.String(conn.Do("HGET", "players", username))
	if err != nil {
		return nil, fmt.Errorf("redis error checking `players` mapping: %w", err)
	}
	if playerID == "" {
		return nil, nil
	}

	return GetPlayerByID(playerID, conn)
}

// GetPlayerCharIDs returns the IDs of all the chars of a player
func GetPlayerCharIDs(playerID UID, conn redis.Conn) ([]UID, error) {
	ids, err := redis.Strings(conn.Do("SGETALL", "chars:"+string(playerID)))
	if err != nil {
		return nil, err
	}
	uids := make([]UID, len(ids))
	for ix, id := range ids {
		uids[ix] = UID(id)
	}
	return uids, nil
}

// GetPlayerChars returns the chars of a given player
func GetPlayerChars(playerID UID, conn redis.Conn) ([]Char, error) {
	ids, err := GetPlayerCharIDs(playerID, conn)
	if err != nil {
		return nil, fmt.Errorf("error from GetPlayerCharIDs: %w", err)
	}
	err = conn.Send("MULTI")
	if err != nil {
		return nil, fmt.Errorf("redis error sending `MULTI`: %w", err)
	}
	found := make([]Char, len(ids))
	for _, charID := range ids {
		err = conn.Send("HGETALL", "char:"+string(charID))
		if err != nil {
			return nil, fmt.Errorf(
				"redis error sending `HGETALL` for %v: %w", charID, err,
			)
		}
	}
	charsData, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("redis error sending `EXEC`: %w", err)
	}

	for ix, charData := range charsData {
		var char Char
		err = redis.ScanStruct(charData.([]interface{}), &char)
		if err != nil {
			return nil, fmt.Errorf(
				"redis error parsing char #%v %v: %w", ix, ids[ix], err,
			)
		}
		if char.Name == "" {
			return nil, fmt.Errorf(
				"no data for char #%v %v after redis parse", ix, ids[ix],
			)
		}
		found[ix] = char
	}
	return found, nil
}

// CreatePlayer adds the given Player to the database
func CreatePlayer(player *Player, conn redis.Conn) error {
	err := conn.Send("MULTI")
	if err != nil {
		return fmt.Errorf("redis error sending `MULTI`: %w", err)
	}

	err = conn.Send("HSET", "player_ids", player.Username, player.ID)
	if err != nil {
		return fmt.Errorf("redis error sending `player_ids` `HSET`: %w", err)
	}

	playerData := redis.Args{}.Add(player.redisKey()).AddFlat(player)
	err = conn.Send("HSET", playerData...)
	if err != nil {
		return fmt.Errorf("redis error sending `player:id` `HSET`: %w", err)
	}

	data, err := redis.Ints(conn.Do("Exec"))
	if err != nil {
		return fmt.Errorf("redis error sending `EXEC`: %w", err)
	}

	// expected 1 update for player_ids, n updates for player fields
	if len(data) != 2 || data[0] != 1 || data[1] <= 2 {
		return fmt.Errorf("redis error with multi: expected [1, >1], got %v", data)
	}
	return nil
}
