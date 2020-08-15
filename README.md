# channel
gqlgen channels for subscriptions

### Example
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
// maximum session limit for one subscriber.
MaxLimitSessions(limit int) Option 
// non-blocking message sending.
IgnoreSlowClients() Option
// subscriptions buffered channel size.
SubscriptionBufSize(size int) Option 
```