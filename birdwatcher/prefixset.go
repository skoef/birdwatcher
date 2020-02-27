package birdwatcher

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
)

// PrefixSet represents a list of prefixes alongside a function name
type PrefixSet struct {
	prefixes []net.IPNet
	functionName string
}

// NewPrefixSet returns a new prefixset with given function name
func NewPrefixSet(functionName string) *PrefixSet {
	return &PrefixSet{functionName: functionName}
}

// Add adds a prefix to the PrefixSet if it wasn't already in it
func (p *PrefixSet) Add(prefix net.IPNet) {
	log.WithFields(log.Fields{
		"prefix": prefix,
	}).Debug("adding prefix to prefix set")

	// skip prefix if it's already in the list
	// shouldn't really happen though
	for _, pref := range p.prefixes {
		if pref.IP.Equal(prefix.IP) && bytes.Equal(pref.Mask, prefix.Mask) {
			log.WithFields(log.Fields{
				"prefix": prefix,
			}).Warn("duplicate prefix, skipping")
			return
		}
	}

	// add prefix to the prefix set
	p.prefixes = append(p.prefixes, prefix)
}

// Remove removes a prefix from the PrefixSet
func (p *PrefixSet) Remove(prefix net.IPNet) {
	log.WithFields(log.Fields{
		"prefix": prefix,
	}).Debug("removing prefix from prefix set")

	// go over global prefix list and remove it when found
	for i, pref := range p.prefixes {
		if pref.IP.Equal(prefix.IP) && bytes.Equal(pref.Mask, prefix.Mask) {
			// remove entry from slice, fast approach
			p.prefixes[i] = p.prefixes[len(p.prefixes)-1] // copy last element to index i
			//h.prefixes[len(h.prefixes)-1] = nil // erase last element
			p.prefixes = p.prefixes[:len(p.prefixes)-1] // truncate slice
			return
		}
	}

	log.WithFields(log.Fields{
		"prefix": prefix,
	}).Warn("prefix not found in prefix set, skipping")
}

// Marshal returns the BIRD function for this prefixset
func (p PrefixSet) Marshal() string {
	// begin of function
	output := fmt.Sprintf("function %s()\n{\n\treturn ", p.functionName)

	if len(p.prefixes) == 0 {
		output += "false;\n"
	} else {

		// begin array
		output += "net ~ [\n"

		// add all prefixes on single lines
		suffix := ","
		for i, pref := range p.prefixes {
			// if this is the last entry, we don't need a trailing comma
			if i == len(p.prefixes)-1 {
				suffix = ""
			}
			output += fmt.Sprintf("\t\t%s%s\n", pref.String(), suffix)
		}

		// end array
		output += "\t];\n"
	}

	// add footer
	output += "}\n"

	return output
}