package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"os/user"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
)

type DefaultRoutes []*DefaultRoute

type IpRouteResponseItem struct {
	Destination string        `json:"dst"`
	Interface   string        `json:"dev"`
	Gateway     string        `json:"gateway"`
	Source      string        `json:"prefsrc"`
	Metric      int           `json:"metric"`
	Protocol    string        `json:"protocol"`
	Nexthops    []interface{} `json:"nexthops"`
}

type IpRouteResponse []IpRouteResponseItem

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
	log.Infof("Routes: %s", rs.String())

	err = rs.startPingingAll()
	if err != nil {
		return err
	}
	for {
		time.Sleep(time.Second)
		if rs.checkAll() {
			log.Infof("Routes: %s", rs.String())
		}
	}
}

func readRoutes() (DefaultRoutes, error) {
	var result DefaultRoutes
	err := result.read(false)
	if err != nil {
		return nil, err
	}
	if Config.Ipv6 {
		err = result.read(true)
		if err != nil {
			return nil, err
		}
	}
	if len(result) == 0 {
		return nil, errors.New("nothing to do")
	}
	sort.Slice(result, func(i int, j int) bool {
		return result[i].Metric < result[j].Metric
	})
	return result, nil
}

func (rs *DefaultRoutes) read(ipv6 bool) error {
	var cmd *exec.Cmd
	if ipv6 {
		cmd = exec.Command("ip", "-j", "-6", "route", "list", "default")
	} else {
		cmd = exec.Command("ip", "-j", "route", "list", "default")
	}
	var routes IpRouteResponse
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}
	err = json.Unmarshal(stdout, &routes)
	if err != nil {
		return err
	}
	c := 0
	result := DefaultRoutes{}
	for _, route := range routes {
		if route.Interface == "" || route.Gateway == "" {
			// p2p, multipath routes etc.
			return fmt.Errorf("unsupported route: %s", route.String())
		}
		if route.Protocol != "static" && !Config.DryRun {
			// dhcp/ra routes etc.
			return fmt.Errorf("non-static route can not be used: %s", route.String())
		}
		t := DefaultRoute{
			Interface: route.Interface,
			Gateway:   route.Gateway,
			Source:    route.Source,
			Metric:    route.Metric,
			Active:    true,
		}
		if t.Metric >= Config.InactiveRouteMetric {
			t.Metric -= Config.InactiveRouteMetric
			t.Active = false
		}
		if t.Source == "" {
			source, err := sourceAddr(t.Interface, net.ParseIP(t.Gateway))
			if err != nil {
				return err
			}
			if source != nil {
				t.Source = source.String()
			}
		}
		t.InitPromMetrics()
		result = append(result, &t)
		c++
	}
	if c >= 2 || Config.DryRun {
		*rs = append(*rs, result...)
	} else {
		proto := "IPv4"
		if ipv6 {
			proto = "IPv6"
		}
		log.Warnf("No redundant %s routes found: %v", proto, result.String())
	}
	return nil
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

func sourceAddr(ifname string, gw net.IP) (net.IP, error) {
	intf, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, err
	}
	addrs, err := intf.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ip, subnet, _ := net.ParseCIDR(addr.String())
		if ip.IsGlobalUnicast() && subnet.Contains(gw) ||
			ip.IsLinkLocalUnicast() && gw.IsLinkLocalUnicast() {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no usable source address found for %s", ifname)
}

func (r *DefaultRoutes) String() string {
	j, _ := json.MarshalIndent(*r, "", "    ")
	return string(j)
}

func (r *IpRouteResponseItem) String() string {
	j, _ := json.MarshalIndent(*r, "", "    ")
	return string(j)
}
