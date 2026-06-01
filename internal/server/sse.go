package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

type FrameEvent struct {
	Frame     frameResponse `json:"frame"`
	Animation string        `json:"animation"`
}

type Broker struct {
	mu             sync.RWMutex
	subscribers    map[chan FrameEvent]struct{}
	maxSubscribers int
}

var ErrSubscriberLimitReached = errors.New("maximum event subscribers reached")

func NewBroker() *Broker {
	return NewBrokerWithLimit(DefaultMaxSubscribers)
}

func NewBrokerWithLimit(maxSubscribers int) *Broker {
	return &Broker{
		subscribers:    make(map[chan FrameEvent]struct{}),
		maxSubscribers: maxSubscribers,
	}
}

func (b *Broker) Subscribe() (chan FrameEvent, error) {
	ch := make(chan FrameEvent, 8)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.maxSubscribers > 0 && len(b.subscribers) >= b.maxSubscribers {
		return nil, ErrSubscriberLimitReached
	}

	b.subscribers[ch] = struct{}{}
	return ch, nil
}

func (b *Broker) Unsubscribe(ch chan FrameEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

func (b *Broker) Broadcast(event FrameEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- event:
			default:
			}
		}
	}
}

func writeSSE(w http.ResponseWriter, eventName string, event FrameEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, data)
	return err
}
