package channel

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	ch := New()
	numCPU := runtime.NumCPU()
	assert.Len(t, ch.buckets, numCPU)
	assert.Equal(t, ch.bucketSize, numCPU)
	for _, b := range ch.buckets {
		assert.NotNil(t, b)
		assert.NotNil(t, b.done)
		assert.NotNil(t, b.queue)
		assert.NotNil(t, b.subscribers)
	}
}

func TestChannel_OneSubscriberManySessions(t *testing.T) {
	customerChannel := New()
	s1, err := customerChannel.Subscribe("qwerty")
	assert.Nil(t, err)
	s2, err := customerChannel.Subscribe("qwerty")
	assert.Nil(t, err)
	s3, err := customerChannel.Subscribe("qwerty")
	assert.Nil(t, err)
	stats := customerChannel.Stats()
	assert.Equal(t, 1, stats.Subscribers)
	assert.Equal(t, 3, stats.Sessions)

	assert.Nil(t, customerChannel.Unsubscribe(s1))
	assert.Nil(t, customerChannel.Unsubscribe(s2))
	assert.Nil(t, customerChannel.Unsubscribe(s3))

	stats = customerChannel.Stats()
	assert.Equal(t, 0, stats.Subscribers)
	assert.Equal(t, 0, stats.Sessions)
}

func TestChannel_SubscribeMaxLimitSessions(t *testing.T) {
	customerChannel := New(MaxLimitSessions(1))
	_, err := customerChannel.Subscribe("qwerty")
	assert.Nil(t, err)
	_, err = customerChannel.Subscribe("qwerty")
	assert.Equal(t, ErrTooManySessions, err)
}

func TestChannel_Close(t *testing.T) {
	customerChannel := New(MaxLimitSessions(15))
	s1, err := customerChannel.Subscribe("qwerty1")
	assert.Nil(t, err)
	s2, err := customerChannel.Subscribe("qwerty2")
	assert.Nil(t, err)
	s3, err := customerChannel.Subscribe("qwerty3")
	assert.Nil(t, err)

	stats := customerChannel.Stats()
	assert.Equal(t, 3, stats.Subscribers)
	assert.Equal(t, 3, stats.Sessions)

	assert.Nil(t, customerChannel.Close())
	assert.True(t, s1.IsClosed())
	assert.True(t, s2.IsClosed())
	assert.True(t, s3.IsClosed())

	stats = customerChannel.Stats()
	assert.Equal(t, 0, stats.Subscribers)
	assert.Equal(t, 0, stats.Sessions)
}

func TestChannel_Listen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	customerChannel := New()
	var flagc int32
	err := customerChannel.Listen(ctx, "mmadfox", func(b []byte) {
		assert.Equal(t, "mmadfox", string(b))
		atomic.AddInt32(&flagc, 1)
	}, func() {
		atomic.AddInt32(&flagc, 1)
	})
	assert.Nil(t, err)
	assert.Nil(t, customerChannel.PublishToAllSubscribers([]byte("mmadfox")))
	<-time.After(time.Second)
	cancel()
	<-time.After(time.Second)
	assert.Equal(t, int32(2), atomic.LoadInt32(&flagc))
}

func TestChannel_Subscribe(t *testing.T) {
	customerChannel := New()
	var counter int32
	var wg sync.WaitGroup
	subscribers := 10
	for i := 0; i < subscribers; i++ {
		subscription, err := customerChannel.Subscribe(fmt.Sprintf("user-%d", i))
		assert.Nil(t, err)
		wg.Add(1)
		go func(s Subscription) {
			msg := <-s.Channel()
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			assert.NotEmpty(t, msg)
			atomic.AddInt32(&counter, 1)
			wg.Done()
		}(subscription)
	}
	stats := customerChannel.Stats()
	assert.Equal(t, subscribers, stats.Subscribers)
	assert.Equal(t, subscribers, stats.Sessions)
	err := customerChannel.PublishToAllSubscribers([]byte("MSG"))
	assert.Nil(t, err)
	wg.Wait()
	assert.Equal(t, atomic.LoadInt32(&counter), int32(subscribers))
}

