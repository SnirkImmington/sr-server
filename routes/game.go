package routes

import (
	"github.com/gomodule/redigo/redis"
	"regexp"
	"sr"
	"sr/config"
	"time"
)

var gameRouter = router.PathPrefix("/game").Subrouter()

var _ = router.HandleFunc("/info", handleInfo).Methods("GET")

// GET /info {gameInfo}
func handleInfo(response Response, request *Request) {
	logRequest(request)
	sess, conn, err := requestSession(request)
	if err != nil && httpUnauthorized(response, request, err) {
		return
	}
	defer sr.CloseRedis(conn)

	info, err := sr.GetGameInfo(sess.GameID, conn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}

	err = writeBodyJSON(response, &info)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	httpSuccess(
		response, request,
		info.GameID, ": ", len(info.players), " players",
	)
}

type rollRequest struct {
	Count int    `json:"count"`
	Title string `json:"title"`
	Edge  bool   `json:"edge"`
}

// $ POST /roll count
func handleRoll(response Response, request *Request) {
	logRequest(request)
	sess, conn, err := requestSession(request)
	if err != nil && httpUnauthorized(request, response, err) {
		return
	}
	defer sr.CloseRedis(conn)

	var roll rollRequest
	err = readBodyJSON(request, &roll)
	if err != nil {
		httpInvalidRequest(response, request, "Invalid request")
		return
	}

	if roll.Count > config.MaxSingleRoll {
		httpInvalidRequest(response, request, "Invalid Roll count")
		return
	}

	var event Event
	// Note that roll generation is possibly blocking
	if roll.Edge {
		rolls := sr.ExplodingSixes(roll.Count)
		logf(request, "%v: edge roll: %v",
			sess.LogInfo(), rolls,
		)
		event = sr.EdgeRollEvent{
			EventCore:  EventCore{Type: "edgeRoll"},
			PlayerID:   auth.PlayerID,
			PlayerName: auth.PlayerName,
			Title:      roll.Title,
			Rounds:     rolls,
		}

	} else {
		rolls := make([]int, roll.Count)
		hits := sr.FillRolls(rolls)
		logf(request, "%v: roll: %v (%v hits)",
			sess.LogInfo(), rolls, hits,
		)
		event = RollEvent{
			EventCore:  EventCore{Type: "roll"},
			PlayerID:   auth.PlayerID,
			PlayerName: auth.PlayerName,
			Roll:       rolls,
			Title:      roll.Title,
		}
	}

	id, err := sr.PostEvent(sess.GameID, event, conn)
	if err != nil && httpInternalError(response, request, err) {
		return
	}
	httpSuccess(
		response, request,
		"roll ", id, " posted",
	)
}

// Hacky workaround for logs to show event type.
// A user couldn't actually write "ty":"foo" in the field, though,
// as it'd come back escaped.
var eventParseRegex = regexp.MustCompile(`"ty":"([^"]+)"`)

var _ = gameRouter.HandleFunc("/events", handleEvents).Methods("GET")

// $ GET /events
func handleEvents(response Response, request *Request) {
	logRequest(request)
	sess, conn, err := requestSession(request)
	if err != nil && httpUnauthorized(response, request, err) {
		return
	}
	defer sr.CloseRedis(conn)

	// Upgrade to SSE stream
	stream, err := sseUpgrader.Upgrade(response, request)
	if err != nil {
		httpInternalError(response, request, err)
		return
	}
	err = stream.WriteEvent("ping", []byte("hi"))
	if err != nil {
		logf(request, "Could not say hello: %v", err)
		return
	}

	// Subscribe to redis
	logf(request, "Retrieving events in %v for %v...",
		auth.GameID, auth.PlayerID,
	)

	events, cancelled := sr.ReceiveEvents(auth.GameID)
	defer func() { cancelled <- true }()

	selectInterval := time.Duration(config.SSEPingSecs) * time.Second
	for {
		if !stream.IsOpen() {
			logf(request, "Session %s disconnected", sess)
			ok, err := sr.UnexpireSession(sess, conn)
			if err != nil {
				logf(request,
					"Error unexpiring session %s: %v",
					sess, err,
				)
			} else if !ok {
				logf(request, "Redis did not expire session %s", sess)
			}
			return
		}

		select {
		case event, open := <-events:
			if open {
				eventTy := eventParseRegex.FindString(event)
				logf(request, "Sending %v to %v",
					eventTy[5:], auth,
				)
				err := stream.WriteString(event)
				if err != nil {
					logf(request, "Unable to write to stream: %v", err)
					stream.Close()
					return
				}
			} else {
				stream.Close()
				return
			}
		case <-time.After(selectInterval):
			err := stream.WriteEvent("ping", []byte("hi"))
			if err != nil {
				logf(request, "Unable to ping stream: %v", err)
				stream.Close()
			}
		}
	}
}

type eventRangeResponse struct {
	Events []map[string]interface{} `json:"events"`
	LastID string                   `json:"lastId"`
	More   bool                     `json:"more"`
}

var _ = gameRouter.HandleFunc("/event-range", handleEventRange).Methods("GET")

/*
   on join: { start: '', end: <lastEventID> }, backfill buffer
  -> [ {id: <some-early-id>, ... } ]
  if there's < max responses, client knows it's hit the boundary.
*/
// GET /event-range { start: <id>, end: <id>, max: int }
func handleEventRange(response Response, request *Request) {
	logRequest(request)
	sess, conn, err := requestSession(request)
	if err != nil && httpUnauthorized(response, request, err) {
		return
	}
	defer sr.CloseRedis(conn)

	newest := request.FormValue("newest")
	oldest := request.FormValue("oldest")

	// We want to be careful here because these IDs are user input!
	//

	if newest == "" {
		newest = "-"
	} else if !validEventID(newest) {
		httpInvalidRequest(response, request, "Invalid newest ID")
		return
	}

	if oldest == "" {
		oldest = "+"
	} else if !validEventID(oldest) {
		httpInvalidRequest(response, request, "Invalid oldest ID")
		return
	}

	logf(request, "Retrieve events {%s : %s} for %s",
		oldest, newest, sess.LogInfo(),
	)

	// TODO move to events.go
	eventsData, err := redis.Values(conn.Do(
		"XREVRANGE", "event:"+auth.GameID,
		eventsRange.Oldest, eventsRange.Newest,
		"COUNT", config.MaxEventRange,
	))
	if err != nil {
		logf(request, "Unable to list events from redis")
		httpInternalError(response, request, err)
		return
	}

	var eventRange eventRangeResponse
	var message string

	if len(eventsData) == 0 {
		eventRange = eventRangeResponse{
			Events: make([]map[string]interface{}, 0),
			LastID: "",
			More:   false,
		}
		message = "0 events"
	} else {
		events, err := scanEvents(eventsData)
		if err != nil {
			logf(request, "Unable to parse events: %v", err)
			httpInternalError(response, request, err)
			return
		}

		firstID := events[0]["id"].(string)
		lastID := events[len(events)-1]["id"].(string)

		eventRange = eventRangeResponse{
			Events: events,
			More:   len(events) == config.MaxEventRange,
		}
		message = fmt.Sprintf(
			"%s : %s ; %v events",
			firstID, lastID, len(events),
		)
	}

	err = writeBodyJSON(response, eventRange)
	if err != nil {
		httpInternalError(response, request, err)
		return
	}
	httpSuccess(response, request, message)
}
