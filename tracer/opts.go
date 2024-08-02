package tracer

import "time"

type TracerOption func(t *Tracer)

func WithTimeout(d time.Duration) TracerOption {
	return func(t *Tracer) {
		t.timeout = &d
	}
}

func WithDurationLogging() TracerOption {
	return func(t *Tracer) {
		t.withDurationLogging = true
	}
}
