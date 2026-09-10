package events_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/events"
)

func TestBusRetainsLastFiveHundredEventsInOrder(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	for index := 1; index <= 600; index++ {
		bus.Publish(events.Event{Path: fmt.Sprintf("/%d", index)})
	}

	subscription := bus.Subscribe()
	defer subscription.Close()
	history := subscription.History()
	if len(history) != 500 {
		t.Fatalf("history length = %d, want 500", len(history))
	}
	for index, event := range history {
		wantID := uint64(index + 101)
		if event.ID != wantID || event.Path != fmt.Sprintf("/%d", wantID) {
			t.Fatalf("history[%d] = ID %d path %q, want ID %d", index, event.ID, event.Path, wantID)
		}
	}
}

func TestBusDropsOldestEventForSlowSubscriber(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	subscription := bus.Subscribe()
	defer subscription.Close()

	done := make(chan struct{})
	go func() {
		for index := 0; index < 129; index++ {
			bus.Publish(events.Event{Path: fmt.Sprintf("/%d", index+1)})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}

	if got := subscription.Dropped(); got != 1 {
		t.Fatalf("subscription drops = %d, want 1", got)
	}
	if got := bus.Stats().Dropped; got != 1 {
		t.Fatalf("bus drops = %d, want 1", got)
	}
	first := <-subscription.Events()
	if first.ID != 2 {
		t.Fatalf("oldest buffered event ID = %d, want 2", first.ID)
	}
}

func TestSubscriptionCloseRemovesSubscriber(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	subscription := bus.Subscribe()
	if got := bus.Stats().Subscribers; got != 1 {
		t.Fatalf("subscriber count = %d, want 1", got)
	}
	subscription.Close()
	subscription.Close()
	if got := bus.Stats().Subscribers; got != 0 {
		t.Fatalf("subscriber count = %d, want 0", got)
	}
	if _, open := <-subscription.Events(); open {
		t.Fatal("subscription event channel remained open")
	}
}

func TestBusResetClearsRuntimeState(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	subscription := bus.Subscribe()
	defer subscription.Close()
	for index := 0; index < 130; index++ {
		bus.Publish(events.Event{Path: "/api"})
	}
	bus.Reset()

	if got := bus.Stats(); got.Published != 0 || got.Dropped != 0 || got.History != 0 || got.Subscribers != 1 {
		t.Fatalf("stats after reset = %+v", got)
	}
	if got := subscription.Dropped(); got != 0 {
		t.Fatalf("subscription drops after reset = %d, want 0", got)
	}
	select {
	case event := <-subscription.Events():
		t.Fatalf("queued event remained after reset: %+v", event)
	default:
	}

	bus.Publish(events.Event{Path: "/after-reset"})
	if got := (<-subscription.Events()).ID; got != 1 {
		t.Fatalf("event ID after reset = %d, want 1", got)
	}
}

func TestBusOwnsEventSlices(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	faults := []string{"status"}
	bus.Publish(events.Event{Faults: faults})
	faults[0] = "mutated"
	subscription := bus.Subscribe()
	defer subscription.Close()

	if got := subscription.History()[0].Faults[0]; got != "status" {
		t.Fatalf("stored fault = %q, want status", got)
	}
}
