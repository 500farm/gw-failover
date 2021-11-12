package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"time"

	ping "github.com/go-ping/ping"
	log "github.com/sirupsen/logrus"
)

const InactiveRouteMetric = 10000
const DownThreshold = 30 * time.Second
const UpThreshold = 120 * time.Second
const PingInterval = time.Second

type DefaultRoute struct {
	Interface string
	Gateway   string
	Active    bool
	Metric    int
	LastReply time.Time
	pinger    *ping.Pinger
}

type DefaultRoutes []DefaultRoute

type IpRouteResponse []struct {
	Destination string `json:"dst"`
	Interface   string `json:"dev"`
	Gateway     string `json:"gateway"`
	Metric      int    `json:"metric"`
}

func WatchLoop() error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	if user.Uid != "0" {
		return errors.New("please run under root")
	}

	rs, err := ReadRoutes()
	if err != nil {
		return err
	}
	log.Info(rs.String())

	err = rs.StartPingingAll()
	if err != nil {
		return err
	}
	for {
		time.Sleep(time.Second)
		if rs.CheckAll() {
			log.Info(rs.String())
		}
	}
}

func ReadRoutes() (DefaultRoutes, error) {
	cmd := exec.Command("ip", "-j", "route", "list")
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var routes IpRouteResponse
	err = json.Unmarshal(stdout, &routes)
	if err != nil {
		return nil, err
	}
	var result DefaultRoutes
	for _, route := range routes {
		if route.Destination == "default" {
			t := DefaultRoute{
				Interface: route.Interface,
				Gateway:   route.Gateway,
				Metric:    route.Metric,
			}
			if t.Metric > InactiveRouteMetric {
				t.Metric -= InactiveRouteMetric
				t.Active = false
			} else {
				t.Active = true
				t.LastReply = time.Now()
			}
			result = append(result, t)
		}
	}
	return result, nil
}

func (rs *DefaultRoutes) StartPingingAll() error {
	for _, r := range *rs {
		err := r.StartPinging()
		if err != nil {
			rs.StopPingingAll()
			return err
		}
	}
	return nil
}

func (rs *DefaultRoutes) StopPingingAll() {
	for _, r := range *rs {
		r.StopPinging()
	}
}

func (rs *DefaultRoutes) CheckAll() bool {
	changed := false
	for _, r := range *rs {
		if r.Check() {
			changed = true
		}
	}
	return changed
}

func (r *DefaultRoutes) String() string {
	j, _ := json.Marshal(*r)
	return string(j)
}

func (r *DefaultRoute) StartPinging() error {
	if r.pinger == nil {
		pinger, err := ping.NewPinger(r.Gateway)
		if err != nil {
			return err
		}
		pinger.SetPrivileged(true)
		pinger.OnRecv = func(packet *ping.Packet) {
			r.PingReplyReceived()
		}
		r.pinger = pinger
	}
	go r.pinger.Run()
	return nil
}

func (r *DefaultRoute) StopPinging() {
	if r.pinger != nil {
		r.pinger.Stop()
	}
}

func (r *DefaultRoute) PingReplyReceived() {
	r.LastReply = time.Now()
	// log.Infof("%s: ping reply", r.Name())
}

func (r *DefaultRoute) Check() bool {
	if time.Since(r.LastReply) > DownThreshold {
		log.Warn("Gateway %s is now DOWN", r.Name())
		r.Deactivate()
		return true
	}
	// FIXME activate
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

	r.StopPinging()
	r.ResetLink()
	r.StartPinging()
}

func (r *DefaultRoute) ApplyMetric(metric int) error {
	// FIXME
	cmd := exec.Command("ip", "route", "change")
	_, err := cmd.Output()
	return err
}

func (r *DefaultRoute) ResetLink() {
	// FIXME
}

func (r *DefaultRoute) Name() string {
	return fmt.Sprintf("%s@%s", r.Gateway, r.Interface)
}
