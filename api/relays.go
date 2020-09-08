package api

import (
	"encoding/json"
	"net/http"

	"github.com/xackery/log"
)

func (t *API) relays(w http.ResponseWriter, r *http.Request) {
	log := log.New()
	w.Header().Set("Content-Type", "application/json")
	type Relay struct {
		Action  string `json:"action"`
		From    string `json:"from"`
		Target  string `json:"target"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	type Resp struct {
		Message string  `json:"message"`
		Relays  []Relay `json:"relays"`
	}

	resp := Resp{}

	entries, err := t.registerManager.QueuedEntries()
	if err != nil {
		log.Warn().Err(err).Msg("queuedentries")
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}

	for _, entry := range entries {
		relay := Relay{
			Action: "register",
			From:   entry.DiscordID,
		}
		resp.Relays = append(resp.Relays, relay)
	}

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Warn().Err(err).Msg("encode response")
	}
}
