package tracer

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Span struct {
	ctx                 context.Context
	id                  string
	name                string
	startTime           time.Time
	timeout             *time.Duration
	withDurationLogging bool
	attr                map[string]interface{}
}

func (s *Span) SetAttributes(attributes map[string]interface{}) {
	s.attr = attributes
}

func (s *Span) End() {
	elapsed := time.Since(s.startTime)

	var (
		logEvent *zerolog.Event
		msg      string
	)
	if s.timeout != nil && elapsed.Nanoseconds() > s.timeout.Nanoseconds() {
		logEvent = log.Warn()
		msg = fmt.Sprintf("[%s] timeout occurs, elapsed %v", s.name, elapsed)
	} else if s.withDurationLogging {
		logEvent = log.Info()
		msg = fmt.Sprintf("[%s] process elapsed %v", s.name, elapsed)
	} else {
		return
	}
	logEvent.Str("id", s.id)
	for key, at := range s.attr {
		if at == nil {
			continue
		}
		logEvent.Interface(key, at)
	}
	logEvent.Msg(msg)
}
