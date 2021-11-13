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
				Active:    true,
			}
			if t.Metric >= InactiveRouteMetric {
				t.Metric -= InactiveRouteMetric
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
