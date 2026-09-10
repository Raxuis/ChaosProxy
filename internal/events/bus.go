package events

import (
	"sync"
	"time"
)

const (
	historyCapacity  = 500
	subscriberBuffer = 128
)

// Stats describes the current in-memory event bus state.
type Stats struct {
	Published   uint64 `json:"published"`
	Dropped     uint64 `json:"dropped"`
	Subscribers int    `json:"subscribers"`
	History     int    `json:"history"`
}

// Bus retains recent events and fans out new events without waiting for
// subscribers to consume them.
type Bus struct {
	mu          sync.Mutex
	history     [historyCapacity]Event
	historyNext int
	historySize int
	nextID      uint64
	published   uint64
	dropped     uint64
	subscribers map[*Subscription]struct{}
}

// NewBus creates an empty event bus.
func NewBus() *Bus {
	return &Bus{subscribers: make(map[*Subscription]struct{})}
}

// Publish records and broadcasts event. A full subscriber buffer loses its
// oldest queued event instead of delaying the publisher.
func (bus *Bus) Publish(event Event) {
	event.Faults = append([]string(nil), event.Faults...)

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	bus.nextID++
	event.ID = bus.nextID
	bus.published++
	bus.appendHistory(event)
	for subscription := range bus.subscribers {
		bus.publishTo(subscription, event)
	}
}

// Subscribe atomically captures ordered history and starts a buffered live
// subscription, so events cannot be lost between those two operations.
func (bus *Bus) Subscribe() *Subscription {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	subscription := &Subscription{
		bus:     bus,
		events:  make(chan Event, subscriberBuffer),
		history: bus.historySnapshot(),
	}
	bus.subscribers[subscription] = struct{}{}
	return subscription
}

// Stats returns a consistent snapshot of counters and buffer sizes.
func (bus *Bus) Stats() Stats {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	return Stats{
		Published:   bus.published,
		Dropped:     bus.dropped,
		Subscribers: len(bus.subscribers),
		History:     bus.historySize,
	}
}

// Reset clears history and counters while keeping live subscriptions open.
func (bus *Bus) Reset() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.history = [historyCapacity]Event{}
	bus.historyNext = 0
	bus.historySize = 0
	bus.nextID = 0
	bus.published = 0
	bus.dropped = 0
	for subscription := range bus.subscribers {
		drain(subscription.events)
		subscription.dropped.Store(0)
	}
}

func (bus *Bus) appendHistory(event Event) {
	bus.history[bus.historyNext] = event
	bus.historyNext = (bus.historyNext + 1) % historyCapacity
	if bus.historySize < historyCapacity {
		bus.historySize++
	}
}

func (bus *Bus) historySnapshot() []Event {
	history := make([]Event, bus.historySize)
	start := (bus.historyNext - bus.historySize + historyCapacity) % historyCapacity
	for index := range history {
		history[index] = cloneEvent(bus.history[(start+index)%historyCapacity])
	}
	return history
}

func (bus *Bus) publishTo(subscription *Subscription, event Event) {
	select {
	case subscription.events <- event:
		return
	default:
	}

	dropped := false
	select {
	case <-subscription.events:
		dropped = true
	default:
	}
	select {
	case subscription.events <- event:
	default:
		dropped = true
	}
	if dropped {
		subscription.dropped.Add(1)
		bus.dropped++
	}
}

func (bus *Bus) unsubscribe(subscription *Subscription) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	if _, exists := bus.subscribers[subscription]; !exists {
		return
	}
	delete(bus.subscribers, subscription)
	close(subscription.events)
}

func drain(events chan Event) {
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}

func cloneEvent(event Event) Event {
	event.Faults = append([]string(nil), event.Faults...)
	return event
}
