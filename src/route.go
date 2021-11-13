package main

import (
	"fmt"
	"os/exec"
	"strconv"

	ping "github.com/go-ping/ping"
	log "github.com/sirupsen/logrus"
)

type DefaultRoute struct {
	Interface string
	Gateway   string
	Source    string
	Active    bool
	Metric    int
	pinger    *ping.Pinger
	counter   *PingCounter
}

func (r *DefaultRoute) StartPinging() error {
	if r.pinger == nil {
		pinger, err := ping.NewPinger(r.Gateway)
		if err != nil {
			return err
		}
		pinger.Interval = Config.PingInterval
		pinger.RecordRtts = false
		pinger.SetPrivileged(true)
		pinger.Source = r.Source
		pinger.OnSend = func(packet *ping.Packet) {
			r.counter.AddRequest()
		}
		pinger.OnRecv = func(packet *ping.Packet) {
			r.counter.AddReply()
			// log.Infof("%s: ping reply", r.Name())
		}
		r.pinger = pinger
		r.counter = NewPingCounter(Config.ActivateThreshold)
	}
	go r.pinger.Run()
	return nil
}

func (r *DefaultRoute) StopPinging() {
	if r.pinger != nil {
		r.pinger.Stop()
		r.pinger = nil
		r.counter = nil
	}
}

func (r *DefaultRoute) Check() bool {
	if r.counter == nil {
		return false
	}
	stats := r.counter.Stats(Config.ReplyTimeout)
	if Config.DryRun {
		log.Infof(
			"%s: since last reply %v, down time %v, up time %v",
			r.Name(), stats.waitTime, stats.downTime, stats.upTime,
		)
	}
	if r.Active && stats.downTime >= Config.DeactivateThreshold {
		log.Warnf("Gateway %s is now DOWN after %v of no reply", r.Name(), stats.downTime)
		r.deactivate()
		return true
	}
	if !r.Active && stats.upTime >= Config.ActivateThreshold {
		log.Warnf("Gateway %s is now UP after %v", r.Name(), stats.upTime)
		r.activate()
		return true
	}
	return false
}

func (r *DefaultRoute) activate() {
	r.Active = true
	err := r.applyMetric(r.Metric)
	if err != nil {
		log.Error(err)
	}
}

func (r *DefaultRoute) deactivate() {
	r.Active = false
	err := r.applyMetric(r.Metric + Config.InactiveRouteMetric)
	if err != nil {
		log.Error(err)
		return
	}
	r.resetConnections()
}

func (r *DefaultRoute) applyMetric(metric int) error {
	if Config.DryRun {
		return nil
	}
	cmd := exec.Command(
		"ip", "route", "delete", "default", "via", r.Gateway, "dev", r.Interface
	)
	_, err := cmd.Output()
	if err != nil {
		log.Error(err)
	}
	cmd = exec.Command(
		"ip", "route", "add", "default", "via", r.Gateway, "dev", r.Interface, "metric", strconv.Itoa(metric), "proto", "static"
	)
	_, err = cmd.Output()
	return err
}

func (r *DefaultRoute) resetConnections() {
	if Config.DryRun {
		return
	}
	// FIXME
}

func (r *DefaultRoute) Name() string {
	return fmt.Sprintf("%s@%s", r.Gateway, r.Interface)
}
