package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xackery/talkeq/registerdb"
	"github.com/xackery/talkeq/tlog"
	"github.com/xackery/talkeq/userdb"
)

func (t *API) registerConfirm(w http.ResponseWriter, r *http.Request) {
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
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}
	if len(action) == 0 {
		resp.Message = "action required"
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}

	entry, err := registerdb.FindByCode(code)
	if err != nil {
		resp.Message = err.Error()
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}

	if entry.Status == "Confirmed" || entry.Status == "Denied" {
		resp.Message = "code used already"
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}

	if strings.ToLower(action) == "deny" {
		err = registerdb.Update(entry.DiscordID, "Denied", time.Now().Add(24*time.Hour).Unix())
		if err != nil {
			tlog.Warnf("[api] registerdb update deny failed: %s", err)
		}
		resp.Message = "denied request"
		err = json.NewEncoder(w).Encode(&Resp{})
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}
	if strings.ToLower(action) == "report" {
		err = registerdb.Update(entry.DiscordID, "Reported", time.Now().Add(24*time.Hour).Unix())
		if err != nil {
			tlog.Warnf("[api] registerdb update report failed: %s", err)
		}
		resp.Message = "reported request"
		err = json.NewEncoder(w).Encode(&Resp{})
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}
	if strings.ToLower(action) != "accept" {
		resp.Message = "unknown action: " + action
		err = json.NewEncoder(w).Encode(&Resp{})
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}

	userdb.Set(entry.DiscordID, entry.CharacterName)

	err = registerdb.Update(entry.DiscordID, "Confirmed", time.Now().Add(24*time.Hour).Unix())
	if err != nil {
		tlog.Warnf("[api] registerdb update failed: %s", err)
	}
	err = t.discord.EditMessage(entry.ChannelID, entry.MessageID, fmt.Sprintf("I sent a /tell to %s, you have 2 minutes to go in game and [ accept ] it. Status: Confirmed", entry.CharacterName))
	if err != nil {
		resp.Message = err.Error()
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			tlog.Warnf("[api] encode response failed: %s", err)
		}
		return
	}
	resp.Message = "confirmed successfully"
	err = json.NewEncoder(w).Encode(&Resp{})
	if err != nil {
		tlog.Warnf("[api] encode response failed: %s", err)
	}
}
