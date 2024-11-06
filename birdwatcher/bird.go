package birdwatcher

import (
	"bytes"
	_ "embed"
	"errors"
	"net"
	"os"
	"text/template"
)

//go:embed templates/functions.tpl
var functionsTemplate string

// make sure prefixPad can be used in templates
var tplFuncs = template.FuncMap{
	"prefixPad": prefixPad,
}

var errConfigIdentical = errors.New("configuration file is identical")

func updateBirdConfig(config Config, prefixes PrefixCollection) error {
	// write config to temp file
	tmpFilename := config.ConfigFile + ".tmp"
	// make sure we don't keep tmp file around when something goes wrong
	defer func(x string) {
		if _, err := os.Stat(x); !os.IsNotExist(err) {
			//nolint:errcheck,gosec // it's just a temp file anyway
			os.Remove(tmpFilename)
		}
	}(tmpFilename)

	if err := writeBirdConfig(tmpFilename, prefixes, config.CompatBird213); err != nil {
		return err
	}

	// compare new file with original config file
	if compareFiles(tmpFilename, config.ConfigFile) {
		return errConfigIdentical
	}

	// move tmp file to right place
	return os.Rename(tmpFilename, config.ConfigFile)
}

func writeBirdConfig(filename string, prefixes PrefixCollection, compatBird213 bool) error {
	var err error

	// open file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	tmpl := template.Must(template.New("func").Funcs(tplFuncs).Parse(functionsTemplate))

	tplBody := struct {
		Collections   PrefixCollection
		CompatBird213 bool
	}{
		Collections:   prefixes,
		CompatBird213: compatBird213,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tplBody); err != nil {
		return err
	}

	// write data to file
	_, err = f.Write(buf.Bytes())

	return err
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

func compareFiles(fileA, fileB string) bool {
	data, err := os.ReadFile(fileA)
	if err != nil {
		return false
	}

	datb, err := os.ReadFile(fileB)
	if err != nil {
		return false
	}

	return bytes.Equal(data, datb)
}
