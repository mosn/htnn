package translation

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
)

type Ctx struct {
	context.Context

	logger *logr.Logger
}

func (c *Ctx) Logger() *logr.Logger {
	return c.logger
}

type Info struct {
	// HTTPFilterPolicies indicates what HTTPFilterPolicies are used to generated the EnvoyFilter.
	HTTPFilterPolicies []string `json:"httpfilterpolicies"`
}

func (info *Info) String() string {
	b, _ := json.Marshal(info)
	return string(b)
}
