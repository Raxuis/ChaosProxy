package faults

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

const bandwidthTicksPerSecond = 10

type bandwidthFault struct {
	BaseFault
	bytesPerSecond int64
}

type throttledBody struct {
	io.Closer
	reader         io.Reader
	ctx            context.Context
	bytesPerSecond int64
	started        time.Time
	sent           int64
}

func newBandwidthFault(cfg config.BandwidthConfig) *bandwidthFault {
	return &bandwidthFault{bytesPerSecond: cfg.BytesPerSecond}
}

func (*bandwidthFault) Name() string {
	return "bandwidth"
}

func (f *bandwidthFault) After(ctx *Context, response *http.Response) error {
	response.Body = &throttledBody{
		Closer:         response.Body,
		reader:         response.Body,
		ctx:            ctx.Req.Context(),
		bytesPerSecond: f.bytesPerSecond,
	}
	emit(ctx, Injection{Fault: f.Name()})
	return nil
}

func (b *throttledBody) Read(buffer []byte) (int, error) {
	if b.started.IsZero() {
		b.started = time.Now()
	}
	if chunk := max(b.bytesPerSecond/bandwidthTicksPerSecond, 1); int64(len(buffer)) > chunk {
		buffer = buffer[:chunk]
	}
	read, err := b.reader.Read(buffer)
	b.sent += int64(read)
	due := b.started.Add(time.Duration(float64(b.sent) * float64(time.Second) / float64(b.bytesPerSecond)))
	if waitErr := wait(b.ctx, time.Until(due)); waitErr != nil {
		return read, waitErr
	}
	return read, err
}
