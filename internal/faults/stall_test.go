package faults

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestStallProbabilityZeroDoesNotTrigger(t *testing.T) {
	fault := newStallFault(config.StallConfig{
		Probability: 0,
		AfterBytes:  1024,
		Duration:    time.Second,
	})

	originalBody := io.NopCloser(bytes.NewReader([]byte("payload")))
	response := &http.Response{Body: originalBody}
	ctx := newFaultContext()

	if err := fault.After(ctx, response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	if response.Body != originalBody {
		t.Error("response.Body was wrapped when probability was 0")
	}
}

func TestStallReadsBytesBeforeAndAfterPause(t *testing.T) {
	fault := newStallFault(config.StallConfig{
		Probability: 1,
		AfterBytes:  2048,
		Duration:    60 * time.Millisecond,
	})

	payload := bytes.Repeat([]byte("x"), 4096)
	response := &http.Response{
		Body:          io.NopCloser(bytes.NewReader(payload)),
		ContentLength: int64(len(payload)),
	}

	req := httptest.NewRequest(http.MethodGet, "http://test.local", nil)
	var emitted Injection
	ctx := newFaultContext()
	ctx.Req = req
	ctx.Emit = func(inj Injection) {
		emitted = inj
	}

	if err := fault.After(ctx, response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	if emitted.Fault != "stall" {
		t.Errorf("emitted fault = %q, want stall", emitted.Fault)
	}
	if response.ContentLength != int64(len(payload)) {
		t.Errorf("ContentLength = %d, want %d", response.ContentLength, len(payload))
	}

	buf := make([]byte, 1024)
	start := time.Now()

	n1, err1 := io.ReadFull(response.Body, buf)
	if err1 != nil || n1 != 1024 {
		t.Fatalf("read 1 = %d, %v; want 1024, nil", n1, err1)
	}
	n2, err2 := io.ReadFull(response.Body, buf)
	if err2 != nil || n2 != 1024 {
		t.Fatalf("read 2 = %d, %v; want 1024, nil", n2, err2)
	}
	timeBeforeStall := time.Since(start)
	if timeBeforeStall >= 50*time.Millisecond {
		t.Fatalf("first 2048 bytes took %v, want < 50ms before stall", timeBeforeStall)
	}

	stallStart := time.Now()
	allRemaining, err3 := io.ReadAll(response.Body)
	if err3 != nil {
		t.Fatalf("read remaining: %v", err3)
	}
	stallDuration := time.Since(stallStart)
	if stallDuration < 50*time.Millisecond {
		t.Errorf("stall duration = %v, want >= 50ms", stallDuration)
	}
	if len(allRemaining) != 2048 {
		t.Errorf("remaining bytes = %d, want 2048", len(allRemaining))
	}
}

func TestStallStopsOnContextCancellation(t *testing.T) {
	fault := newStallFault(config.StallConfig{
		Probability: 1,
		AfterBytes:  100,
		Duration:    5 * time.Second,
	})

	payload := bytes.Repeat([]byte("y"), 1000)
	response := &http.Response{
		Body: io.NopCloser(bytes.NewReader(payload)),
	}

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "http://test.local", nil).WithContext(reqCtx)
	ctx := newFaultContext()
	ctx.Req = req

	if err := fault.After(ctx, response); err != nil {
		t.Fatalf("After(): %v", err)
	}

	buf := make([]byte, 100)
	n, err := io.ReadFull(response.Body, buf)
	if err != nil || n != 100 {
		t.Fatalf("read 100 = %d, %v", n, err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err = response.Body.Read(buf)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Read() error = %v, want context.Canceled", err)
	}
	if elapsed >= time.Second {
		t.Errorf("Read() took %v, want < 1s abort on cancel", elapsed)
	}
}

func TestStallStopsOnBodyClose(t *testing.T) {
	fault := newStallFault(config.StallConfig{
		Probability: 1,
		AfterBytes:  50,
		Duration:    5 * time.Second,
	})

	payload := bytes.Repeat([]byte("z"), 500)
	response := &http.Response{
		Body: io.NopCloser(bytes.NewReader(payload)),
	}

	req := httptest.NewRequest(http.MethodGet, "http://test.local", nil)
	ctx := newFaultContext()
	ctx.Req = req

	if err := fault.After(ctx, response); err != nil {
		t.Fatalf("After(): %v", err)
	}

	buf := make([]byte, 50)
	n, err := io.ReadFull(response.Body, buf)
	if err != nil || n != 50 {
		t.Fatalf("read 50 = %d, %v", n, err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = response.Body.Close()
	}()

	start := time.Now()
	_, err = response.Body.Read(buf)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Read() error = %v, want context.Canceled", err)
	}
	if elapsed >= time.Second {
		t.Errorf("Read() took %v, want < 1s abort on Close()", elapsed)
	}
}
