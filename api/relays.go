package api

import (
	"encoding/json"
	"net/http"

	"github.com/xackery/log"
	"github.com/xackery/talkeq/registerdb"
)

func (t *API) relays(w http.ResponseWriter, r *http.Request) {
	log := log.New()
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
			From:   entry.DiscordName,
			Target: entry.CharacterName,
			Code:   entry.Code,
		}
		resp.Relays = append(resp.Relays, relay)
	}

	log.Debug().Int("relays", len(resp.Relays)).Msg("[api->questapi]")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Warn().Err(err).Msg("encode response")
	}
}
