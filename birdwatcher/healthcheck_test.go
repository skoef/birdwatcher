package birdwatcher

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_addPrefix(t *testing.T) {
	hc := HealthCheck{}
	assert.Nil(t, hc.prefixes)

	// adding a prefix should initialise the prefixcollection
	// and add the prefix under the right prefixset
	_, prefix, _ := net.ParseCIDR("1.2.3.0/24")
	hc.addPrefix("foo", *prefix)
	assert.Equal(t, 1, len(hc.prefixes))
	assert.Equal(t, *prefix, hc.prefixes["foo"].prefixes[0])

	_, prefix, _ = net.ParseCIDR("2.3.4.0/24")
	hc.addPrefix("bar", *prefix)
	assert.Equal(t, 2, len(hc.prefixes))
	assert.Equal(t, *prefix, hc.prefixes["bar"].prefixes[0])
}

func TestHealthCheck_removePrefix(t *testing.T) {
	hc := HealthCheck{}
	assert.Nil(t, hc.prefixes)
	_, prefix, _ := net.ParseCIDR("1.2.3.0/24")
	hc.addPrefix("foo", *prefix)
	assert.Equal(t, 1, len(hc.prefixes))
	assert.Equal(t, 1, len(hc.prefixes["foo"].prefixes))

	// this should initialise the prefixset but won't remove any prefixes
	hc.removePrefix("bar", *prefix)
	assert.Equal(t, 2, len(hc.prefixes))
	assert.Equal(t, 1, len(hc.prefixes["foo"].prefixes))
	assert.Equal(t, 0, len(hc.prefixes["bar"].prefixes))

	// remove the prefix from the right prefixset
	hc.removePrefix("foo", *prefix)
	assert.Equal(t, 0, len(hc.prefixes["foo"].prefixes))
}

func TestHealthCheckDidReloadBefore(t *testing.T) {
	hc := NewHealthCheck(Config{})

	// expect both to fail
	assert.Equal(t, false, hc.didReloadBefore("ipv4"))
	assert.Equal(t, false, hc.didReloadBefore("ipv6"))

	hc.reloads["ipv4"] = true

	// expect only IPv6 to fail
	assert.Equal(t, true, hc.didReloadBefore("ipv4"))
	assert.Equal(t, false, hc.didReloadBefore("ipv6"))

	hc.reloads["ipv6"] = true

	// expect both to succeed
	assert.Equal(t, true, hc.didReloadBefore("ipv4"))
	assert.Equal(t, true, hc.didReloadBefore("ipv6"))

	hc.reloads["ipv4"] = false
	hc.reloads["ipv6"] = false

	// expect both to fail again
	assert.Equal(t, false, hc.didReloadBefore("ipv4"))
	assert.Equal(t, false, hc.didReloadBefore("ipv6"))
}

func TestHealthCheck_handleAction(t *testing.T) {
	// empty healthcheck
	hc := HealthCheck{}
	assert.Nil(t, hc.prefixes)

	// create action with state up and 2 prefixes
	action := &Action{
		State:    ServiceStateUp,
		Prefixes: make([]net.IPNet, 2),
	}
	var prefix *net.IPNet
	_, prefix, _ = net.ParseCIDR("1.2.3.0/24")
	action.Prefixes[0] = *prefix
	_, prefix, _ = net.ParseCIDR("2.3.4.0/24")
	action.Prefixes[1] = *prefix
	action.Service = &ServiceCheck{
		FunctionName: "test",
	}
	// handle service state up
	hc.handleAction(action)
	if assert.Contains(t, hc.prefixes, "test") {
		assert.Equal(t, 2, len(hc.prefixes["test"].prefixes))
	}

	// action switches to down for one of the prefixes
	action.State = ServiceStateDown
	action.Prefixes = action.Prefixes[1:]
	hc.handleAction(action)
	if assert.Contains(t, hc.prefixes, "test") {
		assert.Equal(t, 1, len(hc.prefixes["test"].prefixes))
	}
}
