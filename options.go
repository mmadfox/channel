package channel

type Option func(c *bucketOptions)

const (
	DefaultMaxLimitSessions = 10
	DefaultBufSize          = 10
)

func MaxLimitSessions(limit int) Option {
	return func(c *bucketOptions) {
		if limit <= 0 {
			return
		}
		c.maxLimitSessions = limit
	}
}

func IgnoreSlowClients() Option {
	return func(c *bucketOptions) {
		c.ignoreSlowClients = true
	}
}

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
