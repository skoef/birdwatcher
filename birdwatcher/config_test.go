package birdwatcher

import (
	"net"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	var err error
	var testConf Config

	// test check for valid file
	fixFile := "testdata/config/filedoesntexists"
	testConf = Config{}
	err = ReadConfig(&testConf, fixFile)
	if assert.Error(t, err) {
		assert.Equal(t, "config file testdata/config/filedoesntexists not found", err.Error())
	}

	// read invalid TOML from file and check if it gets detected
	err = ReadConfig(&testConf, "testdata/config/invalidtoml")
	if assert.Error(t, err) {
		assert.Regexp(t, regexp.MustCompile("^could not parse config:"), err.Error())
	}

	// read minimal TOML and check defaults
	err = ReadConfig(&testConf, "testdata/config/minimal")
	assert.NoError(t, err)
	assert.Equal(t, "/etc/bird/birdwatcher.conf", testConf.IPv4.ConfigFile)
	assert.Equal(t, false, testConf.IPv4.Enable)
	assert.Equal(t, "match_route", testConf.IPv4.FunctionName)
	assert.Equal(t, "/usr/sbin/birdc configure", testConf.IPv4.ReloadCommand)
	assert.Equal(t, "/etc/bird/birdwatcher6.conf", testConf.IPv6.ConfigFile)
	assert.Equal(t, false, testConf.IPv6.Enable)
	assert.Equal(t, "match_route", testConf.IPv6.FunctionName)
	assert.Equal(t, "/usr/sbin/birdc6 configure", testConf.IPv6.ReloadCommand)
	assert.Equal(t, 1, len(testConf.Services))
	assert.Equal(t, "foo", testConf.Services["foo"].name)
	assert.Equal(t, 1, len(testConf.Services["foo"].prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{192, 168, 0, 0}, Mask: net.IPMask{255, 255, 255, 0}}, testConf.Services["foo"].prefixes[0])

	// read overridden TOML file and check if overrides are picked up
	err = ReadConfig(&testConf, "testdata/config/overriden")
	assert.NoError(t, err)
	assert.Equal(t, "/etc/birdwatcher.conf", testConf.IPv4.ConfigFile)
	assert.Equal(t, true, testConf.IPv4.Enable)
	assert.Equal(t, "bar_foo", testConf.IPv4.FunctionName)
	assert.Equal(t, "/sbin/birdc configure", testConf.IPv4.ReloadCommand)
	assert.Equal(t, "/birdwatcher6.conf", testConf.IPv6.ConfigFile)
	assert.Equal(t, false, testConf.IPv6.Enable)
	assert.Equal(t, "foo_bar", testConf.IPv6.FunctionName)
	assert.Equal(t, "/usr/bin/birdc6 configure", testConf.IPv6.ReloadCommand)
	assert.Equal(t, 2, len(testConf.Services["bar"].prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{192, 168, 1, 0}, Mask: net.IPMask{255, 255, 255, 0}}, testConf.Services["bar"].prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{192, 168, 2, 0}, Mask: net.IPMask{255, 255, 255, 128}}, testConf.Services["bar"].prefixes[1])
}
