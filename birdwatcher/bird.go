package birdwatcher

import (
	"bytes"
	"errors"
	"os"
)

var errConfigIdentical = errors.New("configuration file is identical")

func updateBirdConfig(filename string, prefixes PrefixCollection) error {
	// write config to temp file
	tmpFilename := filename + ".tmp"
	// make sure we don't keep tmp file around when something goes wrong
	defer func(x string) {
		if _, err := os.Stat(x); !os.IsNotExist(err) {
			//nolint:errcheck,gosec // it's just a temp file anyway
			os.Remove(tmpFilename)
		}
	}(tmpFilename)

	if err := writeBirdConfig(tmpFilename, prefixes); err != nil {
		return err
	}

	// compare new file with original config file
	if compareFiles(tmpFilename, filename) {
		return errConfigIdentical
	}

	// move tmp file to right place
	return os.Rename(tmpFilename, filename)
}

func writeBirdConfig(filename string, prefixes PrefixCollection) error {
	var err error

	// open file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	// prepare content with a header
	output := "# DO NOT EDIT MANUALLY\n"

	// append marshalled prefixsets
	for _, p := range prefixes {
		output += p.Marshal()
	}

	// write data to file
	_, err = f.WriteString(output)

	return err
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
