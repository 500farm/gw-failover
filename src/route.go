package main

import (
	"fmt"
	"os/exec"
	"strconv"

	ping "github.com/go-ping/ping"
	"github.com/prometheus/client_golang/prometheus"
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
			r.IncPromMetric(coll.ping_requests_total)
		}
		pinger.OnRecv = func(packet *ping.Packet) {
			r.counter.AddReply()
			r.IncPromMetric(coll.ping_replies_total)
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
	r.SetPromMetricBool(coll.route_up, stats.upTime > 0)
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
	err := r.applyMetric(r.Metric)
	if err != nil {
		log.Error(err)
		return
	}

	r.Active = true
	r.UpdatePromMetrics()
}

func (r *DefaultRoute) deactivate() {
	err := r.applyMetric(r.Metric + Config.InactiveRouteMetric)
	if err != nil {
		log.Error(err)
		return
	}

	r.resetConnections()
	r.Active = false
	r.UpdatePromMetrics()
}

func (r *DefaultRoute) applyMetric(metric int) error {
	if Config.DryRun {
		return nil
	}
	cmd := exec.Command(
		"ip", "route", "delete", "default", "via", r.Gateway, "dev", r.Interface, "proto", "static",
	)
	_, err := cmd.Output()
	if err != nil {
		log.Error(err)
	}
	cmd = exec.Command(
		"ip", "route", "add", "default", "via", r.Gateway, "dev", r.Interface, "metric", strconv.Itoa(metric), "proto", "static",
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

func (r *DefaultRoute) InitPromMetrics() {
	r.SetPromMetricBool(coll.route_up, true)
	r.SetPromMetric(coll.ping_requests_total, 0)
	r.SetPromMetric(coll.ping_replies_total, 0)
	r.UpdatePromMetrics()
}

func (r *DefaultRoute) UpdatePromMetrics() {
	r.SetPromMetricBool(coll.route_active, r.Active)
	r.SetPromMetric(coll.route_metric, float64(r.Metric))
}

func (r *DefaultRoute) SetPromMetric(vec *prometheus.GaugeVec, value float64) {
	vec.With(prometheus.Labels{"gateway": r.Gateway, "interface": r.Interface}).Set(value)
}

func (r *DefaultRoute) IncPromMetric(vec *prometheus.GaugeVec) {
	vec.With(prometheus.Labels{"gateway": r.Gateway, "interface": r.Interface}).Inc()
}

func (r *DefaultRoute) SetPromMetricBool(vec *prometheus.GaugeVec, value bool) {
	if value {
		r.SetPromMetric(vec, 1)
	} else {
		r.SetPromMetric(vec, 0)
	}
}
