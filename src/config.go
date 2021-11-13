package main

import "time"

type ConfigType struct {
	InactiveRouteMetric int
	DeactivateThreshold time.Duration
	ActivateThreshold   time.Duration
	PingInterval        time.Duration
	ReplyTimeout        time.Duration
	DryRun              bool
}

var Config = ConfigType{
	InactiveRouteMetric: 10000,
	DeactivateThreshold: 30 * time.Second,
	ActivateThreshold:   120 * time.Second,
	PingInterval:        time.Second,
	ReplyTimeout:        5 * time.Second,
	DryRun:              true,
}
