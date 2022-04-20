package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/coreos/go-systemd/daemon"
	log "github.com/sirupsen/logrus"
	"github.com/skoef/birdwatcher/birdwatcher"
)

const (
	systemdStatusBufferSize = 32
)

//nolint:gochecknoinits
func init() {
	// initialize logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	var (
		configFile  = flag.String("config", "/etc/birdwatcher.conf", "path to config file")
		checkConfig = flag.Bool("check-config", false, "check config file and exit")
		debugFlag   = flag.Bool("debug", false, "increase loglevel to debug")
		useSystemd  = flag.Bool("systemd", false, "optimize behavior for running under systemd")
		version     = flag.Bool("version", false, "show version and exit")
	)
	flag.Parse()

	versionString := "(devel)"
	if vcs, ok := debug.ReadBuildInfo(); ok {
		versionString = vcs.Main.Version
	}

	if *version {
		fmt.Printf("birdwatcher, %s\n", versionString)

		return
	}

	log.Infof("starting birdwatcher, %s", versionString)

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}

	if *useSystemd {
		// if we're running under systemd, we don't need the timestamps
		// since journald will take care of those
		log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	}

	log.WithFields(log.Fields{
		"configFile": *configFile,
	}).Debug("opening configuration file")

	var config birdwatcher.Config
	if err := birdwatcher.ReadConfig(&config, *configFile); err != nil {
		// return slightly different message when birdwatcher was invoked with -check-config
		if *checkConfig {
			fmt.Printf("Configuration file %s not OK: %s\n", *configFile, errors.Unwrap(err))
			os.Exit(1)
		}

		log.Fatal(err.Error())
	}

	if *checkConfig {
		fmt.Printf("Configuration file %s OK\n", *configFile)
		if *debugFlag {
			configJSON, _ := json.MarshalIndent(config, "", "  ")
			fmt.Println(string(configJSON))
		}

		return
	}

	// start health checker
	hc := birdwatcher.NewHealthCheck(config)
	ready := make(chan bool)
	var status *chan string
	if *useSystemd {
		// create status update channel for systemd
		// give it a little buffer so the chances of it blocking the health check
		// is low
		s := make(chan string, systemdStatusBufferSize)
		status = &s
		go func() {
			for update := range *status {
				log.Debug("notifying systemd of new status")
				sdnotify(fmt.Sprintf("STATUS=%s", update))
			}
		}()
	}
	go hc.Start(config.GetServices(), ready, status)
	// wait for all health services to have started
	<-ready

	if *useSystemd {
		log.Debug("notifying systemd birdwatcher is ready")
		sdnotify(daemon.SdNotifyReady)
	}

	// wait until interrupted
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-signalCh
	log.WithFields(log.Fields{
		"signal": sig,
	}).Info("signal received, stopping")

	if *useSystemd {
		log.Debug("notifying systemd birdwatcher is stopping")
		sdnotify(daemon.SdNotifyStopping)
	}

	hc.Stop()
}

// sdnotify is a little wrapper for daemon.SdNotify
func sdnotify(msg string) {
	if ok, err := daemon.SdNotify(false, msg); ok && err != nil {
		log.WithError(err).Error("could not notify systemd")
	}
}
