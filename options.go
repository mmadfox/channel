package channel

type Option func(c *bucketOptions)

const (
	DefaultMaxLimitSessions = 10
	DefaultBufSize          = 1
)

// MaxLimitSessions maximum sessions per subscriber.
func MaxLimitSessions(limit int) Option {
	return func(c *bucketOptions) {
		if limit <= 0 {
			return
		}
		c.maxLimitSessions = limit
	}
}

// IgnoreSlowClients non-blocking message sending.
func IgnoreSlowClients() Option {
	return func(c *bucketOptions) {
		c.ignoreSlowClients = true
	}
}

// SubscriptionBufSize subscriptions buffered channel size.
func SubscriptionBufSize(size int) Option {
	return func(c *bucketOptions) {
		if size < 0 {
			size = 0
		}
		if size > 1000 {
			size = 1000
		}
		c.bufSize = size
	}
}
