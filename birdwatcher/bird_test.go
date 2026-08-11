package birdwatcher

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBirdConfig(t *testing.T) {
	t.Parallel()

	t.Run("filter down", func(t *testing.T) {
		t.Parallel()

		// open tempfile
		tmpFile, err := os.CreateTemp(t.TempDir(), "bird_test")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// no prefixes, service unhealthy
		prefixes := make(PrefixCollection)
		prefixes["match_route"] = NewPrefixSet(&ServiceCheck{FunctionName: "match_route"})

		// write bird config with empty prefix list and unhealthy
		err = writeBirdConfig(tmpFile.Name(), prefixes, false)
		require.NoError(t, err)

		// read data from temp file and compare it to file fixture
		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		fixture, err := os.ReadFile("testdata/bird/filter_down")
		require.NoError(t, err)

		assert.Equal(t, string(fixture), string(data))

		// switch to compat mode and do it again
		err = writeBirdConfig(tmpFile.Name(), prefixes, true)
		require.NoError(t, err)

		// read data from temp file and compare it to file fixture
		data, err = os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		fixture, err = os.ReadFile("testdata/bird/filter_down_compat")
		require.NoError(t, err)

		assert.Equal(t, string(fixture), string(data))
	})

	t.Run("one prefixset", func(t *testing.T) {
		t.Parallel()

		// open tempfile
		tmpFile, err := os.CreateTemp(t.TempDir(), "bird_test")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// one healthy prefix, service assumed healthy
		prefixes := make(PrefixCollection)
		prefixes["match_route"] = NewPrefixSet(&ServiceCheck{FunctionName: "match_route"})

		for _, pref := range []string{"1.2.3.4/32", "2.3.4.5/26", "3.4.5.6/24", "4.5.6.7/21"} {
			_, prf, _ := net.ParseCIDR(pref)
			prefixes["match_route"].Add(*prf)
		}

		// write bird config to it
		err = writeBirdConfig(tmpFile.Name(), prefixes, false)
		require.NoError(t, err)

		// read data from temp file and compare it to file fixture
		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		fixture, err := os.ReadFile("testdata/bird/filter_one_prefix")
		require.NoError(t, err)

		assert.Equal(t, string(fixture), string(data))

		// switch to compat mode and do it again
		// write bird config to it
		err = writeBirdConfig(tmpFile.Name(), prefixes, true)
		require.NoError(t, err)

		// read data from temp file and compare it to file fixture
		data, err = os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		fixture, err = os.ReadFile("testdata/bird/filter_one_prefix_compat")
		require.NoError(t, err)

		assert.Equal(t, string(fixture), string(data))
	})

	t.Run("filter up", func(t *testing.T) {
		t.Parallel()

		// open tempfile
		tmpFile, err := os.CreateTemp(t.TempDir(), "bird_test")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// no healthy prefixes, service explicitly healthy
		prefixes := make(PrefixCollection)
		prefixes["match_route"] = NewPrefixSet(&ServiceCheck{
			FunctionName: "match_route",
			state:        ServiceStateUp,
		})

		// write bird config to it
		err = writeBirdConfig(tmpFile.Name(), prefixes, false)
		require.NoError(t, err)

		// read data from temp file and compare it to file fixture
		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		fixture, err := os.ReadFile("testdata/bird/filter_up")
		require.NoError(t, err)

		assert.Equal(t, string(fixture), string(data))

		// switch to compat mode and do it again
		// write bird config to it
		err = writeBirdConfig(tmpFile.Name(), prefixes, true)
		require.NoError(t, err)

		// read data from temp file and compare it to file fixture
		data, err = os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		fixture, err = os.ReadFile("testdata/bird/filter_up_compat")
		require.NoError(t, err)

		assert.Equal(t, string(fixture), string(data))
	})
}

func TestPrefixPad(t *testing.T) {
	t.Parallel()

	prefixes := make([]net.IPNet, 4)

	for i, pref := range []string{"1.2.3.0/24", "2.3.4.0/24", "3.4.5.0/24", "3.4.5.0/26"} {
		_, prf, _ := net.ParseCIDR(pref)
		prefixes[i] = *prf
	}

	padded := prefixPad(prefixes)
	assert.Equal(t, "1.2.3.0/24,2.3.4.0/24,3.4.5.0/24,3.4.5.0/26", strings.Join(padded, ""))
}

func TestBirdCompareFiles(t *testing.T) {
	t.Parallel()

	// open 2 tempfiles
	tmpFileA, err := os.CreateTemp(t.TempDir(), "bird_test")
	require.NoError(t, err)
	defer os.Remove(tmpFileA.Name())

	tmpFileB, err := os.CreateTemp(t.TempDir(), "bird_test")
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
