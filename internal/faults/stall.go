package faults

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type stallFault struct {
	BaseFault
	probability float64
	afterBytes  int64
	duration    time.Duration
}

type stalledBody struct {
	io.Closer
	reader     io.Reader
	ctx        context.Context
	cancel     context.CancelFunc
	afterBytes int64
	duration   time.Duration
	sent       int64
	stalled    bool
}

func newStallFault(cfg config.StallConfig) *stallFault {
	return &stallFault{
		probability: cfg.Probability,
		afterBytes:  cfg.AfterBytes,
		duration:    cfg.Duration,
	}
}

func (*stallFault) Name() string {
	return "stall"
}

func (f *stallFault) After(ctx *Context, response *http.Response) error {
	if !shouldTrigger(ctx.Rng, f.probability) {
		return nil
	}

	stallCtx, cancel := context.WithCancel(ctx.Req.Context())
	response.Body = &stalledBody{
		Closer:     response.Body,
		reader:     response.Body,
		ctx:        stallCtx,
		cancel:     cancel,
		afterBytes: f.afterBytes,
		duration:   f.duration,
	}
	emit(ctx, Injection{Fault: f.Name()})
	return nil
}

func (b *stalledBody) Close() error {
	b.cancel()
	return b.Closer.Close()
}

func (b *stalledBody) Read(buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}

	if !b.stalled && b.sent >= b.afterBytes {
		b.stalled = true
		if err := wait(b.ctx, b.duration); err != nil {
			return 0, err
		}
	}

	readBuffer := buffer
	if !b.stalled && b.afterBytes > 0 {
		remaining := b.afterBytes - b.sent
		if remaining > 0 && int64(len(readBuffer)) > remaining {
			readBuffer = readBuffer[:remaining]
		}
	}

	read, err := b.reader.Read(readBuffer)
	b.sent += int64(read)
	return read, err
}
