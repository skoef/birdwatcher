package birdwatcher

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

var (
	serviceInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "birdwatcher",
		Subsystem: "service",
		Name:      "info",
		Help:      "Services and their configuration",
	}, []string{"service", "function_name", "command", "interval", "timeout", "rise", "fail"})

	serviceCheckDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "birdwatcher",
		Subsystem: "service",
		Name:      "check_duration",
		Help:      "Service check duration in milliseconds",
	}, []string{"service"})

	serviceStateMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "birdwatcher",
		Subsystem: "service",
		Name:      "state",
		Help:      "Current health state per service",
	}, []string{"service"})

	serviceTransitionMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "birdwatcher",
		Subsystem: "service",
		Name:      "transition_total",
		Help:      "Number of transitions per service",
	}, []string{"service"})

	serviceSuccessMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "birdwatcher",
		Subsystem: "service",
		Name:      "success_total",
		Help:      "Number of successful probes per service",
	}, []string{"service"})

	serviceFailMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "birdwatcher",
		Subsystem: "service",
		Name:      "fail_total",
		Help:      "Number of failed probes per service",
	}, []string{"service"})

	serviceTimeoutMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "birdwatcher",
		Subsystem: "service",
		Name:      "timeout_total",
		Help:      "Number of timed out probes per service",
	}, []string{"service"})
)

// ServiceState represents the state the service is considered to be in
type ServiceState string

const (
	// ServiceStateDown considers the service to be down
	ServiceStateDown ServiceState = "down"
	// ServiceStateUp considers the service to be up
	ServiceStateUp ServiceState = "up"
)

// ServiceCheck is the struct for holding all information and state about a
// specific service health check
type ServiceCheck struct {
	name         string
	FunctionName string
	Command      string
	Interval     int
	Timeout      time.Duration
	Fail         int
	Rise         int
	Prefixes     []string
	//nolint:revive // these prefixes are converted into net.IPNet
	prefixes           []net.IPNet
	state              ServiceState
	disablePrefixCheck bool
	stopped            chan any
}

// Start starts the process of health checking its service and sends actions to
// the action channel when service state changes
//
//nolint:funlen // we should refactor this a bit
func (s *ServiceCheck) Start(action *chan *Action) {
	s.stopped = make(chan any)
	ticker := time.NewTicker(time.Second * time.Duration(s.Interval))

	var err error

	upCounter := 0
	downCounter := 0

	sLog := log.WithFields(log.Fields{
		"service": s.name,
		"command": s.Command,
	})

	// set service info metric
	serviceInfoMetric.With(prometheus.Labels{
		"service":       s.name,
		"function_name": s.FunctionName,
		"command":       s.Command,
		"interval":      strconv.Itoa(s.Interval),
		"timeout":       s.Timeout.String(),
		"rise":          strconv.Itoa(s.Rise),
		"fail":          strconv.Itoa(s.Fail),
	}).Set(1.0)

	for {
		select {
		case <-s.stopped:
			sLog.Debug("received stop signal")
			// we're done
			return

		case <-ticker.C:
			beginCheck := time.Now()
			// perform check synchronously to prevent checks to queue
			err = s.performCheck()
			// keep track of the time it took for the check to perform
			serviceCheckDuration.WithLabelValues(s.name).Set(float64(time.Since(beginCheck)))

			// based on the check result, decide if we're going up or down
			//
			// check gave positive result
			if err == nil {
				// reset downCounter
				downCounter = 0

				// update success metric
				serviceSuccessMetric.WithLabelValues(s.name).Inc()

				sLog.Debug("check command exited without error")

				// are we up enough to consider service to be healthy
				if upCounter >= (s.Rise - 1) {
					if s.state != ServiceStateUp {
						sLog.WithFields(log.Fields{
							"successes": upCounter,
						}).Info("service transitioning to up")

						// mark current state as up
						s.state = ServiceStateUp

						// update state metric
						serviceStateMetric.WithLabelValues(s.name).Set(1)
						// update transition metric
						serviceTransitionMetric.WithLabelValues(s.name).Inc()

						// send action on channel
						*action <- s.getAction()
					}
				} else {
					// or are we still in the process of coming up
					upCounter++

					sLog.WithFields(log.Fields{
						"successes": upCounter,
					}).Debug("service moving towards up")
				}
			} else {
				// check gave negative result
				//
				// reset upcounter
				upCounter = 0

				// update success metric
				serviceFailMetric.WithLabelValues(s.name).Inc()
				// if this was a timeout, increment that counter as well
				if errors.Is(err, context.DeadlineExceeded) {
					serviceTimeoutMetric.WithLabelValues(s.name).Inc()
					sLog.Debug("check command timed out")
				} else {
					sLog.Debug("check command failed")
				}

				// are we down long enough to consider service down
				if downCounter >= (s.Fail - 1) {
					if s.state != ServiceStateDown {
						sLog.WithFields(log.Fields{
							"failures": downCounter,
						}).Info("service transitioning to down")

						// mark current state as down
						s.state = ServiceStateDown

						// update state metric
						serviceStateMetric.WithLabelValues(s.name).Set(0)
						// update transition metric
						serviceTransitionMetric.WithLabelValues(s.name).Inc()

						// send action on channel
						*action <- s.getAction()
					}
				} else {
					downCounter++

					sLog.WithFields(log.Fields{
						"failures": downCounter,
					}).Debug("service moving towards down")
				}
			}
		}
	}
}

// Stop stops the service check from running
func (s *ServiceCheck) Stop() {
	s.stopped <- true

	log.WithFields(log.Fields{
		"service": s.name,
	}).Debug("stopped service")
}

// Name returns the service check's name
func (s *ServiceCheck) Name() string {
	return s.name
}

// IsUp returns whether the service is considered up by birdwatcher
func (s *ServiceCheck) IsUp() bool {
	return (s.state == ServiceStateUp)
}

func (s *ServiceCheck) getAction() *Action {
	return &Action{
		Service:  s,
		State:    s.state,
		Prefixes: s.prefixes,
	}
}

func (s *ServiceCheck) performCheck() error {
	sLog := log.WithFields(log.Fields{
		"service": s.name,
		"command": s.Command,
	})
	sLog.Debug("performing check")

	// create context that automatically times out
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	// split reload command into command/args assuming the first part is the command
	// and the rest are the arguments
	commandArgs := strings.Split(s.Command, " ")

	// set up command execution within that context
	cmd := exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...)

	// get exit code of command
	output, err := cmd.Output()

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ctx.Err()
	}

	if err != nil {
		sLog.WithError(err).WithField("output", output).Debug("check output")
	}

	return err
}
