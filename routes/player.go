package routes

import (
	"fmt"
	"sr"
	"strings"
)

var playerRouter = restRouter.PathPrefix("/player").Subrouter()

var _ = playerRouter.HandleFunc("/update", handleUpdatePlayer).Methods("POST")

func handleUpdatePlayer(response Response, request *Request) {
	logRequest(request)
	sess, conn, err := requestSession(request)
	defer closeRedis(request, conn)
	httpUnauthorizedIf(response, request, err)

	var requestDiff map[string]interface{}
	err = readBodyJSON(request, &requestDiff)
	httpUnauthorizedIf(response, request, err)
	logf(request,
		"%v requests update %v", sess.PlayerInfo(), requestDiff,
	)
	update := sr.MakePlayerDiffUpdate(sess.PlayerID)

	for key, value := range requestDiff {
		switch key {
		case "name":
			name, ok := value.(string)
			if !ok {
				httpBadRequest(response, request, "name: expected string")
			}
			name = strings.TrimSpace(name)
			if !sr.ValidPlayerName(name) {
				httpBadRequest(response, request, "name: invalid")
			}
			update.AddField("name", name)
		case "hue":
			hue, ok := value.(int)
			if !ok || hue < 0 || hue > 360 {
				httpBadRequest(response, request, "hue: expected int 0-360")
			}
			update.AddField("hue", hue)
		default:
			httpBadRequest(response, request,
				fmt.Sprintf("Cannot update field %v", key),
			)
		}
	}
	// TODO update is only sent to a specific game. Need pubsub for players :/
	err = sr.UpdatePlayer(sess.GameID, sess.PlayerID, &update, conn)
	httpInternalErrorIf(response, request, err)
	httpSuccess(response, request,
		"Player ", sess.PlayerID, " update ", update.Diff,
	)
}