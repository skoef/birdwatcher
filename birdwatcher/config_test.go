package birdwatcher

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	// test check for valid file
	t.Run("config not found", func(t *testing.T) {
		t.Parallel()

		err := ReadConfig(&Config{}, "testdata/config/filedoesntexists")
		if assert.Error(t, err) {
			assert.Equal(t, "config file testdata/config/filedoesntexists not found", err.Error())
		}
	})

	// read invalid TOML from file and check if it gets detected
	t.Run("invalid toml", func(t *testing.T) {
		t.Parallel()

		err := ReadConfig(&Config{}, "testdata/config/invalidtoml")
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "could not parse config")
			assert.Contains(t, err.Error(), "line 2, column 6")
		}
	})

	// check for error when no services are defined
	t.Run("no services defined", func(t *testing.T) {
		t.Parallel()

		err := ReadConfig(&Config{}, "testdata/config/no_services")
		if assert.Error(t, err) {
			assert.Equal(t, "no services configured", err.Error())
		}
	})

	// check for error for service with no command
	t.Run("service no command", func(t *testing.T) {
		t.Parallel()

		err := ReadConfig(&Config{}, "testdata/config/service_nocommand")
		if assert.Error(t, err) {
			assert.Regexp(t, regexp.MustCompile("^service .+ has no command set"), err.Error())
		}
	})

	// check for error for service with no prefixes
	t.Run("service no prefixes", func(t *testing.T) {
		t.Parallel()

		err := ReadConfig(&Config{}, "testdata/config/service_noprefixes")
		if assert.Error(t, err) {
			assert.Regexp(t, regexp.MustCompile("^service .+ has no prefixes set"), err.Error())
		}
	})

	// check for error for service with invalid prefix
	t.Run("invalid prefix", func(t *testing.T) {
		t.Parallel()

		err := ReadConfig(&Config{}, "testdata/config/service_invalidprefix")
		if assert.Error(t, err) {
			assert.Regexp(t, regexp.MustCompile("^could not parse prefix for service"), err.Error())
		}
	})

	// check for error for service with duplicate prefix
	t.Run("duplicate prefix", func(t *testing.T) {
		t.Parallel()

		err := ReadConfig(&Config{}, "testdata/config/service_duplicateprefix")
		if assert.Error(t, err) {
			assert.Regexp(t, regexp.MustCompile("^duplicate prefix .+ found"), err.Error())
		}
	})

	// read minimal valid config and check defaults
	t.Run("minimal valid config", func(t *testing.T) {
		t.Parallel()

		testConf := Config{}

		err := ReadConfig(&testConf, "testdata/config/minimal")
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, defaultConfigFile, testConf.ConfigFile)
		assert.Equal(t, defaultReloadCommand, testConf.ReloadCommand)
		assert.False(t, testConf.Prometheus.Enabled)
		assert.Equal(t, defaultPrometheusPort, testConf.Prometheus.Port)
		assert.Equal(t, defaultPrometheusPath, testConf.Prometheus.Path)
		assert.Len(t, testConf.Services, 1)
		assert.Equal(t, "foo", testConf.Services["foo"].name)
		assert.Equal(t, defaultCheckInterval, testConf.Services["foo"].Interval)
		assert.Equal(t, defaultFunctionName, testConf.Services["foo"].FunctionName)
		assert.Equal(t, defaultServiceFail, testConf.Services["foo"].Fail)
		assert.Equal(t, defaultServiceRise, testConf.Services["foo"].Rise)
		assert.Equal(t, defaultServiceTimeout, testConf.Services["foo"].Timeout)

		if assert.Len(t, testConf.Services["foo"].prefixes, 1) {
			assert.Equal(t, "192.168.0.0/24", testConf.Services["foo"].prefixes[0].String())
		}

		// check GetServices result
		svcs := testConf.GetServices()
		if assert.Len(t, svcs, 1) {
			assert.Equal(t, "foo", svcs[0].name)
		}
	})

	// read overridden TOML file and check if overrides are picked up
	t.Run("all options overridden", func(t *testing.T) {
		t.Parallel()

		testConf := Config{}

		err := ReadConfig(&testConf, "testdata/config/overridden")
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "/etc/birdwatcher.conf", testConf.ConfigFile)
		assert.Equal(t, "/sbin/birdc configure", testConf.ReloadCommand)
		assert.True(t, testConf.Prometheus.Enabled)
		assert.Equal(t, 1234, testConf.Prometheus.Port)
		assert.Equal(t, "/something", testConf.Prometheus.Path)
		assert.Equal(t, "foo_bar", testConf.Services["foo"].FunctionName)

		if assert.Len(t, testConf.Services["foo"].prefixes, 1) {
			assert.Equal(t, "192.168.0.0/24", testConf.Services["foo"].prefixes[0].String())
		}

		if assert.Len(t, testConf.Services["bar"].prefixes, 2) {
			assert.Equal(t, "192.168.1.0/24", testConf.Services["bar"].prefixes[0].String())
			assert.Equal(t, "fc00::/7", testConf.Services["bar"].prefixes[1].String())
		}

		// check GetServices result
		svcs := testConf.GetServices()
		if assert.Len(t, svcs, 2) {
			// order of the services is not guaranteed
			for _, svc := range svcs {
				switch svc.name {
				case "foo":
					assert.Equal(t, 10, svc.Interval)
					assert.Equal(t, 20, svc.Rise)
					assert.Equal(t, 30, svc.Fail)
					assert.Equal(t, time.Second*40, svc.Timeout)
				case "bar":
				default:
					assert.Fail(t, "unexpected service name", "service name: %s", svc.name)
				}
			}
		}
	})
}
