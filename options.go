package channel

type Option func(c *Channel)

const (
	DefaultMaxLimitSessions = 10
)

func MaxLimitSessions(limit int) Option {
	return func(c *Channel) {
		if limit <= 0 {
			return
		}
		c.maxLimitSessions = limit
	}
}
