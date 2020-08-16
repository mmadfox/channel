package channel

import "github.com/google/uuid"

type Subscription struct {
	subscriber string
	channel    chan []byte
	session    string
}

func MakeSubscription(subscriber string, chBufSize int) Subscription {
	return Subscription{
		session:    uuid.New().String(),
		channel:    make(chan []byte, chBufSize),
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

func (s *Subscription) Publish(payload []byte, async bool) {
	if async {
		select {
		case s.channel <- payload:
		default:
		}
	} else {
		s.channel <- payload
	}
}

func (s Subscription) String() string {
	return "Subscription{sid:" + s.session + ", subscriber:" + s.subscriber + "}"
}

type Stats struct {
	Subscribers uint
	Sessions    uint
}
