package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/skoef/birdwatcher/birdwatcher"
)

var (
	config birdwatcher.Config
)

func init() {
	// initialize logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {

	fs := flag.NewFlagSet("birdwatcher", flag.ContinueOnError)
	configFile := fs.String("config", "", "config file (defaults to /etc/birdwatcher.conf)")
	debug := fs.Bool("debug", false, "increase loglevel to debug")

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if *configFile == "" {
		*configFile = "/etc/birdwatcher.conf"
	}

	log.WithFields(log.Fields{
		"configFile": *configFile,
	}).Debug("Opening configuration file")

	if err := birdwatcher.ReadConfig(&config, *configFile); err != nil {
		log.Fatal(err.Error())
	}

	// start health checker
	hc := birdwatcher.NewHealthCheck(config)
	hc.Start(config.GetServices())

	// wait until interrupted
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case sig := <-signalCh:
		log.WithFields(log.Fields{
			"signal": sig,
		}).Info("Signal received, stopping")
		hc.Stop(config.GetServices())
	}
}
