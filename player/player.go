package player

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"math/rand"
	"sr/update"
	"strings"
)

// ErrPlayerNotFound means a player was not found.
var ErrPlayerNotFound = errors.New("player not found")

// Player is a user of Shadowroller.
//
// Players may be registered for a number of games.
// Within those games, they may have a number of chars.
type Player struct {
	ID       sr.UID `json:"id" redis:"-"`
	Username string `json:"username" redis:"uname"`
	Name     string `json:"name" redis:"name"`
	Hue      int    `json:"hue" redis:"hue"`
}

// Info is data other players can see about a player.
// - `username` is not shown.
type Info struct {
	ID   UID    `json:"id"`
	Name string `json:"name"`
	Hue  int    `json:"hue"`
}

func (p *Player) String() string {
	return fmt.Sprintf(
		"%v (%v / %v)", p.ID, p.Username, p.Name,
	)
}

// Info returns game-readable information about the player
func (p *Player) Info() Info {
	return Info{
		ID:   p.ID,
		Name: p.Name,
		Hue:  p.Hue,
	}
}

// RedisKey is the key for acccessing player info from redis
func (p *Player) RedisKey() string {
	if p == nil || p.ID == "" {
		panic("Attempted to call redisKey() on nil player")
	}
	return "player:" + string(p.ID)
}

// MakePlayer constructs a new Player object, giving it a UID
func MakePlayer(username string, name string) Player {
	return Player{
		ID:       sr.GenUID(),
		Username: username,
		Name:     name,
		Hue:      RandomHue(),
	}
}

var ErrNilPlayer = errors.New("nil PlayerID requested")

// Exists determines if a player with the given ID exists in the database
func Exists(playerID string, conn redis.Conn) (bool, error) {
	if playerID == "" {
		return false, fmt.Errorf(
			"empty PlayerID passed to PlayerExists: %w", errNilPlayer,
		)
	}
	return redis.Bool(conn.Do("exists", "player:"+playerID))
}

// GetByID retrieves a player from Redis
func GetByID(playerID string, conn redis.Conn) (*Player, error) {
	if playerID == "" {
		return nil, fmt.Errorf(
			"%w: empty PlayerID passed to GetPlayerByID", errNilPlayer,
		)
	}
	var player Player
	data, err := conn.Do("hgetall", "player:"+playerID)
	if errors.Is(err, redis.ErrNil) {
		return nil, fmt.Errorf(
			"%w: %v", ErrPlayerNotFound, playerID,
		)
	} else if err != nil {
		return nil, fmt.Errorf(
			"redis error retrieving data for %v: %w", playerID, err,
		)
	}
	if data == nil || len(data.([]interface{})) == 0 {
		return nil, fmt.Errorf(
			"%w: no data for %v", ErrPlayerNotFound, playerID,
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

// GetIDOf returns the playerID for the given username.
func GetIDOf(username string, conn redis.Conn) (string, error) {
	if username == "" {
		return "", fmt.Errorf("%w: empty username passed to GetPlayerIDOf", errNilPlayer)
	}
	playerID, err := redis.String(conn.Do("GET", "player_id:"+username))
	if errors.Is(err, redis.ErrNil) {
		return "", fmt.Errorf(
			"%w: %v", ErrPlayerNotFound, username,
		)
	} else if err != nil {
		return "", fmt.Errorf(
			"redis error getting player ID of %v: %w", username, err,
		)
	}
	if playerID == "" {
		return "", fmt.Errorf("%w: %v (empty string stored)", ErrPlayerNotFound, username)
	}
	return playerID, nil
}

// GetByUsername retrieves a player based on the username given.
// Returns ErrPlayerNotFound if no player is found.
func GetByUsername(username string, conn redis.Conn) (*Player, error) {
	if username == "" {
		return nil, fmt.Errorf("empty username passed to GetPlayerByUsername")
	}

	playerID, err := redis.String(conn.Do("HGET", "player_ids", username))
	if errors.Is(err, redis.ErrNil) {
		return nil, fmt.Errorf("%w: %v", ErrPlayerNotFound, username)
	} else if err != nil {
		return nil, fmt.Errorf("redis error checking `player_ids`: %w", err)
	}
	if playerID == "" {
		return nil, fmt.Errorf("%w: %v (empty string stored)", ErrPlayerNotFound, username)
	}

	return GetByID(playerID, conn)
}

// RandomHue creates a random hue value for a player
func RandomHue() int {
	return rand.Intn(360)
}

// ValidName determines if a player name is valid.
// It checks for 1-32 chars with no newlines.
func ValidName(name string) bool {
	return len(name) > 0 && len(name) < 32 && !strings.ContainsAny(name, "\r\n")
}

/*
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
*/

// Create adds the given Player to the database
func Create(player *Player, conn redis.Conn) error {
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

// Update updates a player in the database.
// It does not allow for username updates. It only publishes the update to the given game.
func Update(gameID string, playerID UID, update update.Player, conn redis.Conn) error {
	playerData := redis.Args{}.Add("player:" + playerID).AddFlat(update.Diff)
	updateBytes, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("unable to marshal update to JSON :%w", err)
	}

	// MULTI: update player, publish update
	if err := conn.Send("MULTI"); err != nil {
		return fmt.Errorf("redis error sending MULTI for player update: %w", err)
	}
	if err = conn.Send("HSET", playerData...); err != nil {
		return fmt.Errorf("redis error sending HSET for player update: %w", err)
	}
	if err = conn.Send("PUBLISH", "update:"+gameID, updateBytes); err != nil {
		return fmt.Errorf("redis error sending event publish: %w", err)
	}
	// EXEC: [#updated>0, #players]
	results, err := redis.Ints(conn.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("redis error sending EXEC: %w", err)
	}
	if len(results) != 2 {
		return fmt.Errorf("redis error updating player, expected 2 results got %v", results)
	}
	if results[0] <= 0 {
		return fmt.Errorf("redis error updating player, expected [1, *] got %v", results)
	}
	return nil
}
