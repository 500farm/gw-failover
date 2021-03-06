package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	route_info             *prometheus.GaugeVec
	route_up               *prometheus.GaugeVec
	route_active           *prometheus.GaugeVec
	route_metric           *prometheus.GaugeVec
	ping_requests_total    *prometheus.GaugeVec
	ping_replies_total     *prometheus.GaugeVec
	ping_rtt_total_seconds *prometheus.GaugeVec
}

var coll *Collector

func newCollector() *Collector {
	namespace := "gw_failover"
	return &Collector{
		route_info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "route_info",
			Help:      "Route info.",
		}, []string{"gateway", "interface", "source", "external"}),
		route_up: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "route_up",
			Help:      "Whether the gateway on the route replies to pings.",
		}, []string{"gateway", "interface"}),
		route_active: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "route_active",
			Help:      "Whether this route is considered usable for outgoing traffic.",
		}, []string{"gateway", "interface"}),
		route_metric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "route_metric",
			Help:      "Metric of the route.",
		}, []string{"gateway", "interface"}),
		ping_requests_total: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ping_requests_total",
			Help:      "Counter of ping requests sent to the gateway.",
		}, []string{"gateway", "interface"}),
		ping_replies_total: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ping_replies_total",
			Help:      "Counter of ping replies received from the gateway.",
		}, []string{"gateway", "interface"}),
		ping_rtt_total_seconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ping_rtt_total_seconds",
			Help:      "Counter of ping round-trip time (divide by ping_replies_total to get single packet avg rtt).",
		}, []string{"gateway", "interface"}),
	}
}

func (e *Collector) Describe(ch chan<- *prometheus.Desc) {
	e.route_info.Describe(ch)
	e.route_up.Describe(ch)
	e.route_active.Describe(ch)
	e.route_metric.Describe(ch)
	e.ping_requests_total.Describe(ch)
	e.ping_replies_total.Describe(ch)
	e.ping_rtt_total_seconds.Describe(ch)
}

func (e *Collector) Collect(ch chan<- prometheus.Metric) {
	e.route_info.Collect(ch)
	e.route_up.Collect(ch)
	e.route_active.Collect(ch)
	e.route_metric.Collect(ch)
	e.ping_requests_total.Collect(ch)
	e.ping_replies_total.Collect(ch)
	e.ping_rtt_total_seconds.Collect(ch)
}
