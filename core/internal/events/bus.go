// Package events implements a lightweight in-process pub/sub bus.
//
// Producers call Publish; consumers Subscribe and receive events on a channel.
// Slow consumers are dropped (their channel is skipped) to avoid blocking producers.
package events

import (
	"sync"
)

// EventType identifies the kind of event.
type EventType string

const (
	EventPeerUp        EventType = "peer.up"
	EventPeerDown      EventType = "peer.down"
	EventOfferIncoming EventType = "offer.incoming"
	EventProgress      EventType = "progress"
	EventTransferDone  EventType = "transfer.done"
	EventTransferFailed EventType = "transfer.failed"
	EventGuestRequest  EventType = "guest.request"
)

// Event is a generic bus message.
type Event struct {
	Type EventType `json:"type"`
	Data any       `json:"data"`
}

// Bus is a thread-safe publish/subscribe event bus.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan Event
	nextID      uint64
}

// NewBus creates a new empty Bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[uint64]chan Event),
	}
}

// Subscribe returns a channel that receives events and a cancel function.
// The channel is buffered; if the buffer is full the event is dropped for this subscriber.
func (b *Bus) Subscribe(bufSize int) (<-chan Event, func()) {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	ch := make(chan Event, bufSize)
	b.subscribers[id] = ch
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		delete(b.subscribers, id)
		close(ch)
		b.mu.Unlock()
	}
	return ch, cancel
}

// Publish sends e to all subscribers. Slow consumers (full buffer) are skipped.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- e:
		default:
			// Slow consumer; drop this event rather than block.
		}
	}
}
