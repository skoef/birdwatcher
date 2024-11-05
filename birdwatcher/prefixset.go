package birdwatcher

import (
	"bytes"
	// use embed for embedding the function template
	_ "embed"
	"net"
	"text/template"

	log "github.com/sirupsen/logrus"
)

//go:embed templates/function.tpl
var functionTemplate string

var tplFuncs = template.FuncMap{
	"prefpad": prefixPad,
}

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

// Marshal returns the BIRD function for this prefixset
func (p PrefixSet) Marshal() string {
	// init template
	tmpl := template.Must(template.New("func").Funcs(tplFuncs).Parse(functionTemplate))

	// init template body
	tplBody := struct {
		FunctionName string
		Prefixes     []net.IPNet
	}{
		FunctionName: p.functionName,
		Prefixes:     p.prefixes,
	}

	// execute template and return output
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tplBody); err != nil {
		log.WithError(err).Error("could not parse template body")
	}

	return buf.String()
}

// prefixPad is a helper function for the template
// basically returns CIDR notations per IPNet, each suffixed with a , except for
// the last entry
func prefixPad(x []net.IPNet) []string {
	pp := make([]string, len(x))
	for i, p := range x {
		pp[i] = p.String()
		if i < len(x)-1 {
			pp[i] += ","
		}
	}

	return pp
}
