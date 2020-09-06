package routes

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"sr"
	"strconv"
	"strings"
)

var tasksRouter = restRouter.PathPrefix("/tasks").Subrouter()

var _ = tasksRouter.HandleFunc("/migrate-events", handleMigrateEvents)

// EventOut is the type of unmanaged JSON
type EventOut map[string]interface{}

func handleMigrateEvents(response Response, request *Request) {
	logRequest(request)
	endID := "+"
	startID := "-"
	batchSize := 50

	conn := sr.RedisPool.Get()
	defer sr.CloseRedis(conn)

	gameID := request.FormValue("gameID")
	if gameID == "" {
		httpBadRequest(response, request, "Invalid game ID")
	}

	for {
		eventsData, err := redis.Values(conn.Do(
			"XREVRANGE", "event:"+gameID,
			endID, startID, "COUNT", batchSize,
		))
		if err != nil {
			logf(request, "Error retrieving events: %v", err)
			httpInternalErrorIf(response, request, err)
		}

		if len(eventsData) == 0 {
			logf(request, "Got empty data back")
			break
		}
		events, err := scanEvents(eventsData)
		if err != nil {
			logf(request, "Unable to parse events: %v", err)
			httpInternalErrorIf(response, request, err)
		}

		for ix, event := range events {
			if ix == len(events)-1 {
				// set id
				startID = event["id"].(string)
			}
			eventID := event["id"].(string)
			eventTy := event["ty"].(string)
			logf(request, "Processing %v event %v", eventID, eventTy)

			newID, err := strconv.Atoi(strings.SplitN(eventID, "-", 2)[0])
			httpInternalErrorIf(response, request, err)
			logf(request, "-> new ID %v", newID)

			core := sr.EventCore{
				ID:         int64(newID),
				Type:       eventTy,
				PlayerID:   sr.UID(event["pID"].(string)),
				PlayerName: event["pName"].(string),
			}

			var out sr.Event
			switch eventTy {
			case sr.EventTypeRoll:
				out = &sr.RollEvent{
					EventCore: core,
					Title:     event["title"].(string),
					Roll:      event["roll"].([]int),
				}
			case sr.EventTypeEdgeRoll:
				out = &sr.EdgeRollEvent{
					EventCore: core,
					Title:     event["title"].(string),
					Rounds:    event["rounds"].([][]int),
				}
			case sr.EventTypeRerollFailures:
				prevID, err := strconv.Atoi(strings.SplitN(event["prevID"].(string), "-", 2)[0])
				httpInternalErrorIf(response, request, err)
				out = &sr.RerollFailuresEvent{
					EventCore: core,
					PrevID:    int64(prevID),
					Title:     event["title"].(string),
					Rounds:    event["rounds"].([][]int),
				}
			case sr.EventTypePlayerJoin:
				out = &sr.PlayerJoinEvent{
					EventCore: core,
				}
			default:
				httpInternalError(response, request, fmt.Sprintf("Found event %v with invalid type %v", newID, eventTy))
			}

			jsonEvent, err := json.Marshal(out)
			httpInternalErrorIf(response, request, err)
			logf(request, "Got event %v", jsonEvent)
		}
	}
	httpSuccess(response, request, "")
}

// scanEvents scans event strings from redis
func scanEvents(eventsData []interface{}) ([]EventOut, error) {
	events := make([]EventOut, len(eventsData))

	for i := 0; i < len(eventsData); i++ {
		eventInfo := eventsData[i].([]interface{})

		eventID := string(eventInfo[0].([]byte))
		fieldList := eventInfo[1].([]interface{})

		eventValue := fieldList[1].([]byte)

		var event map[string]interface{}
		err := json.Unmarshal(eventValue, &event)
		if err != nil {
			return nil, err
		}
		event["id"] = eventID
		events[i] = EventOut(event)
	}
	return events, nil
}

var _ = tasksRouter.HandleFunc("/trim-players", handleTrimPlayers)

func handleTrimPlayers(response Response, request *Request) {
	logRequest(request)
	_, conn, err := requestSession(request)
	httpUnauthorizedIf(response, request, err)
	defer sr.CloseRedis(conn)

	gameID := request.URL.Query().Get("gameID")
	if gameID == "" {
		httpBadRequest(response, request, "No game ID given")
	}
	logf(request, "Trimming players in %v", gameID)
	exists, err := sr.GameExists(gameID, conn)
	httpInternalErrorIf(response, request, err)
	if !exists {
		httpBadRequest(response, request, "No game '"+gameID+"' found")
	}

	sessionKeys, err := redis.Strings(conn.Do("keys", "session:*"))
	httpInternalErrorIf(response, request, err)
	logf(request, "Found %v sessions", len(sessionKeys))
	if len(sessionKeys) == 0 {
		httpInternalError(response, request, "There are no sessions")
	}
	err = conn.Send("MULTI")
	httpInternalErrorIf(response, request, err)

	// var foundPlayers map[string]bool

	for _, key := range sessionKeys {
		sessionID := key[8:]
		logf(request, "Checking for session %v in %v", sessionID, gameID)
		sess, err := sr.GetSessionByID(sessionID, conn)
		logf(request, "Found %v", sess.LogInfo())
		httpInternalErrorIf(response, request, err)
	}
}
