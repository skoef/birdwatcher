package birdwatcher

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"testing"
)

func TestPrefixSet_Add(t *testing.T) {
	p := NewPrefixSet("foobar")
	// should be empty
	assert.Equal(t, 0, len(p.prefixes))

	// add some prefixes
	p.Add(net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	p.Add(net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	p.Add(net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	p.Add(net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}})

	// check if all 4 prefixes are there
	assert.Equal(t, 4, len(p.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[2])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}}, p.prefixes[3])

	// try to add a duplicate prefix
	p.Add(net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}})

	// this shouldn't have changed the content of the PrefixSet
	assert.Equal(t, 4, len(p.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[2])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}}, p.prefixes[3])
}

func TestPrefixSet_Remove(t *testing.T) {
	p := NewPrefixSet("foobar")

	// add some prefixes
	p.Add(net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	p.Add(net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	p.Add(net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}})
	p.Add(net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 192}})

	// remove last prefix
	// array should only be truncated
	p.Remove(net.IPNet{
		IP:   net.IP{3, 4, 5, 0},
		Mask: net.IPMask{255, 255, 255, 192},
	})

	assert.Equal(t, 3, len(p.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{1, 2, 3, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[2])

	// remove first prefix
	// last prefix will be first now
	p.Remove(net.IPNet{
		IP:   net.IP{1, 2, 3, 0},
		Mask: net.IPMask{255, 255, 255, 0},
	})

	assert.Equal(t, 2, len(p.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[0])

	// removing same prefix again, should make no difference
	p.Remove(net.IPNet{
		IP:   net.IP{1, 2, 3, 0},
		Mask: net.IPMask{255, 255, 255, 0},
	})

	assert.Equal(t, 2, len(p.prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{2, 3, 4, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[1])
	assert.Equal(t, net.IPNet{IP: net.IP{3, 4, 5, 0}, Mask: net.IPMask{255, 255, 255, 0}}, p.prefixes[0])
}

func TestPrefixSet_Marshal(t *testing.T) {
	p := NewPrefixSet("foobar")

	fixture, err := ioutil.ReadFile("testdata/prefixset/no_prefixes")
	require.NoError(t, err)
	// should represent empty function returning false
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv4))

	// add some prefixes
	p.Add(net.IPNet{IP: net.IP{1, 2, 3, 4}, Mask: net.IPMask{255, 255, 255, 255}})
	p.Add(net.IPNet{IP: net.IP{2, 3, 4, 5}, Mask: net.IPMask{255, 255, 255, 192}})
	p.Add(net.IPNet{IP: net.IP{3, 4, 5, 6}, Mask: net.IPMask{255, 255, 255, 0}})
	p.Add(net.IPNet{IP: net.IP{4, 5, 6, 7}, Mask: net.IPMask{255, 255, 248, 0}})

	// since these prefixes are only IPv4, IPv6 output should still be the same
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv6))

	// IPv4 should represent function matching above prefixes
	fixture, err = ioutil.ReadFile("testdata/prefixset/some_prefixes")
	require.NoError(t, err)
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv4))

	// add IPv6 prefixes
	_, pref, _ := net.ParseCIDR("2001::/64")
	p.Add(*pref)
	_, pref, _ = net.ParseCIDR("2002::/48")
	p.Add(*pref)

	// IPv4 output should still be the same
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv4))

	// IPv6 should represent the two prefixes
	fixture, err = ioutil.ReadFile("testdata/prefixset/some_prefixes_v6")
	require.NoError(t, err)
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv6))

	// if we change the function name, it should reflect in the output
	p.functionName = "something_else"

	fixture, err = ioutil.ReadFile("testdata/prefixset/function_name")
	require.NoError(t, err)
	// should represent function matching above prefixes
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv4))
}
