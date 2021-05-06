package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/skoef/birdwatcher/birdwatcher"
)

var (
	// set during building
	buildVersion = "HEAD"   //nolint:gochecknoglobals
	buildBranch  = "master" //nolint:gochecknoglobals
)

//nolint:gochecknoinits
func init() {
	// initialize logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	var versionString string
	// release or custom build
	if regexp.MustCompile(`^v[0-9\.]+$`).MatchString(buildVersion) {
		versionString = fmt.Sprintf("version %s", strings.Replace(buildVersion, "v", "", 1))
	} else {
		versionString = fmt.Sprintf("build %s (%s branch)", buildVersion, buildBranch)
	}

	var (
		configFile  = flag.String("config", "/etc/birdwatcher.conf", "path to config file")
		checkConfig = flag.Bool("check-config", false, "check config file and exit")
		debug       = flag.Bool("debug", false, "increase loglevel to debug")
		version     = flag.Bool("version", false, "show version and exit")
	)
	flag.Parse()

	if *version {
		fmt.Printf("birdwatcher, %s\n", versionString)
		os.Exit(0)
	}

	log.Infof("starting birdwatcher, %s", versionString)

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.WithFields(log.Fields{
		"configFile": *configFile,
	}).Debug("Opening configuration file")

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

	sig := <-signalCh
	log.WithFields(log.Fields{
		"signal": sig,
	}).Info("Signal received, stopping")
	hc.Stop(config.GetServices())
}
