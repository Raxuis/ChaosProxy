package faults

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/Raxuis/chaosproxy/internal/config"
)

const maxBufferedTruncation = 1 << 20

var errTruncated = errors.New("response truncated by chaosproxy")

type truncateFault struct {
	BaseFault
	probability float64
	at          float64
}

// A read error makes httputil.ReverseProxy abort the client connection, which is
// what a real network truncation looks like.
type truncatedBody struct {
	io.Closer
	reader    io.Reader
	remaining int64
}

func newTruncateFault(cfg config.TruncateConfig) *truncateFault {
	return &truncateFault{probability: cfg.Probability, at: cfg.At}
}

func (*truncateFault) Name() string {
	return "truncate"
}

func (f *truncateFault) After(ctx *Context, response *http.Response) error {
	if !shouldTrigger(ctx.Rng, f.probability) || f.at >= 1 {
		return nil
	}

	length := response.ContentLength
	var reader io.Reader = response.Body
	if length < 0 {
		buffered, err := io.ReadAll(io.LimitReader(response.Body, maxBufferedTruncation))
		if err != nil {
			return fmt.Errorf("buffer response of unknown length: %w", err)
		}
		length = int64(len(buffered))
		reader = io.MultiReader(bytes.NewReader(buffered), response.Body)
	}

	response.Body = &truncatedBody{
		Closer:    response.Body,
		reader:    reader,
		remaining: int64(math.Floor(float64(length) * f.at)),
	}
	emit(ctx, Injection{Fault: f.Name()})
	return nil
}

func (b *truncatedBody) Read(buffer []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, errTruncated
	}
	if int64(len(buffer)) > b.remaining {
		buffer = buffer[:b.remaining]
	}
	read, err := b.reader.Read(buffer)
	b.remaining -= int64(read)
	return read, err
}
