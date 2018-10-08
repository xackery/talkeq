package manager

import (
	"context"

	"github.com/xackery/talkeq/model"
)

// ChatMessage consumes endpoint chat messages
func (m *Manager) ChatMessage(ctx context.Context, req *model.ChatMessage) (err error) {
	_, err = m.runQuery(ctx, "ChatMessage", req)
	return
}

func (m *Manager) onChatMessage(ctx context.Context, req *model.ChatMessage) (err error) {
	logger := model.NewLogger(ctx)
	routes := m.config.Routes[req.SourceEndpoint]

	logger.Debug().Str("source", req.SourceEndpoint).Strs("dest", routes).Msgf("%s: %s", req.Author, req.Message)
	for _, route := range routes {
		req.DestinationEndpoint = route
		_, err = m.runClientQuery(ctx, "ChatMessage", req)
		if err != nil {
			logger.Error().Str("source", req.SourceEndpoint).Str("dest", route).Err(err).Msgf("failed msg %s: %s", req.Author, req.Message)
			continue
		}
		logger.Debug().Str("source", req.SourceEndpoint).Str("dest", route).Msgf("%s: %s", req.Author, req.Message)
	}
	return
}
