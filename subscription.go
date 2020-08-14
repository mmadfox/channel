package channel

import "github.com/google/uuid"

type Subscription struct {
	subscriber string
	channel    chan []byte
	session    string
}

func MakeSubscription(subscriber string) Subscription {
	return Subscription{
		session:    uuid.New().String(),
		channel:    make(chan []byte, 1),
		subscriber: subscriber,
	}
}

func (s Subscription) Equal(session string) bool {
	return s.session == session
}

func (s Subscription) Channel() chan []byte {
	return s.channel
}

func (s Subscription) Subscriber() string {
	return s.subscriber
}

func (s Subscription) Session() string {
	return s.session
}

func (s Subscription) Close() {
	if s.IsClosed() {
		return
	}
	close(s.channel)
}

func (s Subscription) IsClosed() bool {
	select {
	case <-s.channel:
		return true
	default:
		return false
	}
}

type Stats struct {
	Subscribers int
	Sessions    int
}
