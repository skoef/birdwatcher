package birdwatcher

import (
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBirdConfig(t *testing.T) {

	// open tempfile
	tmpFile, err := ioutil.TempFile("", "bird_test")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// write bird config with empty prefix list
	err = writeBirdConfig(tmpFile.Name(), "match_route", []net.IPNet{})
	require.NoError(t, err)

	// read data from temp file and compare it to file fixture
	data, err := ioutil.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	fixture, err := ioutil.ReadFile("testdata/bird/config_empty")
	require.NoError(t, err)

	assert.Equal(t, fixture, data)

	// write bird config to it
	err = writeBirdConfig(tmpFile.Name(), "match_route", []net.IPNet{
		{IP: net.IP{1, 2, 3, 4}, Mask: net.IPMask{255, 255, 255, 255}},
		{IP: net.IP{2, 3, 4, 5}, Mask: net.IPMask{255, 255, 255, 192}},
		{IP: net.IP{3, 4, 5, 6}, Mask: net.IPMask{255, 255, 255, 0}},
		{IP: net.IP{4, 5, 6, 7}, Mask: net.IPMask{255, 255, 248, 0}},
	})
	require.NoError(t, err)

	// read data from temp file and compare it to file fixture
	data, err = ioutil.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	fixture, err = ioutil.ReadFile("testdata/bird/config")
	require.NoError(t, err)

	assert.Equal(t, fixture, data)
}
