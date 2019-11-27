package birdwatcher

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
)

type BirdConfig struct {
	EnableIPv4   bool
	EnableIPv6   bool
	ConfigIPv4   string
	ConfigIPv6   string
	FunctionName string
}

var (
	errConfigIdentical = errors.New("configuration file is identical")
)

func updateBirdConfig(filename, functionName string, prefixes []net.IPNet) error {
	// write config to temp file
	tmpFilename := fmt.Sprintf("%s.tmp", filename)
	if err := writeBirdConfig(tmpFilename, functionName, prefixes); err != nil {
		return err
	}

	// compare new file with original config file
	if compareFiles(tmpFilename, filename) {
		return errConfigIdentical
	}

	// move tmp file to right place
	return os.Rename(tmpFilename, filename)
}

func writeBirdConfig(filename, functionName string, prefixes []net.IPNet) error {
	var err error

	// open file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	// prepare content with a header
	output := "# DO NOT EDIT MANUALLY\n"

	// set function name
	output += fmt.Sprintf("function %s()\n{\n\treturn ", functionName)

	if len(prefixes) == 0 {
		output += "false;\n"
	} else {

		// begin array
		output += "net ~ [\n"

		// add all prefixes on single lines
		suffix := ","
		for i, p := range prefixes {
			if i == len(prefixes)-1 {
				suffix = ""
			}
			output += fmt.Sprintf("\t\t%s%s\n", p.String(), suffix)
		}

		// end array
		output += "\t];\n"
	}

	// add footer
	output += "}\n"

	// write data to file
	_, err = f.WriteString(output)
	return err
}

func compareFiles(a, b string) bool {
	data, err := ioutil.ReadFile(a)
	if err != nil {
		return false
	}

	datb, err := ioutil.ReadFile(b)
	if err != nil {
		return false
	}

	return bytes.Equal(data, datb)
}
