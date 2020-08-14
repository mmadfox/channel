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
	subscription, err := s.orderChanged.Subscribe(input.UserID)
	if err != nil {
		return nil, err
	}
	out := make(chan *Order)
	go func() {
		for {
			select {
			case payload := <-subscription.Channel():
				var o Order
				// OR protobuf, avro, etc
				if err := json.Unmarshal(payload, &o); err != nil {
					// logging here
				}

				out <- &o
			case <-ctx.Done():
				_ = s.orderChanged.Unsubscribe(subscription)
				close(out)
				return
			}
		}
	}()
	return out, nil
}

func (s *Subscriptions) SubscribeToCustomerChanged(ctx context.Context, input *GQLModelInput) (<-chan *Customer, error) {
	subscription, err := s.customerChanged.Subscribe(input.UserID)
	if err != nil {
		return nil, err
	}
	out := make(chan *Customer)
	go func() {
		for {
			select {
			case payload := <-subscription.Channel():
				var o Customer
				// OR protobuf, avro, etc
				if err := json.Unmarshal(payload, &o); err != nil {
					// logging
				}

				out <- &o
			case <-ctx.Done():
				_ = s.customerChanged.Unsubscribe(subscription)
				close(out)
				return
			}
		}
	}()
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
