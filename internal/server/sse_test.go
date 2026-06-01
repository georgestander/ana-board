package server

import "testing"

func TestBrokerKeepsLatestEventWhenSubscriberIsFull(t *testing.T) {
	broker := NewBroker()
	ch, err := broker.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
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

func TestBrokerRejectsSubscriberPastLimit(t *testing.T) {
	broker := NewBrokerWithLimit(1)

	ch, err := broker.Subscribe()
	if err != nil {
		t.Fatalf("first Subscribe returned error: %v", err)
	}
	defer broker.Unsubscribe(ch)

	_, err = broker.Subscribe()
	if err != ErrSubscriberLimitReached {
		t.Fatalf("second Subscribe error = %v, want %v", err, ErrSubscriberLimitReached)
	}
}
