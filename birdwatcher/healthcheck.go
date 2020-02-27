package birdwatcher

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// HealthCheck -- struct holding everything needed for the never-ending health
// check loop
type HealthCheck struct {
	stopped  chan interface{}
	actions  chan *Action
	prefixes PrefixCollection
	Config   Config
	reloads  map[string]bool
}

// NewHealthCheck returns a HealthCheck with given configuration
func NewHealthCheck(c Config) HealthCheck {
	h := HealthCheck{}
	h.Config = c
	h.reloads = make(map[string]bool)

	return h
}

// Start starts the process of health checking the services and handling
// Actions that come from them
func (h *HealthCheck) Start(services []*ServiceCheck) {
	// create channel for service check to push there events on
	h.actions = make(chan *Action, 16)
	// create a channel to signal we're stopping
	h.stopped = make(chan interface{})

	// start each service and keep a pointer to the services
	// we'll need this later to stop them
	for _, s := range services {
		log.WithFields(log.Fields{
			"service": s.name,
		}).Info("Starting service check")

		go s.Start(&h.actions)
	}

	// mean while process incoming actions from the channel
	for {
		select {
		case <-h.stopped:
			log.Debug("received stop signal")
			// we're done
			return
		case action := <-h.actions:
			log.WithFields(log.Fields{
				"service": action.Service.name,
				"state":   action.State,
			}).Debug("Incoming action")

			h.handleAction(action)
		}
	}
}

func (h *HealthCheck) didReloadBefore(protocol PrefixFamily) bool {
	reloaded, found := h.reloads[string(protocol)]
	return (reloaded && found)
}

func (h *HealthCheck) handleAction(action *Action) {
	for _, p := range action.Prefixes {
		if action.State == ServiceStateUp {
			h.addPrefix(action.Service.FunctionName, p)
		} else if action.State == ServiceStateDown {
			h.removePrefix(action.Service.FunctionName, p)
		} else {
			log.WithFields(log.Fields{
				"state":   action.State,
				"service": action.Service.name,
			}).Warning("unhandled state received")
			return
		}
	}

	if h.Config.IPv4.Enable {
		h.applyConfig(PrefixFamilyIPv4, h.Config.IPv4, h.prefixes)
	}

	if h.Config.IPv6.Enable {
		h.applyConfig(PrefixFamilyIPv6, h.Config.IPv6, h.prefixes)
	}
}

func (h *HealthCheck) applyConfig(protocol PrefixFamily, config protoConfig, prefixes PrefixCollection) error {
	var err error
	// update bird config
	err = updateBirdConfig(config.ConfigFile, protocol, prefixes)
	if err != nil {
		// if config did not change, we should still reload if we don't know the
		// state of BIRD
		if err == errConfigIdentical {
			if h.didReloadBefore(protocol) {
				log.WithFields(log.Fields{
					"file": config.ConfigFile,
				}).Warning("config did not change, not reloading")

				return err
			}

			log.WithFields(log.Fields{
				"file": config.ConfigFile,
			}).Info("config did not change, but reloading anyway")

			// break on any other error
		} else {
			log.WithFields(log.Fields{
				"file":  config.ConfigFile,
				"error": err.Error(),
			}).Warning("error updating configuration")

			return err
		}
	}

	log.WithFields(log.Fields{
		"file":    config.ConfigFile,
		"command": config.ReloadCommand,
	}).Info("prefixes updated, reloading")

	reloadTimeout := 10 * time.Second

	// issue reload command, with some reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), reloadTimeout)
	defer cancel()

	// split reload command into command/args assuming the first part is the command
	// and the rest are the arguments
	commandArgs := strings.Split(config.ReloadCommand, " ")

	// set up command execution within that context
	cmd := exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...)

	// get exit code of command
	output, err := cmd.Output()

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if ctx.Err() == context.DeadlineExceeded {
		log.WithFields(log.Fields{
			"command": config.ReloadCommand,
			"timeout": reloadTimeout,
		}).Warning("reloading timed out")

		return ctx.Err()
	}

	if err != nil {
		log.WithFields(log.Fields{
			"command": config.ReloadCommand,
			"output":  output,
			"error":   err.Error(),
		}).Warning("reloading failed")
	} else {
		log.WithFields(log.Fields{
			"command": config.ReloadCommand,
		}).Debug("reloading succeeded")

		// mark succesful reload
		h.reloads[string(protocol)] = true
	}

	return err
}

func (h *HealthCheck) addPrefix(functionName string, prefix net.IPNet) {
	h.ensurePrefixSet(functionName)

	h.prefixes[functionName].Add(prefix)
}

func (h *HealthCheck) removePrefix(functionName string, prefix net.IPNet) {
	h.ensurePrefixSet(functionName)

	h.prefixes[functionName].Remove(prefix)
}

func (h *HealthCheck) ensurePrefixSet(functionName string) {
	// make sure the top level map is prepared
	if h.prefixes == nil {
		h.prefixes = make(PrefixCollection)
	}

	// make sure a mapping for this function name exists
	if _, found := h.prefixes[functionName]; !found {
		h.prefixes[functionName] = NewPrefixSet(functionName)
	}
}

// Stop signals all servic checks to stop as well and then stops itself
func (h *HealthCheck) Stop(services []*ServiceCheck) {
	// signal each service to stop
	for _, s := range services {
		log.WithFields(log.Fields{
			"service": s.name,
		}).Info("Stopping service check")

		s.Stop()
	}

	h.stopped <- true
}
