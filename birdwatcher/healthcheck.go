package birdwatcher

import (
	"bytes"
	"context"
	"net"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

type HealthCheck struct {
	stopped  chan interface{}
	actions  chan *Action
	prefixes []net.IPNet
	Config   Config
}

func NewHealthCheck(c Config) HealthCheck {
	h := HealthCheck{}
	h.Config = c

	return h
}

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

func (h *HealthCheck) handleAction(action *Action) {
	for _, p := range action.Prefixes {
		if action.State == ServiceStateUp {
			h.addPrefix(p)
		} else if action.State == ServiceStateDown {
			h.removePrefix(p)
		} else {
			log.WithFields(log.Fields{
				"state":   action.State,
				"service": action.Service.name,
			}).Warning("unhandled state received")
			return
		}
	}

	if h.Config.IPv4.Enable {
		h.applyConfig(h.Config.IPv4, func(f []net.IPNet) []net.IPNet {
			r := make([]net.IPNet, 0)
			for _, p := range f {
				if p.IP.To4().Equal(p.IP) {
					r = append(r, p)
				}
			}
			return r
		}(h.prefixes))
	}

	if h.Config.IPv6.Enable {
		h.applyConfig(h.Config.IPv6, func(f []net.IPNet) []net.IPNet {
			r := make([]net.IPNet, 0)
			for _, p := range f {
				if len(p.IP) == net.IPv6len {
					r = append(r, p)
				}
			}
			return r
		}(h.prefixes))
	}
}

func (h *HealthCheck) applyConfig(config protoConfig, prefixes []net.IPNet) error {
	var err error
	// update bird config
	err = updateBirdConfig(config.ConfigFile, config.FunctionName, prefixes)
	if err != nil {
		if err == errConfigIdentical {
			log.WithFields(log.Fields{
				"file": config.ConfigFile,
			}).Warning("config did not change")
		} else {
			log.WithFields(log.Fields{
				"file":  config.ConfigFile,
				"error": err.Error(),
			}).Warning("error updating configuration")
		}

		return err
	}

	log.WithFields(log.Fields{
		"file":    config.ConfigFile,
		"command": config.ReloadCommand,
	}).Info("prefixes updated, reloading")

	reloadTimeout := 10 * time.Second

	// issue reload command, with some reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), reloadTimeout)
	defer cancel()

	// set up command execution within that context
	cmd := exec.CommandContext(ctx, config.ReloadCommand)

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
		}).Warning("reloading failed")
	} else {
		log.WithFields(log.Fields{
			"command": config.ReloadCommand,
		}).Debug("reloading succeeded")
	}

	return err
}

func (h *HealthCheck) addPrefix(prefix net.IPNet) {
	log.WithFields(log.Fields{
		"prefix": prefix,
	}).Debug("adding prefix to global list")

	// skip prefix if it's already in the list
	// shouldn't really happen though
	for _, p := range h.prefixes {
		if p.IP.Equal(prefix.IP) && bytes.Equal(p.Mask, prefix.Mask) {
			log.WithFields(log.Fields{
				"prefix": prefix,
			}).Warn("duplicate prefix, skipping")
			return
		}
	}

	// add prefix to the global prefix list
	h.prefixes = append(h.prefixes, prefix)
}

func (h *HealthCheck) removePrefix(prefix net.IPNet) {
	log.WithFields(log.Fields{
		"prefix": prefix,
	}).Debug("removing prefix from global list")

	// go over global prefix list and remove it when found
	for i, p := range h.prefixes {
		if p.IP.Equal(prefix.IP) && bytes.Equal(p.Mask, prefix.Mask) {
			// remove entry from slice, fast approach
			h.prefixes[i] = h.prefixes[len(h.prefixes)-1] // copy last element to index i
			//h.prefixes[len(h.prefixes)-1] = nil // erase last element
			h.prefixes = h.prefixes[:len(h.prefixes)-1] // truncate slice
			return
		}
	}

	log.WithFields(log.Fields{
		"prefix": prefix,
	}).Warn("prefix not found in list, skipping")
}

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
