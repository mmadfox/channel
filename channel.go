package channel

import (
	"context"
	"errors"
	"runtime"
)

var (
	ErrInitBucket           = errors.New("channel: unable to initialize buckets")
	ErrSubscriptionNotFound = errors.New("channel: subscription not found")
	ErrTooManySessions      = errors.New("channel: too many sessions")
	ErrSubscriberNotFound   = errors.New("channel: subscriber not found")
)

func New(options ...Option) *Channel {
	numCPU := runtime.NumCPU()
	channel := Channel{
		bucketSize: numCPU,
		buckets:    make([]*bucket, numCPU),
		bucketOpts: &bucketOptions{
			maxLimitSessions:    DefaultMaxLimitSessions,
			skipSlowSubscribers: false,
			bufSize:             DefaultBufSize,
		},
	}
	for _, opt := range options {
		opt(channel.bucketOpts)
	}
	for i := 0; i < numCPU; i++ {
		channel.buckets[i] = newBucket(channel.bucketOpts)
	}
	return &channel
}

type Channel struct {
	bucketSize int
	buckets    []*bucket
	bucketOpts *bucketOptions
}

func (c *Channel) Close() error {
	for _, bucket := range c.buckets {
		bucket.close()
	}
	return nil
}

func (c *Channel) PublishToSubscriber(subscriber string, message []byte) error {
	bucket, err := c.bucket(subscriber)
	if err != nil {
		return err
	}
	return bucket.publishTo(subscriber, message)
}

func (c *Channel) PublishToSubscribers(subscribers []string, message []byte) error {
	for _, subscriber := range subscribers {
		bucket, err := c.bucket(subscriber)
		if err != nil {
			return err
		}
		err = bucket.publishTo(subscriber, message)
		if err != nil {
			if errors.Is(err, ErrSubscriberNotFound) {
				continue
			}
			return err
		}
	}
	return nil
}

func (c *Channel) PublishToAllSubscribers(message []byte) error {
	for _, bucket := range c.buckets {
		bucket.queue <- message
	}
	return nil
}

func (c *Channel) Listen(ctx context.Context, subscriber string,
	handle func(b []byte), cleanup func()) error {
	subscription, err := c.Subscribe(subscriber)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case payload := <-subscription.Channel():
				handle(payload)
			case <-ctx.Done():
				_ = c.Unsubscribe(subscription)
				cleanup()
				return
			}
		}
	}()
	return nil
}

func (c *Channel) Unsubscribe(s Subscription) error {
	bucket, err := c.bucket(s.Subscriber())
	if err != nil {
		return err
	}
	return bucket.unsubscribe(s.Subscriber(), s.Session())
}

func (c *Channel) Stats() (s Stats) {
	for _, bucket := range c.buckets {
		bucket.RLock()
		s.Sessions += uint(bucket.sessionCount)
		s.Subscribers += uint(bucket.subscribersCount)
		bucket.RUnlock()
	}
	return s
}

func (c *Channel) Subscribe(subscriber string) (Subscription, error) {
	bucket, err := c.bucket(subscriber)
	if err != nil {
		return Subscription{}, err
	}
	return bucket.subscribe(subscriber)
}

func (c *Channel) bucket(key string) (*bucket, error) {
	bucketIndex := int(fnv32(key)) % c.bucketSize
	if bucketIndex < c.bucketSize && bucketIndex > c.bucketSize {
		return nil, ErrInitBucket
	}
	return c.buckets[bucketIndex], nil
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
