package birdwatcher

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrefixSet_Add(t *testing.T) {
	p := NewPrefixSet("foobar")
	// should be empty
	assert.Equal(t, 0, len(p.prefixes))

	// add some prefixes
	for _, pref := range []string{"1.2.3.0/24", "2.3.4.0/24", "3.4.5.0/24", "3.4.5.0/26"} {
		_, prf, _ := net.ParseCIDR(pref)
		p.Add(*prf)
	}

	// check if all 4 prefixes are there
	if assert.Equal(t, 4, len(p.prefixes)) {
		assert.Equal(t, "1.2.3.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
		assert.Equal(t, "3.4.5.0/24", p.prefixes[2].String())
		assert.Equal(t, "3.4.5.0/26", p.prefixes[3].String())
	}

	// try to add a duplicate prefix
	_, prf, _ := net.ParseCIDR("1.2.3.0/24")
	p.Add(*prf)

	// this shouldn't have changed the content of the PrefixSet
	if assert.Equal(t, 4, len(p.prefixes)) {
		assert.Equal(t, "1.2.3.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
		assert.Equal(t, "3.4.5.0/24", p.prefixes[2].String())
		assert.Equal(t, "3.4.5.0/26", p.prefixes[3].String())
	}
}

func TestPrefixSet_Remove(t *testing.T) {
	p := NewPrefixSet("foobar")

	// add some prefixes
	prefixes := make([]net.IPNet, 4)
	for i, pref := range []string{"1.2.3.0/24", "2.3.4.0/24", "3.4.5.0/24", "3.4.5.0/26"} {
		_, prf, _ := net.ParseCIDR(pref)
		p.Add(*prf)
		prefixes[i] = *prf
	}

	// remove last prefix
	// array should only be truncated
	p.Remove(prefixes[3])

	if assert.Equal(t, 3, len(p.prefixes)) {
		assert.Equal(t, "1.2.3.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
		assert.Equal(t, "3.4.5.0/24", p.prefixes[2].String())
	}

	// remove first prefix
	// last prefix will be first now
	p.Remove(prefixes[0])

	if assert.Equal(t, 2, len(p.prefixes)) {
		assert.Equal(t, "3.4.5.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
	}

	// removing same prefix again, should make no difference
	// (we're creating the prefix again since it is no longer in prefixes)
	p.Remove(net.IPNet{
		IP:   net.IP{1, 2, 3, 0},
		Mask: net.IPMask{255, 255, 255, 0},
	})

	if assert.Equal(t, 2, len(p.prefixes)) {
		assert.Equal(t, "3.4.5.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
	}
}

func TestPrefixSet_Marshal(t *testing.T) {
	p := NewPrefixSet("foobar")

	fixture, err := os.ReadFile("testdata/prefixset/no_prefixes")
	require.NoError(t, err)
	// should represent empty function returning false
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv4))

	// add some prefixes
	for _, pref := range []string{"1.2.3.4/32", "2.3.4.0/26", "3.4.5.6/24", "4.5.6.7/21"} {
		_, prf, _ := net.ParseCIDR(pref)
		p.Add(*prf)
	}

	// since these prefixes are only IPv4, IPv6 output should still be the same
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv6))

	// IPv4 should represent function matching above prefixes
	fixture, err = os.ReadFile("testdata/prefixset/some_prefixes")
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
	fixture, err = os.ReadFile("testdata/prefixset/some_prefixes_v6")
	require.NoError(t, err)
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv6))

	// if we change the function name, it should reflect in the output
	p.functionName = "something_else"

	fixture, err = os.ReadFile("testdata/prefixset/function_name")
	require.NoError(t, err)
	// should represent function matching above prefixes
	assert.Equal(t, string(fixture), p.Marshal(PrefixFamilyIPv4))
}

func TestPrefixSet_prefixPad(t *testing.T) {
	prefixes := make([]net.IPNet, 4)
	for i, pref := range []string{"1.2.3.0/24", "2.3.4.0/24", "3.4.5.0/24", "3.4.5.0/26"} {
		_, prf, _ := net.ParseCIDR(pref)
		prefixes[i] = *prf
	}

	padded := prefixPad(prefixes)
	assert.Equal(t, "1.2.3.0/24,2.3.4.0/24,3.4.5.0/24,3.4.5.0/26", strings.Join(padded, ""))
}
