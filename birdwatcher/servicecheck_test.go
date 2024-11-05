package birdwatcher

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServiceCheckPushChannel(t *testing.T) {
	t.Parallel()

	buf := make(chan *Action)
	sc := ServiceCheck{
		disablePrefixCheck: true,
		name:               "test",
		Command:            "/usr/bin/true",
		Fail:               3,
		Rise:               2,
		Interval:           1,
		Timeout:            2 * time.Second,
		prefixes: []net.IPNet{
			{IP: net.IP{1, 2, 3, 4}, Mask: net.IPMask{255, 255, 255, 0}},
		},
	}

	// start the check
	go sc.Start(&buf)
	defer sc.Stop()

	// wait for action on channel
	action := <-buf
	assert.Equal(t, ServiceStateUp, action.State)
	assert.Len(t, action.Prefixes, 1)
	assert.Equal(t, sc.prefixes[0], action.Prefixes[0])

	// all of a sudden, the check gives wrong result
	sc.Command = "/usr/bin/false"

	// wait for action on channel
	action = <-buf
	assert.Equal(t, ServiceStateDown, action.State)
}
