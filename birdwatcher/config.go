package birdwatcher

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds definitions from configuration file
type Config struct {
	ConfigFile    string
	ReloadCommand string
	Prometheus    PrometheusConfig
	Services      map[string]*ServiceCheck
}

// PrometheusConfig holds configuration related to prometheus
type PrometheusConfig struct {
	Enabled bool
	Port    int
	Path    string
}

const (
	defaultConfigFile     = "/etc/bird/birdwatcher.conf"
	defaultReloadCommand  = "/usr/sbin/birdc configure"
	defaultPrometheusPort = 9091
	defaultPrometheusPath = "/metrics"

	defaultFunctionName   = "match_route"
	defaultCheckInterval  = 1
	defaultServiceTimeout = 10 * time.Second
	defaultServiceFail    = 1
	defaultServiceRise    = 1
)

// ReadConfig reads TOML config from given file into given Config or returns
// error on invalid configuration
func ReadConfig(conf *Config, configFile string) error {
	if _, err := os.Stat(configFile); err != nil {
		return fmt.Errorf("config file %s not found", configFile)
	}

	if _, err := toml.DecodeFile(configFile, conf); err != nil {
		errMsg := err.Error()

		var parseErr toml.ParseError

		if errors.As(err, &parseErr) {
			errMsg = parseErr.ErrorWithPosition()
		}

		return fmt.Errorf("could not parse config: %s", errMsg)
	}

	if conf.ConfigFile == "" {
		conf.ConfigFile = defaultConfigFile
	}

	if conf.ReloadCommand == "" {
		conf.ReloadCommand = defaultReloadCommand
	}

	if conf.Prometheus.Path == "" {
		conf.Prometheus.Path = defaultPrometheusPath
	}

	if conf.Prometheus.Port == 0 {
		conf.Prometheus.Port = defaultPrometheusPort
	}

	if len(conf.Services) == 0 {
		return errors.New("no services configured")
	}

	allPrefixes := map[string]bool{}

	for name, s := range conf.Services {
		// copy service name to ServiceCheck
		s.name = name

		if s.FunctionName == "" {
			s.FunctionName = defaultFunctionName
		}

		// validate service
		if err := validateService(s); err != nil {
			return err
		}

		// convert all prefixes into ipnets
		s.prefixes = make([]net.IPNet, len(s.Prefixes))
		for i, p := range s.Prefixes {
			_, ipn, err := net.ParseCIDR(p)
			if err != nil {
				return fmt.Errorf("could not parse prefix for service %s: %w", name, err)
			}

			s.prefixes[i] = *ipn

			// validate whether the prefixes overlap
			if _, found := allPrefixes[ipn.String()]; found {
				return fmt.Errorf("duplicate prefix %s found", ipn.String())
			}

			allPrefixes[ipn.String()] = true
		}

		// map name to each search
		conf.Services[name] = s
	}

	return nil
}

func validateService(s *ServiceCheck) error {
	if s.Command == "" {
		return fmt.Errorf("service %s has no command set", s.name)
	}

	if s.Interval <= 0 {
		s.Interval = defaultCheckInterval
	}

	if s.Timeout <= 0 {
		s.Timeout = defaultServiceTimeout
	}

	if s.Fail <= 0 {
		s.Fail = defaultServiceFail
	}

	if s.Rise <= 0 {
		s.Rise = defaultServiceRise
	}

	if len(s.Prefixes) == 0 {
		return fmt.Errorf("service %s has no prefixes set", s.name)
	}

	return nil
}

// GetServices converts the services map into a slice of ServiceChecks and returns it
func (c Config) GetServices() []*ServiceCheck {
	sc := make([]*ServiceCheck, len(c.Services))
	j := 0

	for i := range c.Services {
		sc[j] = c.Services[i]
		j++
	}

	return sc
}
