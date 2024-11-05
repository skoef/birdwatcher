package birdwatcher

import (
	"net"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_addPrefix(t *testing.T) {
	t.Parallel()

	hc := HealthCheck{}
	assert.Nil(t, hc.prefixes)

	// adding a prefix should initialise the prefixcollection
	// and add the prefix under the right prefixset
	_, prefix, _ := net.ParseCIDR("1.2.3.0/24")
	hc.addPrefix(&ServiceCheck{name: "svc1", FunctionName: "foo"}, *prefix)
	assert.Len(t, hc.prefixes, 1)
	assert.Equal(t, *prefix, hc.prefixes["foo"].prefixes[0])

	assert.InEpsilon(t, 1.0, testutil.ToFloat64(prefixStateMetric.WithLabelValues("svc1", "1.2.3.0/24")), 0.00001)

	_, prefix, _ = net.ParseCIDR("2.3.4.0/24")
	hc.addPrefix(&ServiceCheck{name: "svc2", FunctionName: "bar"}, *prefix)
	assert.Len(t, hc.prefixes, 2)
	assert.Equal(t, *prefix, hc.prefixes["bar"].prefixes[0])

	assert.InEpsilon(t, 1.0, testutil.ToFloat64(prefixStateMetric.WithLabelValues("svc2", "2.3.4.0/24")), 0.00001)
}

func TestHealthCheck_removePrefix(t *testing.T) {
	t.Parallel()

	hc := HealthCheck{}
	assert.Nil(t, hc.prefixes)

	_, prefix, _ := net.ParseCIDR("1.2.3.0/24")

	svc1 := &ServiceCheck{name: "svc1", FunctionName: "foo"}
	hc.addPrefix(svc1, *prefix)
	assert.Len(t, hc.prefixes, 1)
	assert.Len(t, hc.prefixes["foo"].prefixes, 1)

	assert.InEpsilon(t, 1.0, testutil.ToFloat64(prefixStateMetric.WithLabelValues("svc1", "1.2.3.0/24")), 0.00001)

	// this should initialise the prefixset but won't remove any prefixes
	svc2 := &ServiceCheck{name: "svc2", FunctionName: "bar"}
	hc.removePrefix(svc2, *prefix)
	assert.Len(t, hc.prefixes, 2)
	assert.Len(t, hc.prefixes["foo"].prefixes, 1)
	assert.Empty(t, hc.prefixes["bar"].prefixes)

	assert.Empty(t, testutil.ToFloat64(prefixStateMetric.WithLabelValues("svc2", "1.2.3.0/24")))

	// remove the prefix from the right prefixset
	hc.removePrefix(svc1, *prefix)
	assert.Empty(t, hc.prefixes["foo"].prefixes)

	assert.Empty(t, testutil.ToFloat64(prefixStateMetric.WithLabelValues("svc1", "1.2.3.0/24")))
}

func TestHealthCheckDidReloadBefore(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck(Config{})

	// expect both to fail
	assert.False(t, hc.didReloadBefore())

	// should succeed now
	hc.reloadedBefore = true
	assert.True(t, hc.didReloadBefore())

	hc.reloadedBefore = false

	// expect to fail again
	assert.False(t, hc.didReloadBefore())
}

func TestHealthCheck_handleAction(t *testing.T) {
	t.Parallel()

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
	hc.handleAction(action, nil)

	if assert.Contains(t, hc.prefixes, "test") {
		assert.Len(t, hc.prefixes["test"].prefixes, 2)
	}

	// action switches to down for one of the prefixes
	action.State = ServiceStateDown
	action.Prefixes = action.Prefixes[1:]
	hc.handleAction(action, nil)

	if assert.Contains(t, hc.prefixes, "test") {
		assert.Len(t, hc.prefixes["test"].prefixes, 1)
	}
}

func TestHealthCheck_statusUpdate(t *testing.T) {
	t.Parallel()

	// healthcheck with 2 empty services
	hc := HealthCheck{services: []*ServiceCheck{
		{name: "foo"}, {name: "bar"},
	}}

	assert.Equal(t, "all 2 service(s) down", hc.statusUpdate())
	hc.services[0].state = ServiceStateUp
	assert.Equal(t, "service(s) bar down, 1 service(s) up", hc.statusUpdate())
	hc.services[1].state = ServiceStateUp
	assert.Equal(t, "all 2 service(s) up", hc.statusUpdate())
}
