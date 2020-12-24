package sr

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"math/rand"
	"strings"
)

// ErrPlayerNotFound means a player was not found.
var ErrPlayerNotFound = errors.New("player not found")

// Player is a user of Shadowroller.
//
// Players may be registered for a number of games.
// Within those games, they may have a number of chars.
type Player struct {
	ID       UID    `redis:"-"`
	Username string `redis:"uname"`
	Name     string `redis:"name"`
	Hue      int    `redis:"hue"`
}

// PlayerInfo is data other players can see about a player.
//
// - `username` is not shown.
type PlayerInfo struct {
	ID   UID    `json:"-"`
	Name string `json:"name"`
	Hue  int    `json:"hue"`
}

func (p *Player) String() string {
	return fmt.Sprintf(
		"%v (%v / %v)", p.ID, p.Username, p.Name,
	)
}

// Info returns game-readable information about the player
func (p *Player) Info() PlayerInfo {
	return PlayerInfo{
		ID:   p.ID,
		Name: p.Name,
		Hue:  p.Hue,
	}
}

func (p *Player) redisKey() string {
	if p == nil || p.ID == "" {
		panic("Attempted to call redisKey() on nil player")
	}
	return "player:" + string(p.ID)
}

// NewPlayer constructs a new Player object, giving it a UID
func NewPlayer(username string, name string) Player {
	return Player{
		ID:       GenUID(),
		Username: username,
		Name:     name,
		Hue:      RandomPlayerHue(),
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
			"%w: empty PlayerID passed to GetPlayerByID", errNilPlayer,
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

// GetPlayerIDOf returns the playerID for the given username.
func GetPlayerIDOf(username string, conn redis.Conn) (string, error) {
	if username == "" {
		return "", fmt.Errorf("empty username passed to GetPlayerIDOf")
	}
	playerID, err := redis.String(conn.Do("GET", "player_id:"+username))
	if err != nil {
		return "", fmt.Errorf(
			"redis error getting player ID of %v: %w", username, err,
		)
	}
	if playerID == "" {
		return "", fmt.Errorf("player %v not found: %w", username, ErrPlayerNotFound)
	}
	return playerID, nil
}

// GetPlayerByUsername retrieves a player based on the username given.
// Returns ErrPlayerNotFound if no player is found.
func GetPlayerByUsername(username string, conn redis.Conn) (*Player, error) {
	if username == "" {
		return nil, fmt.Errorf("empty username passed to GetPlayerByUsername")
	}

	playerID, err := redis.String(conn.Do("GET", "player_ids:"+username))
	if err != nil {
		return nil, fmt.Errorf("redis error checking `players` mapping: %w", err)
	}
	if playerID == "" {
		return nil, fmt.Errorf("%w: %v", ErrPlayerNotFound, username)
	}

	return GetPlayerByID(playerID, conn)
}

// RandomPlayerHue creates a random hue value for a player
func RandomPlayerHue() int {
	return rand.Intn(360)
}

// ValidPlayerName determines if a player name is valid.
// It checks for 1-32 chars with no newlines.
func ValidPlayerName(name string) bool {
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

// UpdatePlayer updates a player in the database.
// It does not allow for username updates. It only publishes the update to the given game.
func UpdatePlayer(gameID string, playerID UID, update *PlayerDiffUpdate, conn redis.Conn) error {
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
