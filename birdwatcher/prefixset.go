package birdwatcher

import (
	"bytes"
	// use embed for embedding the function template
	_ "embed"
	"net"

	log "github.com/sirupsen/logrus"
)

// PrefixCollection represents prefixsets per function name
type PrefixCollection map[string]*PrefixSet

// PrefixSet represents a list of prefixes alongside a function name
type PrefixSet struct {
	prefixes     []net.IPNet
	functionName string
}

// NewPrefixSet returns a new prefixset with given function name
func NewPrefixSet(functionName string) *PrefixSet {
	return &PrefixSet{functionName: functionName}
}

// FunctionName returns the function name
func (p PrefixSet) FunctionName() string {
	return p.functionName
}

// Prefixes returns the prefixes
func (p PrefixSet) Prefixes() []net.IPNet {
	return p.prefixes
}

// Add adds a prefix to the PrefixSet if it wasn't already in it
func (p *PrefixSet) Add(prefix net.IPNet) {
	pLog := log.WithFields(log.Fields{
		"prefix": prefix,
	})
	pLog.Debug("adding prefix to prefix set")

	// skip prefix if it's already in the list
	// shouldn't really happen though
	for _, pref := range p.prefixes {
		if pref.IP.Equal(prefix.IP) && bytes.Equal(pref.Mask, prefix.Mask) {
			pLog.Warn("duplicate prefix, skipping")

			return
		}
	}

	// add prefix to the prefix set
	p.prefixes = append(p.prefixes, prefix)
}

// Remove removes a prefix from the PrefixSet
func (p *PrefixSet) Remove(prefix net.IPNet) {
	pLog := log.WithFields(log.Fields{
		"prefix": prefix,
	})
	pLog.Debug("removing prefix from prefix set")

	// go over global prefix list and remove it when found
	for i, pref := range p.prefixes {
		if pref.IP.Equal(prefix.IP) && bytes.Equal(pref.Mask, prefix.Mask) {
			// remove entry from slice, fast approach
			p.prefixes[i] = p.prefixes[len(p.prefixes)-1] // copy last element to index i
			p.prefixes = p.prefixes[:len(p.prefixes)-1]   // truncate slice

			return
		}
	}

	pLog.Warn("prefix not found in prefix set, skipping")
}
