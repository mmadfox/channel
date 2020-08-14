package channel

import (
	"sync"
)

type bucket struct {
	sync.RWMutex
	subscribers      map[string][]Subscription
	queue            chan []byte
	done             chan struct{}
	sessionCount     int
	subscribersCount int
	maxLimitSessions int
}

func newBucket(
	maxLimitSessions int,
) *bucket {
	b := &bucket{
		maxLimitSessions: maxLimitSessions,
		subscribers:      make(map[string][]Subscription),
		queue:            make(chan []byte, 1),
		done:             make(chan struct{}),
	}
	go b.listen()
	return b
}

func (b *bucket) listen() {
	for {
		select {
		case <-b.done:
			return
		case msg := <-b.queue:
			b.publish(msg)
		}
	}
}

func (b *bucket) unsubscribe(subscriber string, session string) error {
	b.Lock()
	defer b.Unlock()
	subscriptions, found := b.subscribers[subscriber]
	if !found {
		return ErrSubscriptionNotFound
	}
	for i, subscription := range subscriptions {
		if subscription.Equal(session) {
			subscriptions = append(subscriptions[:i], subscriptions[i+1:]...)
			subscription.Close()
			break
		}
	}
	if len(subscriptions) == 0 {
		delete(b.subscribers, subscriber)
	} else {
		b.subscribers[subscriber] = subscriptions
	}
	b.subscribersCount = len(b.subscribers)
	if b.sessionCount > 0 {
		b.sessionCount--
	}
	return nil
}

func (b *bucket) publish(payload []byte) {
	for _, sessions := range b.subscribers {
		for _, session := range sessions {
			select {
			case session.channel <- payload:
			default:
			}
		}
	}
}

func (b *bucket) close() {
	b.Lock()
	defer b.Unlock()
	close(b.done)
	for sid, subscriptions := range b.subscribers {
		for _, subscription := range subscriptions {
			if !subscription.IsClosed() {
				subscription.Close()
				b.sessionCount--
			}
		}
		delete(b.subscribers, sid)
	}
	b.subscribersCount = len(b.subscribers)
}

func (b *bucket) subscribe(subscriber string) (Subscription, error) {
	b.Lock()
	defer b.Unlock()
	subscriptions, found := b.subscribers[subscriber]
	if !found {
		subscriptions = make([]Subscription, 0, 3)
	}
	if len(subscriptions) >= b.maxLimitSessions {
		return Subscription{}, ErrTooManySessions
	}
	s := MakeSubscription(subscriber)
	subscriptions = append(subscriptions, s)
	b.subscribers[subscriber] = subscriptions
	b.subscribersCount = len(b.subscribers)
	b.sessionCount++
	return s, nil
}
