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
func (s *Subscription) History() []Event {
	history := make([]Event, len(s.history))
	for index, event := range s.history {
		history[index] = cloneEvent(event)
	}
	return history
}

// Events returns the live event stream.
func (s *Subscription) Events() <-chan Event {
	return s.events
}

// Dropped reports how many queued live events this subscriber lost.
func (s *Subscription) Dropped() uint64 {
	return s.dropped.Load()
}

// Close unregisters the subscription and closes its event stream.
func (s *Subscription) Close() {
	s.closeOnce.Do(func() {
		s.bus.unsubscribe(s)
	})
}
