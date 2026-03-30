package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSlogLogger(t *testing.T) {
	log := NewSlogLogger(slog.LevelDebug)
	assert.NotNil(t, log)
}
