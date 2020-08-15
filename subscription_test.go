package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeSubscription(t *testing.T) {
	s := MakeSubscription("subs", 1)
	assert.Equal(t, "subs", s.Subscriber())
	assert.NotEmpty(t, s.Session())
	assert.True(t, s.Equal(s.Session()))
	assert.NotNil(t, s.Channel())
	assert.False(t, s.IsClosed())
	s.Close()
	s.Close()
	assert.True(t, s.IsClosed())
	defer func() {
		assert.Contains(t, "send on closed channel", recover())
	}()
	s.Channel() <- []byte("HOHO")
}
