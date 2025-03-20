package birdwatcher

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefixSet_Add(t *testing.T) {
	t.Parallel()

	p := NewPrefixSet("foobar", true)
	// should be empty
	assert.Empty(t, p.prefixes)

	// add some prefixes
	for _, pref := range []string{"1.2.3.0/24", "2.3.4.0/24", "3.4.5.0/24", "3.4.5.0/26"} {
		_, prf, _ := net.ParseCIDR(pref)
		p.Add(*prf)
	}

	// check if all 4 prefixes are there
	if assert.Len(t, p.prefixes, 4) {
		assert.Equal(t, "1.2.3.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
		assert.Equal(t, "3.4.5.0/24", p.prefixes[2].String())
		assert.Equal(t, "3.4.5.0/26", p.prefixes[3].String())
	}

	// try to add a duplicate prefix
	_, prf, _ := net.ParseCIDR("1.2.3.0/24")
	p.Add(*prf)

	// this shouldn't have changed the content of the PrefixSet
	if assert.Len(t, p.prefixes, 4) {
		assert.Equal(t, "1.2.3.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
		assert.Equal(t, "3.4.5.0/24", p.prefixes[2].String())
		assert.Equal(t, "3.4.5.0/26", p.prefixes[3].String())
	}
}

func TestPrefixSet_Remove(t *testing.T) {
	t.Parallel()

	p := NewPrefixSet("foobar", true)

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

	if assert.Len(t, p.prefixes, 3) {
		assert.Equal(t, "1.2.3.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
		assert.Equal(t, "3.4.5.0/24", p.prefixes[2].String())
	}

	// remove first prefix
	// last prefix will be first now
	p.Remove(prefixes[0])

	if assert.Len(t, p.prefixes, 2) {
		assert.Equal(t, "3.4.5.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
	}

	// removing same prefix again, should make no difference
	p.Remove(prefixes[0])

	if assert.Len(t, p.prefixes, 2) {
		assert.Equal(t, "3.4.5.0/24", p.prefixes[0].String())
		assert.Equal(t, "2.3.4.0/24", p.prefixes[1].String())
	}
}
