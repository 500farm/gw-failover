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
		pinger.Interval = PingInterval
		pinger.RecordRtts = false
		pinger.SetPrivileged(true)
		pinger.OnSend = func(packet *ping.Packet) {
			r.counter.AddRequest()
		}
		pinger.OnRecv = func(packet *ping.Packet) {
			r.counter.AddReply()
			// log.Infof("%s: ping reply", r.Name())
		}
		r.pinger = pinger
		r.counter = NewPingCounter(UpThreshold)
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
	stats := r.counter.Stats(PingInterval)
	log.Infof(
		"%s: down time %v, up time %v, reply ratio %.2f",
		r.Name(), stats.downTime, stats.upTime, stats.replyRatio,
	)
	if r.Active && stats.downTime >= DownThreshold {
		log.Warnf("Gateway %s is now DOWN after %v of no reply", r.Name(), stats.downTime)
		r.Deactivate()
		return true
	}
	if !r.Active && stats.upTime >= UpThreshold && stats.replyRatio >= AcceptableReplyRatio {
		log.Warnf("Gateway %s is now UP after %v with reply ratio %.2f", r.Name(), stats.upTime, stats.replyRatio)
		r.Activate()
		return true
	}
	return false
}

func (r *DefaultRoute) Activate() {
	r.Active = true
	err := r.ApplyMetric(r.Metric)
	if err != nil {
		log.Error(err)
	}
}

func (r *DefaultRoute) Deactivate() {
	r.Active = false
	err := r.ApplyMetric(r.Metric + InactiveRouteMetric)
	if err != nil {
		log.Error(err)
		return
	}
	r.ResetConnections()
}

func (r *DefaultRoute) ApplyMetric(metric int) error {
	if DryRun {
		return nil
	}
	cmd := exec.Command("ip", "route", "delete", "default", "via", r.Gateway, "dev", r.Interface)
	_, err := cmd.Output()
	if err != nil {
		log.Error(err)
	}
	cmd = exec.Command("ip", "route", "add", "default", "via", r.Gateway, "dev", r.Interface, "metric", strconv.Itoa(metric))
	_, err = cmd.Output()
	return err
}

func (r *DefaultRoute) ResetConnections() {
	if DryRun {
		return
	}
	// FIXME
}

func (r *DefaultRoute) Name() string {
	return fmt.Sprintf("%s@%s", r.Gateway, r.Interface)
}
