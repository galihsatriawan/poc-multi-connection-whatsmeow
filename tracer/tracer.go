package tracer

import (
	"context"
	"time"

	uuid "github.com/satori/go.uuid"
)

type Tracer struct {
	name                string
	timeout             *time.Duration
	withDurationLogging bool
}

type TracerCtx struct {
	Id   string
	Name string
}

type tracerCtxKey string

func (t *Tracer) Start(ctx context.Context) (context.Context, *Span) {
	parent, ok := ctx.Value(tracerCtxKey("tracer")).(*TracerCtx)
	name := t.name
	id := uuid.NewV4().String()
	if ok {
		id = parent.Id
		name = parent.Name + "." + name
	}
	newCtx := context.WithValue(ctx, tracerCtxKey("tracer"), &TracerCtx{
		Id:   id,
		Name: name,
	})
	return newCtx, &Span{
		ctx:                 newCtx,
		id:                  id,
		name:                name,
		timeout:             t.timeout,
		withDurationLogging: t.withDurationLogging,
		startTime:           time.Now(),
	}
}

func New(name string, opts ...TracerOption) *Tracer {
	t := &Tracer{
		name: name,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}
