package events

import (
	"sync"
	"testing"
	"time"
)

func TestEventBusPubSub(t *testing.T) {
	bus := NewBus()

	ch1, cancel1 := bus.Subscribe(10)
	defer cancel1()
	ch2, cancel2 := bus.Subscribe(10)
	defer cancel2()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case ev := <-ch1:
			if ev.Type != EventPeerUp {
				t.Errorf("ch1 unexpected event: %v", ev)
			}
		case <-time.After(2 * time.Second):
			t.Error("ch1 timeout waiting for event")
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case ev := <-ch2:
			if ev.Type != EventPeerUp {
				t.Errorf("ch2 unexpected event: %v", ev)
			}
		case <-time.After(2 * time.Second):
			t.Error("ch2 timeout waiting for event")
		}
	}()

	bus.Publish(Event{Type: EventPeerUp, Data: map[string]string{"alias": "device-1"}})
	wg.Wait()
}

func TestEventBusSlowConsumerDrop(t *testing.T) {
	bus := NewBus()

	// Buffer size 1: second event should be dropped for this subscriber without blocking
	ch, cancel := bus.Subscribe(1)
	defer cancel()

	bus.Publish(Event{Type: EventProgress, Data: 1})
	bus.Publish(Event{Type: EventProgress, Data: 2}) // should be dropped

	ev := <-ch
	if ev.Data != 1 {
		t.Errorf("expected first event data 1, got %v", ev.Data)
	}

	select {
	case dropped := <-ch:
		t.Errorf("expected no more events, but got %v", dropped)
	default:
		// success: second event dropped
	}
}
