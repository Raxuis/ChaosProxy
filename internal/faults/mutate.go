package faults

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type mutateFault struct {
	BaseFault
	probability float64
	maxBytes    int64
	operations  []mutation
}

type mutation struct {
	op       string
	path     string
	segments []string
	factor   int
}

type replayBody struct {
	io.Reader
	io.Closer
}

func newMutateFault(cfg config.MutateConfig) *mutateFault {
	fault := &mutateFault{probability: cfg.Probability, maxBytes: cfg.MaxBytes}
	for _, operation := range cfg.Operations {
		fault.operations = append(fault.operations, mutation{
			op:       operation.Op,
			path:     operation.Path,
			segments: strings.Split(operation.Path, "."),
			factor:   operation.Factor,
		})
	}
	return fault
}

func (*mutateFault) Name() string {
	return "mutate"
}

func (f *mutateFault) After(ctx *Context, response *http.Response) error {
	if !shouldTrigger(ctx.Rng, f.probability) || !isJSON(response.Header.Get("Content-Type")) {
		return nil
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		note(ctx, "mutate skipped: %s-encoded body", encoding)
		return nil
	}
	if response.ContentLength > f.maxBytes {
		note(ctx, "mutate skipped: body of %d bytes exceeds max_bytes %d", response.ContentLength, f.maxBytes)
		return nil
	}

	original, err := io.ReadAll(io.LimitReader(response.Body, f.maxBytes+1))
	if err != nil {
		return fmt.Errorf("read JSON body: %w", err)
	}
	if int64(len(original)) > f.maxBytes {
		response.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(original), response.Body), Closer: response.Body}
		note(ctx, "mutate skipped: body exceeds max_bytes %d", f.maxBytes)
		return nil
	}
	_ = response.Body.Close()

	body, notes := f.mutate(original)
	for _, detail := range notes {
		note(ctx, "%s", detail)
	}
	if body == nil {
		body = original
	} else {
		response.Header.Del("ETag")
		emit(ctx, Injection{Fault: f.Name()})
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}

func (f *mutateFault) mutate(original []byte) ([]byte, []string) {
	decoder := json.NewDecoder(bytes.NewReader(original))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, []string{"mutate skipped: response is not valid JSON"}
	}

	var notes []string
	changed := 0
	for _, operation := range f.operations {
		updated, count := operation.apply(document, operation.segments)
		document = updated
		changed += count
		if count == 0 {
			notes = append(notes, fmt.Sprintf("mutate %s %s matched nothing", operation.op, operation.path))
		}
	}
	if changed == 0 {
		return nil, notes
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(document); err != nil {
		return nil, append(notes, "mutate skipped: "+err.Error())
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), notes
}

func (m mutation) apply(node any, segments []string) (any, int) {
	segment, rest := segments[0], segments[1:]
	switch container := node.(type) {
	case map[string]any:
		keys := []string{segment}
		if segment == "*" {
			keys = slices.Sorted(maps.Keys(container))
		}
		changed := 0
		for _, key := range keys {
			child, found := container[key]
			switch {
			case !found:
			case len(rest) > 0:
				updated, count := m.apply(child, rest)
				container[key] = updated
				changed += count
			case m.op == "drop":
				delete(container, key)
				changed++
			default:
				if updated, ok := m.transform(child); ok {
					container[key] = updated
					changed++
				}
			}
		}
		return container, changed
	case []any:
		changed := 0
		dropped := make(map[int]bool)
		for _, index := range arrayIndexes(segment, len(container)) {
			switch {
			case len(rest) > 0:
				updated, count := m.apply(container[index], rest)
				container[index] = updated
				changed += count
			case m.op == "drop":
				dropped[index] = true
				changed++
			default:
				if updated, ok := m.transform(container[index]); ok {
					container[index] = updated
					changed++
				}
			}
		}
		if len(dropped) == 0 {
			return container, changed
		}
		kept := make([]any, 0, len(container)-len(dropped))
		for index, item := range container {
			if !dropped[index] {
				kept = append(kept, item)
			}
		}
		return kept, changed
	default:
		return node, 0
	}
}

func (m mutation) transform(value any) (any, bool) {
	switch m.op {
	case "nullify":
		return nil, value != nil
	case "empty":
		switch value.(type) {
		case string:
			return "", true
		case []any:
			return []any{}, true
		case map[string]any:
			return map[string]any{}, true
		case json.Number:
			return json.Number("0"), true
		case bool:
			return false, true
		}
	case "inflate":
		if items, ok := value.([]any); ok && len(items) > 0 {
			inflated := make([]any, 0, len(items)*m.factor)
			for range m.factor {
				inflated = append(inflated, items...)
			}
			return inflated, true
		}
	case "stretch":
		if text, ok := value.(string); ok && text != "" {
			return strings.Repeat(text, m.factor), true
		}
	}
	return value, false
}

func arrayIndexes(segment string, length int) []int {
	if segment == "*" {
		indexes := make([]int, length)
		for index := range indexes {
			indexes[index] = index
		}
		return indexes
	}
	index, err := strconv.Atoi(segment)
	if err != nil || index < 0 || index >= length {
		return nil
	}
	return []int{index}
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && (mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"))
}
