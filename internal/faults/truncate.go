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

func newTruncateFault(configured config.TruncateConfig) (*truncateFault, error) {
	if err := validateProbability(configured.Probability); err != nil {
		return nil, err
	}
	if math.IsNaN(configured.At) || configured.At < 0 || configured.At > 1 {
		return nil, errors.New("at must be between 0 and 1")
	}
	return &truncateFault{probability: configured.Probability, at: configured.At}, nil
}

func (*truncateFault) Name() string {
	return "truncate"
}

func (fault *truncateFault) After(ctx *Context, response *http.Response) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if response == nil || response.Body == nil {
		return errors.New("upstream response and body must not be nil")
	}

	triggered, err := shouldTrigger(ctx.Rng, fault.probability)
	if err != nil || !triggered || fault.at >= 1 {
		return err
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
		remaining: int64(math.Floor(float64(length) * fault.at)),
	}
	emit(ctx, Event{Faults: []string{fault.Name()}})
	return nil
}

func (body *truncatedBody) Read(buffer []byte) (int, error) {
	if body.remaining <= 0 {
		return 0, errTruncated
	}
	if int64(len(buffer)) > body.remaining {
		buffer = buffer[:body.remaining]
	}
	read, err := body.reader.Read(buffer)
	body.remaining -= int64(read)
	return read, err
}
