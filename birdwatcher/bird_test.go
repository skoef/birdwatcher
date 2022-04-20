package birdwatcher

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBirdConfig(t *testing.T) {
	// open tempfile
	tmpFile, err := os.CreateTemp("", "bird_test")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	prefixes := make(PrefixCollection)
	prefixes["match_route"] = NewPrefixSet("match_route")

	// write bird config with empty prefix list
	err = writeBirdConfig(tmpFile.Name(), prefixes)
	require.NoError(t, err)

	// read data from temp file and compare it to file fixture
	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	fixture, err := os.ReadFile("testdata/bird/config_empty")
	require.NoError(t, err)

	assert.Equal(t, fixture, data)

	for _, pref := range []string{"1.2.3.4/32", "2.3.4.5/26", "3.4.5.6/24", "4.5.6.7/21"} {
		_, prf, _ := net.ParseCIDR(pref)
		prefixes["match_route"].Add(*prf)
	}

	// write bird config to it
	err = writeBirdConfig(tmpFile.Name(), prefixes)
	require.NoError(t, err)

	// read data from temp file and compare it to file fixture
	data, err = os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	fixture, err = os.ReadFile("testdata/bird/config")
	require.NoError(t, err)

	assert.Equal(t, fixture, data)
}

func TestBirdCompareFiles(t *testing.T) {
	// open 2 tempfiles
	tmpFileA, err := os.CreateTemp("", "bird_test")
	require.NoError(t, err)
	defer os.Remove(tmpFileA.Name())

	tmpFileB, err := os.CreateTemp("", "bird_test")
	require.NoError(t, err)
	defer os.Remove(tmpFileB.Name())

	// write same string to both files
	_, err = tmpFileA.WriteString("test")
	require.NoError(t, err)
	_, err = tmpFileB.WriteString("test")
	require.NoError(t, err)

	assert.True(t, compareFiles(tmpFileA.Name(), tmpFileB.Name()))

	// write something else to one file
	_, err = tmpFileB.WriteString("test123")
	require.NoError(t, err)

	assert.False(t, compareFiles(tmpFileA.Name(), tmpFileB.Name()))
}
