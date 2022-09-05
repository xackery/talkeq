package api

import (
	"encoding/json"
	"net/http"

	"github.com/xackery/talkeq/registerdb"
	"github.com/xackery/talkeq/tlog"
)

func (t *API) relays(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type Relay struct {
		Action string `json:"action"`
		From   string `json:"from"`
		Target string `json:"target"`
		Code   string `json:"code"`
	}
	type Resp struct {
		Message string  `json:"message"`
		Relays  []Relay `json:"relays"`
	}

	resp := Resp{}

	entries, err := registerdb.QueuedEntries()
	if err != nil {
		tlog.Warnf("[api] queuedentries failed: %s", err)
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}

	for _, entry := range entries {
		relay := Relay{
			Action: "register",
			From:   entry.DiscordName,
			Target: entry.CharacterName,
			Code:   entry.Code,
		}
		resp.Relays = append(resp.Relays, relay)
	}

	tlog.Debugf("[api->questapi] relays count: %d", len(resp.Relays))
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		tlog.Warnf("[api] encode response failed: %s", err)
	}
}
