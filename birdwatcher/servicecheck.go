package birdwatcher

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
	name               string
	FunctionName       string
	Command            string
	Interval           int
	Timeout            int
	Fail               int
	Rise               int
	Prefixes           []string
	prefixes           []net.IPNet
	state              ServiceState
	disablePrefixCheck bool
	stopped            chan interface{}
}

// Start starts the process of health checking its service and sends actions to
// the action channel when service state changes
func (s *ServiceCheck) Start(action *chan *Action) {
	s.stopped = make(chan interface{})
	ticker := time.NewTicker(time.Second * time.Duration(s.Interval))

	var err error
	upCounter := 0
	downCounter := 0

	for {

		select {
		case <-s.stopped:
			log.WithFields(log.Fields{
				"service": s.name,
			}).Debug("received stop signal")
			// we're done
			return

		case <-ticker.C:
			// perform check synchronously to prevent checks to queue
			err = s.performCheck(action)

			// based on the check result, decide if we're going up or down
			//
			// check gave positive result
			if err == nil {
				// reset downCounter
				downCounter = 0

				log.WithFields(log.Fields{
					"service": s.name,
					"command": s.Command,
				}).Debug("check command exited without error")

				// are we up enough to consider service to be healthy
				if upCounter >= (s.Rise - 1) {
					if s.state != ServiceStateUp {
						log.WithFields(log.Fields{
							"service":   s.name,
							"command":   s.Command,
							"successes": upCounter,
						}).Info("service transitioning to up")

						s.state = ServiceStateUp

						// send action on channel
						*action <- s.getAction()
					}
					// or are we still in the process of coming up
				} else {
					upCounter++

					log.WithFields(log.Fields{
						"service":   s.name,
						"command":   s.Command,
						"successes": upCounter,
					}).Debug("service moving towards up")
				}

				// check gave negative result
			} else {
				// reset upcounter
				upCounter = 0

				log.WithFields(log.Fields{
					"service": s.name,
					"command": s.Command,
				}).Debug("check command failed or timed out")

				// are we down long enough to consider service down
				if downCounter >= (s.Fail - 1) {
					if s.state != ServiceStateDown {
						log.WithFields(log.Fields{
							"service":  s.name,
							"command":  s.Command,
							"failures": downCounter,
						}).Info("service transitioning to down")

						s.state = ServiceStateDown

						// send action on channel
						*action <- s.getAction()
					}
				} else {
					downCounter++

					log.WithFields(log.Fields{
						"service":  s.name,
						"command":  s.Command,
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

func (s *ServiceCheck) getAction() *Action {
	return &Action{
		Service:  s,
		State:    s.state,
		Prefixes: s.prefixes,
	}
}

func (s *ServiceCheck) performCheck(action *chan *Action) error {
	log.WithFields(log.Fields{
		"service": s.name,
		"command": s.Command,
	}).Debug("performing check")

	if s.Timeout <= 0 {
		s.Timeout = 1
	}

	// create context that automatically times out
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.Timeout)*time.Second)
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
	if ctx.Err() == context.DeadlineExceeded {
		return ctx.Err()
	}

	if err != nil {
		log.WithFields(log.Fields{
			"service": s.name,
			"command": s.Command,
			"error":   err.Error(),
			"output":  output,
		}).Debug("check output")
	}

	return err
}
