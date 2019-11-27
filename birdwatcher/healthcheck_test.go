package birdwatcher

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheckAddRemovePrefix(t *testing.T) {
	hc := HealthCheck{}

	// add some prefixes
	hc.addPrefix(net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	hc.addPrefix(net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	hc.addPrefix(net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	hc.addPrefix(net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}})

	assert.Equal(t, 4, len(hc.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[2])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}}, hc.prefixes[3])

	// add same prefix again
	// list should be the same
	hc.addPrefix(net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}})

	assert.Equal(t, 4, len(hc.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[2])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}}, hc.prefixes[3])

	// remove last prefix
	// array should only be truncated
	hc.removePrefix(net.IPNet{
		IP:   net.IP{3, 4, 5, 0},
		Mask: net.IPMask{255, 255, 255, 192},
	})

	assert.Equal(t, 3, len(hc.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[2])

	// remove first prefix
	// last prefix will be first now
	hc.removePrefix(net.IPNet{
		IP:   net.IP{1, 2, 3, 0},
		Mask: net.IPMask{255, 255, 255, 0},
	})

	assert.Equal(t, 2, len(hc.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[0])

	// removing same prefix again
	hc.removePrefix(net.IPNet{
		IP:   net.IP{1, 2, 3, 0},
		Mask: net.IPMask{255, 255, 255, 0},
	})

	assert.Equal(t, 2, len(hc.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, hc.prefixes[0])
}
