package faults

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestBandwidthThrottlesReadsWithoutBuffering(t *testing.T) {
	body := newTrackingReadCloser(strings.Repeat("x", 2000))
	response := &http.Response{Body: body, ContentLength: 2000, Header: make(http.Header)}
	if err := newBandwidthFault(config.BandwidthConfig{BytesPerSecond: 10_000}).After(newFaultContext(), response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	if body.reads != 0 || response.ContentLength != 2000 {
		t.Fatalf("After() reads/ContentLength = %d/%d, want 0/2000", body.reads, response.ContentLength)
	}

	started := time.Now()
	got, err := io.ReadAll(response.Body)
	elapsed := time.Since(started)
	if err != nil || len(got) != 2000 {
		t.Fatalf("ReadAll() = %d bytes, %v; want 2000 bytes", len(got), err)
	}
	if elapsed < 150*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("2000 bytes at 10000 B/s took %s, want about 200ms", elapsed)
	}
	if body.reads < 2 {
		t.Errorf("upstream reads = %d, want chunked reads", body.reads)
	}
	if err := response.Body.Close(); err != nil || !body.closed {
		t.Errorf("Close() = %v, closed = %t; want upstream body closed", err, body.closed)
	}
}

func TestBandwidthStopsWhenRequestIsCanceled(t *testing.T) {
	faultContext := newFaultContext()
	canceled, cancel := context.WithCancel(faultContext.Req.Context())
	defer cancel()
	faultContext.Req = faultContext.Req.WithContext(canceled)

	response := &http.Response{Body: newTrackingReadCloser("0123456789"), Header: make(http.Header)}
	if err := newBandwidthFault(config.BandwidthConfig{BytesPerSecond: 1}).After(faultContext, response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(response.Body)
		result <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ReadAll() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("throttled read did not stop after cancellation")
	}
}
