package events

import (
	"sync"
	"sync/atomic"
)

// Subscription contains a point-in-time history followed by buffered live
// events. Close releases it from the bus.
type Subscription struct {
	bus       *Bus
	events    chan Event
	history   []Event
	dropped   atomic.Uint64
	closeOnce sync.Once
}

// History returns the events that preceded this subscription, oldest first.
func (subscription *Subscription) History() []Event {
	history := make([]Event, len(subscription.history))
	for index, event := range subscription.history {
		history[index] = cloneEvent(event)
	}
	return history
}

// Events returns the live event stream.
func (subscription *Subscription) Events() <-chan Event {
	return subscription.events
}

// Dropped reports how many queued live events this subscriber lost.
func (subscription *Subscription) Dropped() uint64 {
	return subscription.dropped.Load()
}

// Close unregisters the subscription and closes its event stream.
func (subscription *Subscription) Close() {
	subscription.closeOnce.Do(func() {
		subscription.bus.unsubscribe(subscription)
	})
}