func TestChannel_SubscribeConcurrency(t *testing.T) {
	customerChannel := New()
	subscribers := 10
	for i := 0; i < subscribers; i++ {
		userName := fmt.Sprintf("user-%d", i)
		t.Run(userName, func(t *testing.T) {
			t.Parallel()
			subscription, err := customerChannel.Subscribe(userName)
			assert.Nil(t, err)
			assert.NotEmpty(t, subscription.Session())
		})
	}
}

func TestChannel_PublishToSubscribers(t *testing.T) {
	customerChannel := New(IgnoreSlowClients())
	subscribers := 10
	stats := make(map[string]int)
	var mu sync.RWMutex
	var step sync.WaitGroup
	for i := 0; i < subscribers; i++ {
		subscriberID := fmt.Sprintf("sub-%d", i)
		subscription, err := customerChannel.Subscribe(subscriberID)
		assert.Nil(t, err)
		stats[subscription.Subscriber()] = 0
		step.Add(1)
		go func(s Subscription) {
			step.Done()
			for {
				<-s.Channel()
				mu.Lock()
				stats[s.Subscriber()]++
				mu.Unlock()
			}
		}(subscription)
	}
	step.Wait()
	assert.Nil(t, customerChannel.PublishToSubscribers([]string{"sub-0", "sub-1", "_bad_"}, []byte("MSG")))
	time.Sleep(10 * time.Millisecond)
	for sid, cnt := range stats {
		if sid == "sub-0" || sid == "sub-1" {
			assert.Equal(t, 1, cnt)
		} else {
			assert.Equal(t, 0, cnt)
		}
	}
}

func TestChannel_PublishToSubscriber(t *testing.T) {
	customerChannel := New(IgnoreSlowClients())
	subscribers := 10
	stats := make(map[string]int)
	var mu sync.RWMutex
	var step sync.WaitGroup
	for i := 0; i < subscribers; i++ {
		subscriberID := fmt.Sprintf("sub-%d", i)
		subscription, err := customerChannel.Subscribe(subscriberID)
		assert.Nil(t, err)
		stats[subscription.Subscriber()] = 0
		step.Add(1)
		go func(s Subscription) {
			step.Done()
			for {
				<-s.Channel()
				mu.Lock()
				stats[s.Subscriber()]++
				mu.Unlock()
			}
		}(subscription)
	}
	step.Wait()
	assert.Nil(t, customerChannel.PublishToSubscriber("sub-0", []byte("MSG")))
	assert.Nil(t, customerChannel.PublishToSubscriber("sub-0", []byte("MSG")))
	assert.Nil(t, customerChannel.PublishToSubscriber("sub-0", []byte("MSG")))
	assert.Nil(t, customerChannel.PublishToSubscriber("sub-0", []byte("MSG")))
	time.Sleep(10 * time.Millisecond)
	for sid, cnt := range stats {
		if sid == "sub-0" {
			assert.Equal(t, 4, cnt)
		} else {
			assert.Equal(t, 0, cnt)
		}
	}
}

func TestChannel_Unsubscribe(t *testing.T) {
	customerChannel := New()
	count := 10
	subscriptions := make([]Subscription, 10)
	for i := 0; i < count; i++ {
		subscription, err := customerChannel.Subscribe(fmt.Sprintf("user-%d", i))
		assert.Nil(t, err)
		subscriptions[i] = subscription
	}

	stats := customerChannel.Stats()
	assert.Equal(t, count, stats.Subscribers)
	assert.Equal(t, count, stats.Sessions)

	for _, s := range subscriptions {
		assert.False(t, s.IsClosed())
		err := customerChannel.Unsubscribe(s)
		assert.Nil(t, err)
		assert.True(t, s.IsClosed())
	}

	stats = customerChannel.Stats()
	assert.Equal(t, 0, stats.Subscribers)
	assert.Equal(t, 0, stats.Sessions)
}

func BenchmarkChannel_Publish(b *testing.B) {
	customerChannel := New()
	max := 1000
	finished := make(chan struct{}, max)
	for i := 0; i < max; i++ {
		n := fmt.Sprintf("n%d", i)
		s, err := customerChannel.Subscribe(n)
		if err != nil {
			b.Fail()
		}
		go func(s Subscription) {
			<-s.Channel()
			finished <- struct{}{}
		}(s)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = customerChannel.PublishToAllSubscribers([]byte("MSG"))
	}
	for i := 0; i < max; i++ {
		<-finished
	}
}
