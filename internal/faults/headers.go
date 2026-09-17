package faults

import (
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type headersFault struct {
	BaseFault
	probability float64
	set         map[string]string
	remove      []string
}

func newHeadersFault(cfg config.HeadersConfig) *headersFault {
	return &headersFault{
		probability: cfg.Probability,
		set:         cfg.Set,
		remove:      cfg.Remove,
	}
}

func (*headersFault) Name() string {
	return "headers"
}

func (f *headersFault) After(ctx *Context, response *http.Response) error {
	if !shouldTrigger(ctx.Rng, f.probability) {
		return nil
	}
	if response.Header == nil {
		response.Header = make(http.Header)
	}
	for _, key := range f.remove {
		response.Header.Del(key)
	}
	for key, value := range f.set {
		response.Header.Set(key, value)
	}

	var changes []string
	for _, key := range slices.Sorted(maps.Keys(f.set)) {
		changes = append(changes, "set "+key)
	}
	for _, key := range f.remove {
		changes = append(changes, "removed "+key)
	}

	emit(ctx, Injection{
		Fault:  f.Name(),
		Detail: strings.Join(changes, ", "),
	})
	return nil
}
