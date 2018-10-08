package model

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// NewLogger returns a new logger
func NewLogger(ctx context.Context) (logger zerolog.Logger) {
	logger = log.With().Logger()
	return
}
