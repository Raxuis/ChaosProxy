package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestRunStopsWhenStopContextEnds(t *testing.T) {
	t.Parallel()

	stopRun, stop := context.WithCancelCause(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- run(serverOptions{
			host:           "127.0.0.1",
			dataHandler:    http.NotFoundHandler(),
			controlHandler: http.NotFoundHandler(),
			logger:         log.New(io.Discard, "", 0),
			stop:           stopRun,
		})
	}()
	time.Sleep(100 * time.Millisecond)
	stop(errMaxRequests)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("run() error = %v, want a clean stop", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not stop after the stop context ended")
	}
	if reason := stopReason(stopRun, nil); reason != "max-requests" {
		t.Fatalf("stopReason() = %q, want max-requests", reason)
	}
}

func TestStopReason(t *testing.T) {
	t.Parallel()

	failed, stopFailed := context.WithCancelCause(context.Background())
	stopFailed(errRequestFailed)
	running, stopRunning := context.WithCancelCause(context.Background())
	defer stopRunning(nil)

	for _, test := range []struct {
		ctx    context.Context
		runErr error
		want   string
	}{
		{ctx: failed, want: "error"},
		{ctx: running, runErr: errors.New("address already in use"), want: "server-error"},
		{ctx: running, want: "signal"},
	} {
		if got := stopReason(test.ctx, test.runErr); got != test.want {
			t.Errorf("stopReason() = %q, want %q", got, test.want)
		}
	}
}
