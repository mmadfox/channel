package channel

import (
	"sync"
)

type bucketOptions struct {
	maxLimitSessions    int
	skipSlowSubscribers bool
	bufSize             int
}

type bucket struct {
	sync.RWMutex
	subscribers      map[string][]Subscription
	queue            chan []byte
	done             chan struct{}
	sessionCount     uint
	subscribersCount uint
	options          *bucketOptions
}

func newBucket(
	opt *bucketOptions,
) *bucket {
	b := &bucket{
		subscribers: make(map[string][]Subscription),
		queue:       make(chan []byte, 1),
		done:        make(chan struct{}),
		options:     opt,
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
	b.subscribersCount = uint(len(b.subscribers))
	if b.sessionCount > 0 {
		b.sessionCount--
	}
	return nil
}

func (b *bucket) publishTo(subscriber string, payload []byte) error {
	b.RLock()
	defer b.RUnlock()
	subscriptions, found := b.subscribers[subscriber]
	if !found {
		return ErrSubscriberNotFound
	}
	for i := len(subscriptions) - 1; i >= 0; i-- {
		subscriptions[i].Publish(
			payload,
			b.options.skipSlowSubscribers,
		)
	}
	return nil
}

func (b *bucket) publish(payload []byte) {
	b.RLock()
	defer b.RUnlock()
	for _, subscriptions := range b.subscribers {
		for i := len(subscriptions) - 1; i >= 0; i-- {
			subscriptions[i].Publish(
				payload,
				b.options.skipSlowSubscribers,
			)
		}
	}
}

func (b *bucket) close() {
	b.Lock()
	defer b.Unlock()
	for sid, subscriptions := range b.subscribers {
		for i := len(subscriptions) - 1; i >= 0; i-- {
			if !subscriptions[i].IsClosed() {
				subscriptions[i].Close()
				b.sessionCount--
			}

		}
		delete(b.subscribers, sid)
	}
	b.subscribersCount = uint(len(b.subscribers))
	close(b.done)
}

func (b *bucket) subscribe(subscriber string) (Subscription, error) {
	b.Lock()
	defer b.Unlock()
	subscriptions, found := b.subscribers[subscriber]
	if !found {
		subscriptions = make([]Subscription, 0, 1)
	}
	if len(subscriptions) >= b.options.maxLimitSessions {
		return Subscription{}, ErrTooManySessions
	}
	s := MakeSubscription(subscriber, b.options.bufSize)
	subscriptions = append(subscriptions, s)
	b.subscribers[subscriber] = subscriptions
	b.subscribersCount = uint(len(b.subscribers))
	b.sessionCount++
	return s, nil
}
