package birdwatcher

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/BurntSushi/toml"
)

// Config -- struct for holding definitions from configuration file
type Config struct {
	IPv4         protoConfig
	IPv6         protoConfig
	FunctionName string
	Services     map[string]*ServiceCheck
}

type protoConfig struct {
	Enable        bool
	ConfigFile    string
	ReloadCommand string
}

// ReadConfig reads TOML config from given file into given Config or returns
// error on invalid configuration
func ReadConfig(conf *Config, configFile string) error {
	if _, err := os.Stat(configFile); err != nil {
		return fmt.Errorf("config file %s not found", configFile)
	}

	if _, err := toml.DecodeFile(configFile, conf); err != nil {
		return fmt.Errorf("could not parse config: %w", err)
	}

	if !conf.IPv4.Enable && !conf.IPv6.Enable {
		return errors.New("enable either IPv4 or IPv6 or both")
	}

	if conf.IPv4.ConfigFile == "" {
		conf.IPv4.ConfigFile = "/etc/bird/birdwatcher.conf"
	}

	if conf.IPv4.ReloadCommand == "" {
		conf.IPv4.ReloadCommand = "/usr/sbin/birdc configure"
	}

	if conf.IPv6.ConfigFile == "" {
		conf.IPv6.ConfigFile = "/etc/bird/birdwatcher6.conf"
	}

	if conf.IPv6.ReloadCommand == "" {
		conf.IPv6.ReloadCommand = "/usr/sbin/birdc6 configure"
	}

	if len(conf.Services) == 0 {
		return errors.New("no services configured")
	}

	allPrefixes := map[string]bool{}
	for name, s := range conf.Services {
		// copy service name to ServiceCheck
		s.name = name

		if s.FunctionName == "" {
			s.FunctionName = "match_route"
		}

		// validate service
		if err := conf.validateService(s); err != nil {
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

func (c Config) validateService(s *ServiceCheck) error {
	if s.Command == "" {
		return fmt.Errorf("service %s has no command set", s.name)
	}

	if s.Interval <= 0 {
		s.Interval = 1
	}

	if s.Timeout <= 0 {
		s.Timeout = 10
	}

	if s.Fail <= 0 {
		s.Fail = 1
	}

	if s.Rise <= 0 {
		s.Rise = 1
	}

	if len(s.Prefixes) == 0 {
		return fmt.Errorf("service %s has no prefixes set", s.name)
	}

	return nil
}

// GetServices converts the services map into a slice of ServiceChecks and returns it
func (c Config) GetServices() []*ServiceCheck {
	var sc []*ServiceCheck
	for i := range c.Services {
		sc = append(sc, c.Services[i])
	}

	return sc
}
