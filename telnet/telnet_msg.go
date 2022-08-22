package telnet

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xackery/log"
	"github.com/xackery/talkeq/guilddb"
	"github.com/xackery/talkeq/request"
)

var (
	oldItemLink = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{39}([A-Za-z-'`.,!?* ]+)\\x12")
	newItemLink = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{50}([A-Za-z-'`.,!?* ]+)\\x12")
)

func (t *Telnet) convertLinks(message string) string {

	matches := newItemLink.FindAllStringSubmatchIndex(message, -1)
	if len(matches) == 0 {
		matches = oldItemLink.FindAllStringSubmatchIndex(message, -1)
	}
	out := message
	for _, submatches := range matches {
		if len(submatches) < 6 {
			continue
		}
		itemLink := message[submatches[2]:submatches[3]]

		itemID, _ := strconv.ParseInt(itemLink, 16, 32)
		//TODO: smarter debugging
		//if err != nil {

		//}
		itemName := message[submatches[4]:submatches[5]]

		out = message[0:submatches[0]]
		if itemID > 0 && len(t.config.ItemURL) > 0 {
			out += fmt.Sprintf("%s%d (%s)", t.config.ItemURL, itemID, itemName)
		} else {
			out += fmt.Sprintf("*%s* ", itemName)
		}
		out += message[submatches[1]:]
		out = strings.TrimSpace(out)
		out = t.convertLinks(out)
		break
	}
	return out
}

func (t *Telnet) parseMessage(msg string) bool {
	log := log.New()
	msg = t.convertLinks(msg)
	for routeIndex, route := range t.config.Routes {
		if route.Trigger.Custom != "" {
			continue
		}
		pattern, err := regexp.Compile(route.Trigger.Regex)

		if err != nil {
			log.Debug().Err(err).Int("route", routeIndex).Msg("compile")
			continue
		}
		matches := pattern.FindAllStringSubmatch(msg, -1)
		log.Debug().Msgf("regex %s vs %s: %d", route.Trigger.Regex, msg, len(matches))
		if len(matches) == 0 {
			continue
		}

		name := ""
		message := ""
		if route.Trigger.MessageIndex > len(matches[0]) {
			log.Warn().Int("route", routeIndex).Msgf("[telnet] trigger message_index %d greater than matches %d", route.Trigger.MessageIndex, len(matches[0]))
			continue
		}
		message = matches[0][route.Trigger.MessageIndex]
		if route.Trigger.NameIndex > len(matches[0]) {
			log.Warn().Int("route", routeIndex).Msgf("[telnet] name_index %d greater than matches %d", route.Trigger.MessageIndex, len(matches[0]))
			continue
		}
		name = matches[0][route.Trigger.NameIndex]
		if route.Trigger.GuildIndex > 0 && route.Trigger.GuildIndex <= len(matches[0]) {
			route.GuildID = matches[0][route.Trigger.GuildIndex]
			iGuildID, err := strconv.Atoi(route.GuildID)
			if err != nil {
				log.Warn().Int("route", routeIndex).Msgf("[telnet] guild_index %s is not an integer matches %d", route.GuildID, len(matches[0]))
				continue
			}
			tmpChannelID := guilddb.ChannelID(int(iGuildID))
			if tmpChannelID == "" {
				log.Debug().Int("route", routeIndex).Msgf("[telnet] guild_index %d is not in talkeq_guilds, falling back to discord channel %s", iGuildID, route.ChannelID)
			} else {
				route.ChannelID = tmpChannelID
			}
		}

		buf := new(bytes.Buffer)
		if err := route.MessagePatternTemplate().Execute(buf, struct {
			Name    string
			Message string
		}{
			name,
			message,
		}); err != nil {
			log.Warn().Err(err).Int("route", routeIndex).Msg("[discord] execute")
			continue
		}
		switch route.Target {
		case "discord":
			req := request.DiscordSend{
				Ctx:       context.Background(),
				ChannelID: route.ChannelID,
				Message:   buf.String(),
			}
			for _, s := range t.subscribers {
				err = s(req)
				if err != nil {
					log.Warn().Err(err).Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
					continue
				}
				log.Info().Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
			}
		default:
			log.Warn().Msgf("unsupported target type: %s", route.Target)
			continue
		}
	}
	return true
}
