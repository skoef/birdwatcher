package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	checkConfig := fs.Bool("check-config", false, "check config file and exit")
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
		// return slightly different message when birdwatcher was invoked with -check-config
		if *checkConfig {
			fmt.Printf("Configuration file %s not OK: %s\n", *configFile, errors.Unwrap(err))
			os.Exit(1)
		}

		log.Fatal(err.Error())
	}

	if *checkConfig {
		fmt.Printf("Configuration file %s OK\n", *configFile)
		if *debug {
			configJSON, _ := json.MarshalIndent(config, "", "  ")
			fmt.Println(string(configJSON))
		}
		os.Exit(0)
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
