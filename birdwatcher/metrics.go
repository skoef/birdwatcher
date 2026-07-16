package birdwatcher

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	prometheusNamespace        = "birdwatcher"
	prometheusSubsystemService = "service"
	prometheusSubsystemPrefix  = "prefix"
	prometheusNameState        = "state"
)

var (
	serviceInfoMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemService,
		Name:      "info",
		Help:      "Services and their configuration",
	}, []string{"service", "function_name", "command", "interval", "timeout", "rise", "fail"})

	serviceCheckDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemService,
		Name:      "check_duration",
		Help:      "Service check duration in milliseconds",
	}, []string{"service"})

	serviceStateMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemService,
		Name:      prometheusNameState,
		Help:      "Current health state per service",
	}, []string{"service"})

	serviceTransitionMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemService,
		Name:      "transition_total",
		Help:      "Number of transitions per service",
	}, []string{"service"})

	serviceSuccessMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemService,
		Name:      "success_total",
		Help:      "Number of successful probes per service",
	}, []string{"service"})

	serviceFailMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemService,
		Name:      "fail_total",
		Help:      "Number of failed probes per service",
	}, []string{"service"})

	serviceTimeoutMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemService,
		Name:      "timeout_total",
		Help:      "Number of timed out probes per service",
	}, []string{"service"})

	prefixStateMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: prometheusNamespace,
		Subsystem: prometheusSubsystemPrefix,
		Name:      prometheusNameState,
		Help:      "Current health state per prefix",
	}, []string{"service", "prefix"})
)
