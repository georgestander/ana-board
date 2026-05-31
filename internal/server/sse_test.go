package server

import "testing"

func TestBrokerKeepsLatestEventWhenSubscriberIsFull(t *testing.T) {
	broker := NewBroker()
	ch := broker.Subscribe()
	defer broker.Unsubscribe(ch)

	for index := 0; index < cap(ch)+3; index++ {
		broker.Broadcast(FrameEvent{Animation: "event"})
	}
	broker.Broadcast(FrameEvent{Animation: "latest"})

	gotLatest := false
	for len(ch) > 0 {
		event := <-ch
		if event.Animation == "latest" {
			gotLatest = true
		}
	}

	if !gotLatest {
		t.Fatal("subscriber did not receive latest event")
	}
}
