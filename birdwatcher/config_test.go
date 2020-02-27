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
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/invalidtoml")
	if assert.Error(t, err) {
		assert.Regexp(t, regexp.MustCompile("^could not parse config:"), err.Error())
	}

	// check for error when no services are defined
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/no_protocols")
	if assert.Error(t, err) {
		assert.Equal(t, "enable either IPv4 or IPv6 or both", err.Error())
	}

	// check for error when no services are defined
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/no_services")
	if assert.Error(t, err) {
		assert.Equal(t, "no services configured", err.Error())
	}

	// check for error for service with no command
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/service_nocommand")
	if assert.Error(t, err) {
		assert.Regexp(t, regexp.MustCompile("^service .+ has no command set"), err.Error())
	}

	// check for error for service with no prefixes
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/service_noprefixes")
	if assert.Error(t, err) {
		assert.Regexp(t, regexp.MustCompile("^service .+ has no prefixes set"), err.Error())
	}

	// check for error for service with invalid prefix
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/service_invalidprefix")
	if assert.Error(t, err) {
		assert.Regexp(t, regexp.MustCompile("^could not parse prefix for service"), err.Error())
	}

	// check for error for service with duplicate prefix
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/service_duplicateprefix")
	if assert.Error(t, err) {
		assert.Regexp(t, regexp.MustCompile("^duplicate prefix .+ found"), err.Error())
	}

	// read minimal valid config and check defaults
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/minimal")
	assert.NoError(t, err)
	assert.Equal(t, "/etc/bird/birdwatcher.conf", testConf.IPv4.ConfigFile)
	assert.Equal(t, true, testConf.IPv4.Enable)
	assert.Equal(t, "/usr/sbin/birdc configure", testConf.IPv4.ReloadCommand)
	assert.Equal(t, "/etc/bird/birdwatcher6.conf", testConf.IPv6.ConfigFile)
	assert.Equal(t, false, testConf.IPv6.Enable)
	assert.Equal(t, "/usr/sbin/birdc6 configure", testConf.IPv6.ReloadCommand)
	assert.Equal(t, 1, len(testConf.Services))
	assert.Equal(t, "foo", testConf.Services["foo"].name)
	assert.Equal(t, 1, testConf.Services["foo"].Interval)
	assert.Equal(t, "match_route", testConf.Services["foo"].FunctionName)
	assert.Equal(t, 1, testConf.Services["foo"].Fail)
	assert.Equal(t, 1, testConf.Services["foo"].Rise)
	assert.Equal(t, 10, testConf.Services["foo"].Timeout)
	assert.Equal(t, 1, len(testConf.Services["foo"].prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{192, 168, 0, 0}, Mask: net.IPMask{255, 255, 255, 0}}, testConf.Services["foo"].prefixes[0])

	// check GetServices result
	svcs := testConf.GetServices()
	assert.Equal(t, 1, len(svcs))
	assert.Equal(t, "foo", svcs[0].name)

	// read overridden TOML file and check if overrides are picked up
	testConf = Config{}
	err = ReadConfig(&testConf, "testdata/config/overridden")
	assert.NoError(t, err)
	assert.Equal(t, "/etc/birdwatcher.conf", testConf.IPv4.ConfigFile)
	assert.Equal(t, true, testConf.IPv4.Enable)
	assert.Equal(t, "/sbin/birdc configure", testConf.IPv4.ReloadCommand)
	assert.Equal(t, "/birdwatcher6.conf", testConf.IPv6.ConfigFile)
	assert.Equal(t, false, testConf.IPv6.Enable)
	assert.Equal(t, "/usr/bin/birdc6 configure", testConf.IPv6.ReloadCommand)
	assert.Equal(t, "foo_bar", testConf.Services["foo"].FunctionName)
	assert.Equal(t, 2, len(testConf.Services["bar"].prefixes))
	assert.Equal(t, net.IPNet{IP: net.IP{192, 168, 1, 0}, Mask: net.IPMask{255, 255, 255, 0}}, testConf.Services["bar"].prefixes[0])
	assert.Equal(t, net.IPNet{IP: net.IP{192, 168, 2, 0}, Mask: net.IPMask{255, 255, 255, 128}}, testConf.Services["bar"].prefixes[1])
}
