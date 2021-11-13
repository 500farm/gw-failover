package main

import (
	"encoding/json"
	"errors"
	"os/exec"
	"os/user"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
)

type DefaultRoutes []*DefaultRoute

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

	rs, err := readRoutes()
	if err != nil {
		return err
	}
	log.Info(rs.String())

	err = rs.startPingingAll()
	if err != nil {
		return err
	}
	for {
		time.Sleep(time.Second)
		if rs.checkAll() {
			log.Info(rs.String())
		}
	}
}

func readRoutes() (DefaultRoutes, error) {
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
				Active:    true,
			}
			if t.Metric >= Config.InactiveRouteMetric {
				t.Metric -= Config.InactiveRouteMetric
				t.Active = false
			}
			result = append(result, &t)
		}
	}
	sort.Slice(routes, func(i int, j int) bool {
		return routes[i].Metric < routes[j].Metric
	})
	return result, nil
}

func (rs *DefaultRoutes) startPingingAll() error {
	for _, r := range *rs {
		err := r.StartPinging()
		if err != nil {
			rs.stopPingingAll()
			return err
		}
	}
	return nil
}

func (rs *DefaultRoutes) stopPingingAll() {
	for _, r := range *rs {
		r.StopPinging()
	}
}

func (rs *DefaultRoutes) checkAll() bool {
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
