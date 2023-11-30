package translation

import (
	"context"

	"github.com/go-logr/logr"
)

type Ctx struct {
	context.Context

	logger *logr.Logger
}

func (c *Ctx) Logger() *logr.Logger {
	return c.logger
}
