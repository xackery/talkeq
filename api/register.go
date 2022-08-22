package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xackery/log"
	"github.com/xackery/talkeq/registerdb"
	"github.com/xackery/talkeq/userdb"
)

func (t *API) registerConfirm(w http.ResponseWriter, r *http.Request) {
	log := log.New()
	type Resp struct {
		Message string `json:"message"`
	}
	resp := &Resp{}
	var err error
	w.Header().Set("Content-Type", "application/json")
	query := r.URL.Query()
	code := query.Get("code")
	action := query.Get("action")

	if len(code) == 0 {
		resp.Message = "code required"
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}
	if len(action) == 0 {
		resp.Message = "action required"
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}

	entry, err := registerdb.FindByCode(code)
	if err != nil {
		resp.Message = err.Error()
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}

	if entry.Status == "Confirmed" || entry.Status == "Denied" {
		resp.Message = "code used already"
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}

	if strings.ToLower(action) == "deny" {
		err = registerdb.Update(entry.DiscordID, "Denied", time.Now().Add(24*time.Hour).Unix())
		if err != nil {
			log.Warn().Err(err).Msg("registerdb update deny")
		}
		resp.Message = "denied request"
		err = json.NewEncoder(w).Encode(&Resp{})
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}
	if strings.ToLower(action) == "report" {
		err = registerdb.Update(entry.DiscordID, "Reported", time.Now().Add(24*time.Hour).Unix())
		if err != nil {
			log.Warn().Err(err).Msg("registerdb update report")
		}
		resp.Message = "reported request"
		err = json.NewEncoder(w).Encode(&Resp{})
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}
	if strings.ToLower(action) != "accept" {
		resp.Message = "unknown action: " + action
		err = json.NewEncoder(w).Encode(&Resp{})
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}

	userdb.Set(entry.DiscordID, entry.CharacterName)

	err = registerdb.Update(entry.DiscordID, "Confirmed", time.Now().Add(24*time.Hour).Unix())
	if err != nil {
		log.Warn().Err(err).Msg("registerdb update")
	}
	err = t.discord.EditMessage(entry.ChannelID, entry.MessageID, fmt.Sprintf("I sent a /tell to %s, you have 2 minutes to go in game and [ accept ] it. Status: Confirmed", entry.CharacterName))
	if err != nil {
		resp.Message = err.Error()
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Warn().Err(err).Msg("encode response")
		}
		return
	}
	resp.Message = "confirmed successfully"
	err = json.NewEncoder(w).Encode(&Resp{})
	if err != nil {
		log.Warn().Err(err).Msg("encode response")
	}
}
