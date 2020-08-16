# channel
Channels for subscriptions. 
Used in GQLgen(GrpaphQL) and gRPC to forward messages from kafka broker to websocket clients.

### Example


```go 
import "github.com/mmadfox/channel"

type Channels struct {
     gpsLocationChanged *channel.Channel
     orderChanged *channel.Channel
     accountChanged *channel.Channel
}

func NewChannels() *Channel {
     channels := Channels{
          gpsLocationChanged: channel.New(channel.SkipSlowSubscribers()),
	  orderChanged: channel.New(channel.IgnoreSlowClient()),
	  accountChanged: channel.New(channel.IgnoreSlowClient()),
     }
     return &channels
}

func (c *Channels) PublishToGpsLocationChannel(loc GPSLocation) error {
    message, err := json.Marshal(loc)
    if err != nil {
        return err 
    }
    c.gpsLocationChanged.PublishToAllSubscribers(message)
}

// implementation for GQLgen compatibility
func (c *Channels) GpsLocationChanged(ctx context.Context, input *models.GPSLocationChangedInput) (
	<-chan *models.GPSLocation, error) {
	out := make(chan *models.GPSLocation)
	if err := c.gpsLocationChanged.Listen(ctx, input.AccountID, func(b []byte) {
		var m models.GPSLocation
		if err := json.Unmarshal(b, &m); err != nil {
			_ = s.logger.Log("channel", "gpsLocationChanged",
				"subscriber", input.UserID, "err", err)
		}
		out <- &m
	}, func() {
		close(out)
	}); err != nil {
		_ = s.logger.Log("channel", "gpsLocationChanged",
			"subscriber", input.UserID, "err", err)
		return nil, err
	}
	return out, nil
}

type Resolver struct {
	subscriptionResolver generated.SubscriptionResolver
	queryResolver        generated.QueryResolver
	mutationResolver     generated.MutationResolver
}

func NewResolver(
	subscription *Channels,
) *Resolver {
	return &Resolver{
		subscriptionResolver: subscription,
		mutationResolver:     mutation,
		queryResolver:        query,
	}
}

func (r *Resolver) Mutation() generated.MutationResolver {
	return r.mutationResolver
}

func (r *Resolver) Query() generated.QueryResolver {
	return r.queryResolver
}

func (r *Resolver) Subscription() generated.SubscriptionResolver {
	return r.subscriptionResolver
}

subscriptions := NewChannels()

gqlHandler := handler.NewDefaultServer(
    generated.NewExecutableSchema(
	generated.Config{
	    Resolvers: NewResolver(subscriptions, /* query, mutation */),
	},
    ))

```
#### Publish / Subscribe
```go
import "github.com/mmadfox/channel"

accountID := uuid.New().String()
gpsLocationChanged := channel.New()
subscription, err := gpsLocationChanged.Subscribe(accountID)
message := []byte("MSG")

// publish
go func() {
   gpsLocationChanged.PublishToSubscriber(accountID, message)
   // OR
   gpsLocationChanged.PublishToSubscribers([]string{accountID}, message)
   // OR
   gpsLocationChanged.PublishToAllSubscribers(message)  
}()

// ...

gpsLocationChanged.Close()
```

#### GQLgen
```go
import "github.com/mmadfox/channel"

type Subscriptions struct {
    gpsLocationChanged *channel.Channel 
}

// ...

func (s *Subscriptions) GpsLocationChanged(ctx context.Context, input *models.GPSLocationChangedInput) (
	<-chan *models.GPSLocation, error) {
	out := make(chan *models.GPSLocation)
	if err := s.gpsLocationChanged.Listen(ctx, input.AccountID, func(b []byte) {
		var m models.GPSLocation
		if err := json.Unmarshal(b, &m); err != nil {
			_ = s.logger.Log("channel", "gpsLocationChanged",
				"subscriber", input.UserID, "err", err)
		}
		out <- &m
	}, func() {
		close(out)
	}); err != nil {
		_ = s.logger.Log("channel", "gpsLocationChanged",
			"subscriber", input.UserID, "err", err)
		return nil, err
	}
	return out, nil
}
``` 

#### Options
```go
// maximum sessions per subscriber.
MaxLimitSessions(limit int) Option 
// non-blocking message sending.
SkipSlowSubscribers() Option
// subscriptions buffered channel size.
SubscriptionBufSize(size int) Option 
```
