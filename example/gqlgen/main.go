package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/mmadfox/channel"
)

func main() {
	subscriptions := NewSubscriptions()
	ctx, cancel := context.WithCancel(context.Background())

	gqlgenSubscriptionAPI(ctx, subscriptions)
	serverAPI(ctx, subscriptions)

	<-time.After(10 * time.Second)
	cancel()

	<-time.After(2 * time.Second)
}

func gqlgenSubscriptionAPI(ctx context.Context, s *Subscriptions) {
	go func() {
		ch, _ := s.SubscribeToCustomerChanged(ctx, &GQLModelInput{UserID: "mmadfox"})
		for {
			select {
			case m := <-ch:
				log.Println("Customer", m)
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		ch, _ := s.SubscribeToCustomerChanged(ctx, &GQLModelInput{UserID: "mmadfox2"})
		for {
			select {
			case m := <-ch:
				log.Println("Customer2", m)
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		ch, _ := s.SubscribeToOrderChanged(ctx, &GQLModelInput{UserID: "mmadfox"})
		for {
			select {
			case m := <-ch:
				log.Println("Order", m)
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		ch, _ := s.SubscribeToOrderChanged(ctx, &GQLModelInput{UserID: "mmadfox2"})
		for {
			select {
			case m := <-ch:
				log.Println("Order2", m)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func serverAPI(ctx context.Context, s *Subscriptions) {
	go func() {
		t := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-t.C:
				b, _ := json.Marshal(&Customer{UserID: "mmadfox", Code: 5})
				_ = s.PublishCustomerChangedFromKafka(ctx, b)
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		t := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-t.C:
				b, _ := json.Marshal(&Order{UserID: "mmadfox", Price: 33})
				_ = s.PublishOrderChangedFromKafka(ctx, b)
			case <-ctx.Done():
				return
			}
		}
	}()
}

type Subscriptions struct {
	orderChanged    *channel.Channel
	customerChanged *channel.Channel
}

func NewSubscriptions() *Subscriptions {
	maxSess := channel.MaxLimitSessions(3)
	return &Subscriptions{
		orderChanged:    channel.New(maxSess),
		customerChanged: channel.New(maxSess),
	}
}

func (s *Subscriptions) PublishOrderChangedFromKafka(ctx context.Context, payload []byte) error {
	return s.orderChanged.Publish(payload)
}

func (s *Subscriptions) PublishCustomerChangedFromKafka(ctx context.Context, payload []byte) error {
	return s.customerChanged.Publish(payload)
}

func (s *Subscriptions) SubscribeToOrderChanged(ctx context.Context, input *GQLModelInput) (<-chan *Order, error) {
	out := make(chan *Order)
	if err := s.orderChanged.Listen(ctx, input.UserID, func(msg []byte) {
		var o Order
		if err := json.Unmarshal(msg, &o); err != nil {
			// logging here
		}
		out <- &o
	}, func() {
		close(out)
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Subscriptions) SubscribeToCustomerChanged(ctx context.Context, input *GQLModelInput) (<-chan *Customer, error) {
	out := make(chan *Customer)
	if err := s.customerChanged.Listen(ctx, input.UserID, func(msg []byte) {
		var o Customer
		if err := json.Unmarshal(msg, &o); err != nil {
			// logging here
		}
		out <- &o
	}, func() {
		close(out)
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type GQLModelInput struct {
	UserID string
}

type Order struct {
	UserID string
	Price  float64
}

type Customer struct {
	UserID string
	Code   float64
}
