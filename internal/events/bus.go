package events

import (
	"maps"
	"strconv"
	"sync"
	"time"
)

const (
	historyCapacity  = 500
	subscriberBuffer = 128
)

// Stats describes the current in-memory event bus state and request totals.
type Stats struct {
	Published   uint64               `json:"published"`
	Faulted     uint64               `json:"faulted"`
	Dropped     uint64               `json:"dropped"`
	Subscribers int                  `json:"subscribers"`
	History     int                  `json:"history"`
	Statuses    map[string]uint64    `json:"statuses"`
	Rules       map[string]RuleStats `json:"rules"`
}

// RuleStats counts the requests one rule matched and the faults it injected.
type RuleStats struct {
	Matched uint64            `json:"matched"`
	Faulted uint64            `json:"faulted"`
	Faults  map[string]uint64 `json:"faults"`
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
	faulted     uint64
	dropped     uint64
	statuses    map[string]uint64
	rules       map[string]*RuleStats
	subscribers map[*Subscription]struct{}
}

// NewBus creates an empty event bus.
func NewBus() *Bus {
	return &Bus{
		statuses:    make(map[string]uint64),
		rules:       make(map[string]*RuleStats),
		subscribers: make(map[*Subscription]struct{}),
	}
}

// Publish records and broadcasts event. A full subscriber buffer loses its
// oldest queued event instead of delaying the publisher.
func (b *Bus) Publish(event Event) {
	event.Faults = append([]string(nil), event.Faults...)
	event.Details = append([]string(nil), event.Details...)

	b.mu.Lock()
	defer b.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	b.nextID++
	event.ID = b.nextID
	b.published++
	b.count(event)
	b.appendHistory(event)
	for subscription := range b.subscribers {
		b.publishTo(subscription, event)
	}
}

// Subscribe atomically captures ordered history and starts a buffered live
// subscription, so events cannot be lost between those two operations.
func (b *Bus) Subscribe() *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscription := &Subscription{
		bus:     b,
		events:  make(chan Event, subscriberBuffer),
		history: b.historySnapshot(),
	}
	b.subscribers[subscription] = struct{}{}
	return subscription
}

// Stats returns a consistent snapshot of counters and buffer sizes.
func (b *Bus) Stats() Stats {
	b.mu.Lock()
	defer b.mu.Unlock()
	rules := make(map[string]RuleStats, len(b.rules))
	for name, rule := range b.rules {
		rules[name] = RuleStats{Matched: rule.Matched, Faulted: rule.Faulted, Faults: maps.Clone(rule.Faults)}
	}
	return Stats{
		Published:   b.published,
		Faulted:     b.faulted,
		Dropped:     b.dropped,
		Subscribers: len(b.subscribers),
		History:     b.historySize,
		Statuses:    maps.Clone(b.statuses),
		Rules:       rules,
	}
}

// Reset clears history and counters while keeping live subscriptions open.
func (b *Bus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.history = [historyCapacity]Event{}
	b.historyNext = 0
	b.historySize = 0
	b.nextID = 0
	b.published = 0
	b.faulted = 0
	b.dropped = 0
	b.statuses = make(map[string]uint64)
	b.rules = make(map[string]*RuleStats)
	for subscription := range b.subscribers {
		drain(subscription.events)
		subscription.dropped.Store(0)
	}
}

func (b *Bus) count(event Event) {
	b.statuses[statusClass(event.Status)]++
	if len(event.Faults) > 0 {
		b.faulted++
	}
	if event.Rule == "" {
		return
	}

	rule, found := b.rules[event.Rule]
	if !found {
		rule = &RuleStats{Faults: make(map[string]uint64)}
		b.rules[event.Rule] = rule
	}
	rule.Matched++
	if len(event.Faults) > 0 {
		rule.Faulted++
	}
	for _, fault := range event.Faults {
		rule.Faults[fault]++
	}
}

// Status 0 means the response never started, for example a hang the client abandoned.
func statusClass(status int) string {
	if status < 100 || status > 599 {
		return "aborted"
	}
	return strconv.Itoa(status/100) + "xx"
}

func (b *Bus) appendHistory(event Event) {
	b.history[b.historyNext] = event
	b.historyNext = (b.historyNext + 1) % historyCapacity
	if b.historySize < historyCapacity {
		b.historySize++
	}
}

func (b *Bus) historySnapshot() []Event {
	history := make([]Event, b.historySize)
	start := (b.historyNext - b.historySize + historyCapacity) % historyCapacity
	for index := range history {
		history[index] = cloneEvent(b.history[(start+index)%historyCapacity])
	}
	return history
}

func (b *Bus) publishTo(subscription *Subscription, event Event) {
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
		b.dropped++
	}
}

func (b *Bus) unsubscribe(subscription *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.subscribers[subscription]; !exists {
		return
	}
	delete(b.subscribers, subscription)
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
	event.Details = append([]string(nil), event.Details...)
	return event
}
