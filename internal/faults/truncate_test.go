package faults

import (
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type trackingReadCloser struct {
	reader *strings.Reader
	reads  int
	closed bool
}

func newTrackingReadCloser(body string) *trackingReadCloser {
	return &trackingReadCloser{reader: strings.NewReader(body)}
}

func (body *trackingReadCloser) Read(buffer []byte) (int, error) {
	body.reads++
	return body.reader.Read(buffer)
}

func (body *trackingReadCloser) Close() error {
	body.closed = true
	return nil
}

func TestTruncateAfterWrapsBodyWithoutBuffering(t *testing.T) {
	fault, err := newTruncateFault(config.TruncateConfig{Probability: 1, At: 0.5})
	if err != nil {
		t.Fatalf("newTruncateFault() unexpected error: %v", err)
	}

	body := newTrackingReadCloser("0123456789")
	response := &http.Response{
		Body:          body,
		ContentLength: 10,
		Header:        http.Header{"Content-Length": []string{"10"}},
	}
	if err := fault.After(newFaultContext(), response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	if body.reads != 0 {
		t.Fatalf("After() read the body %d times, want zero", body.reads)
	}
	if response.ContentLength != -1 || response.Header.Get("Content-Length") != "" {
		t.Errorf("Content-Length = (%d, %q), want removed", response.ContentLength, response.Header.Get("Content-Length"))
	}

	got, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read wrapped body: %v", err)
	}
	if string(got) != "01234" {
		t.Errorf("truncated body = %q, want first half", got)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close wrapped body: %v", err)
	}
	if !body.closed {
		t.Error("closing wrapper did not close upstream body")
	}
}

func TestTruncateLeavesUnknownLengthStreaming(t *testing.T) {
	fault, err := newTruncateFault(config.TruncateConfig{Probability: 1, At: 0.5})
	if err != nil {
		t.Fatalf("newTruncateFault() unexpected error: %v", err)
	}
	body := newTrackingReadCloser("stream")
	response := &http.Response{Body: body, ContentLength: -1, Header: make(http.Header)}

	if err := fault.After(newFaultContext(), response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	if response.Body != body || body.reads != 0 {
		t.Fatal("unknown-length stream should remain untouched")
	}
}

func TestTruncateProbabilityZeroDoesNotModifyResponse(t *testing.T) {
	fault, err := newTruncateFault(config.TruncateConfig{Probability: 0, At: 0.5})
	if err != nil {
		t.Fatalf("newTruncateFault() unexpected error: %v", err)
	}
	body := newTrackingReadCloser("body")
	response := &http.Response{Body: body, ContentLength: 4, Header: http.Header{"Content-Length": []string{"4"}}}

	if err := fault.After(newFaultContext(), response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	if response.Body != body || response.ContentLength != 4 {
		t.Fatal("non-triggered truncate fault modified response")
	}
}

func TestTruncateConfigurationErrors(t *testing.T) {
	tests := []config.TruncateConfig{
		{Probability: -1, At: 0.5},
		{Probability: 2, At: 0.5},
		{Probability: 1, At: -0.1},
		{Probability: 1, At: 1.1},
		{Probability: 1, At: math.NaN()},
	}
	for _, configured := range tests {
		if _, err := newTruncateFault(configured); err == nil {
			t.Errorf("newTruncateFault(%+v) error = nil", configured)
		}
	}
}
