package birdwatcher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

const (
	// size of the channels service checks push their events on
	actionsChannelSize = 16
	// timeout when reloading bird
	reloadTimeout = 10 * time.Second
)

var prefixStateMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "birdwatcher",
	Subsystem: "prefix",
	Name:      "state",
	Help:      "Current health state per prefix",
}, []string{"service", "prefix"})

// HealthCheck -- struct holding everything needed for the never-ending health
// check loop
type HealthCheck struct {
	stopped        chan any
	actions        chan *Action
	services       []*ServiceCheck
	prefixes       PrefixCollection
	Config         Config
	reloadedBefore bool
}

// NewHealthCheck returns a HealthCheck with given configuration
func NewHealthCheck(c Config) HealthCheck {
	h := HealthCheck{}
	h.Config = c

	return h
}

// Start starts the process of health checking the services and handling
// Actions that come from them
func (h *HealthCheck) Start(services []*ServiceCheck, ready chan<- bool, status chan string) {
	// copy reference to services
	h.services = services
	// create channel for service check to push there events on
	h.actions = make(chan *Action, actionsChannelSize)
	// create a channel to signal we're stopping
	h.stopped = make(chan any)

	// start each service and keep a pointer to the services
	// we'll need this later to stop them
	for _, s := range services {
		log.WithFields(log.Fields{
			"service": s.Name(),
		}).Info("starting service check")

		go s.Start(&h.actions)
	}

	ready <- true

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
			}).Debug("incoming action")

			h.handleAction(action, status)
		}
	}
}

func (h *HealthCheck) didReloadBefore() bool {
	return h.reloadedBefore
}

func (h *HealthCheck) handleAction(action *Action, status chan string) {
	for _, p := range action.Prefixes {
		switch action.State {
		case ServiceStateUp:
			h.addPrefix(action.Service, p)
		case ServiceStateDown:
			h.removePrefix(action.Service, p)
		default:
			log.WithFields(log.Fields{
				"state":   action.State,
				"service": action.Service.name,
			}).Warning("unhandled state received")

			return
		}
	}

	// gather data for a status update
	su := h.statusUpdate()
	log.WithField("status", su).Debug("status update")
	// send update over channel
	status <- su

	if err := h.applyConfig(h.Config, h.prefixes); err != nil {
		log.WithError(err).Error("could not apply BIRD config")
	}
}

// statusUpdate returns a string with a situational report on how many services
// are configured up
func (h *HealthCheck) statusUpdate() string {
	servicesDown := []string{}

	for _, s := range h.services {
		if s.IsUp() {
			continue
		}

		servicesDown = append(servicesDown, s.Name())
	}

	allServices := len(h.services)

	var status string

	switch {
	case len(servicesDown) == 0:
		status = fmt.Sprintf("all %d service(s) up", allServices)
	case len(servicesDown) == allServices:
		status = fmt.Sprintf("all %d service(s) down", allServices)
	default:
		status = fmt.Sprintf("service(s) %s down, %d service(s) up",
			strings.Join(servicesDown, ","), allServices-len(servicesDown))
	}

	return status
}

func (h *HealthCheck) applyConfig(config Config, prefixes PrefixCollection) error {
	cLog := log.WithFields(log.Fields{
		"file": config.ConfigFile,
	})

	// update bird config
	err := updateBirdConfig(config, prefixes)
	if err != nil {
		// if config did not change, we should still reload if we don't know the
		// state of BIRD
		if errors.Is(err, errConfigIdentical) {
			if h.didReloadBefore() {
				cLog.Warning("config did not change, not reloading")

				return nil
			}

			cLog.Info("config did not change, but reloading anyway")
		} else {
			// break on any other error
			cLog.WithError(err).Warning("error updating configuration")

			return err
		}
	}

	cLog = log.WithFields(log.Fields{
		"command": config.ReloadCommand,
	})
	cLog.Info("prefixes updated, reloading")

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
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		cLog.WithField("timeout", reloadTimeout).Warning("reloading timed out")

		return ctx.Err()
	}

	if err != nil {
		cLog.WithError(err).WithField("output", output).Warning("reloading failed")
	} else {
		cLog.Debug("reloading succeeded")

		// mark successful reload
		h.reloadedBefore = true
	}

	return err
}

func (h *HealthCheck) addPrefix(svc *ServiceCheck, prefix net.IPNet) {
	h.ensurePrefixSet(svc.FunctionName)

	h.prefixes[svc.FunctionName].Add(prefix)
	prefixStateMetric.WithLabelValues(svc.Name(), prefix.String()).Set(1.0)
}

func (h *HealthCheck) removePrefix(svc *ServiceCheck, prefix net.IPNet) {
	h.ensurePrefixSet(svc.FunctionName)

	h.prefixes[svc.FunctionName].Remove(prefix)
	prefixStateMetric.WithLabelValues(svc.Name(), prefix.String()).Set(0.0)
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
func (h *HealthCheck) Stop() {
	// signal each service to stop
	for _, s := range h.services {
		log.WithFields(log.Fields{
			"service": s.Name(),
		}).Info("stopping service check")

		s.Stop()
	}

	h.stopped <- true
}
