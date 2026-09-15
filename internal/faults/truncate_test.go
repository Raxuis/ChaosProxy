package faults

import (
	"errors"
	"io"
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

func TestTruncateKeepsDeclaredLengthAndFailsAfterCut(t *testing.T) {
	fault := newTestTruncateFault(0.5)
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
		t.Fatalf("After() read a body of known length %d times, want zero", body.reads)
	}
	if response.ContentLength != 10 || response.Header.Get("Content-Length") != "10" {
		t.Errorf("Content-Length = (%d, %q), want declared length kept", response.ContentLength, response.Header.Get("Content-Length"))
	}

	got, err := io.ReadAll(response.Body)
	if !errors.Is(err, errTruncated) {
		t.Fatalf("read error = %v, want errTruncated", err)
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

func TestTruncateBuffersUnknownLength(t *testing.T) {
	fault := newTestTruncateFault(0.5)
	response := &http.Response{Body: newTrackingReadCloser("0123456789"), ContentLength: -1, Header: make(http.Header)}

	if err := fault.After(newFaultContext(), response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	got, err := io.ReadAll(response.Body)
	if !errors.Is(err, errTruncated) {
		t.Fatalf("read error = %v, want errTruncated", err)
	}
	if string(got) != "01234" {
		t.Errorf("truncated body = %q, want first half", got)
	}
}

func TestTruncateCapsBufferingOfUnknownLength(t *testing.T) {
	fault := newTestTruncateFault(0.25)
	body := newTrackingReadCloser(strings.Repeat("x", 3*maxBufferedTruncation))
	response := &http.Response{Body: body, ContentLength: -1, Header: make(http.Header)}

	if err := fault.After(newFaultContext(), response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	got, err := io.ReadAll(response.Body)
	if !errors.Is(err, errTruncated) {
		t.Fatalf("read error = %v, want errTruncated", err)
	}
	if len(got) != maxBufferedTruncation/4 {
		t.Errorf("truncated length = %d, want %d", len(got), maxBufferedTruncation/4)
	}
}

func TestTruncateLeavesResponseUntouched(t *testing.T) {
	tests := map[string]config.TruncateConfig{
		"probability zero": {Probability: 0, At: 0.5},
		"at one":           {Probability: 1, At: 1},
	}
	for name, configured := range tests {
		fault := newTruncateFault(configured)
		body := newTrackingReadCloser("body")
		response := &http.Response{Body: body, ContentLength: -1, Header: make(http.Header)}

		if err := fault.After(newFaultContext(), response); err != nil {
			t.Fatalf("%s: After() unexpected error: %v", name, err)
		}
		if response.Body != body || body.reads != 0 {
			t.Errorf("%s: response was modified", name)
		}
	}
}

func newTestTruncateFault(at float64) *truncateFault {
	return newTruncateFault(config.TruncateConfig{Probability: 1, At: at})
}
