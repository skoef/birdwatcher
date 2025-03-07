// Package main is the main runtime of the birdwatcher application
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/skoef/birdwatcher/birdwatcher"
)

const (
	systemdStatusBufferSize = 32
)

//nolint:funlen // we should refactor this a bit
func main() {
	// initialize logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

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
			fmt.Printf("Configuration file %s not OK: %s\n", *configFile, err)
			os.Exit(1)
		}

		log.Fatal(err.Error())
	}

	if *checkConfig {
		fmt.Printf("Configuration file %s OK\n", *configFile)

		if *debugFlag {
			configJSON, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				log.Fatal(err.Error())
			}

			fmt.Println(string(configJSON))
		}

		return
	}

	// enable prometheus
	// Expose /metrics HTTP endpoint using the created custom registry.
	if config.Prometheus.Enabled {
		go func() {
			if err := startPrometheus(config.Prometheus); err != nil {
				log.WithError(err).Fatal("could not start prometheus exporter")
			}
		}()
	}

	// start health checker
	hc := birdwatcher.NewHealthCheck(config)
	ready := make(chan bool)

	// create status update channel for systemd
	// give it a little buffer so the chances of it blocking the health check
	// is low
	sdStatus := make(chan string, systemdStatusBufferSize)
	go func() {
		// make sure we read from the sdStatus channel, regardless if we use
		// systemd integration or not to prevent the channel from blocking
		for update := range sdStatus {
			if *useSystemd {
				log.Debug("notifying systemd of new status")
				sdnotify("STATUS=" + update)
			}
		}
	}()

	go hc.Start(config.GetServices(), ready, sdStatus)
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

func startPrometheus(c birdwatcher.PrometheusConfig) error {
	log.WithFields(log.Fields{
		"port": c.Port,
		"path": c.Path,
	}).Info("starting prometheus exporter")

	mux := http.NewServeMux()
	mux.Handle(c.Path, promhttp.Handler())

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", c.Port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}

	return httpServer.ListenAndServe()
}
