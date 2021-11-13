package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(coll)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func StartMetricsServer() error {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsHandler(w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<head>
		<title>Gateway Failover Service</title>
		</head>
		<body>
		<h1>Gateway Failover Service</h1>
		<a href="/metrics">Metrics</a>
		</body>
		</html>`))
	})
	log.Infof("Metrics server listening on %s", Config.MetricsServer)
	err := http.ListenAndServe(Config.MetricsServer, nil)
	if err != nil {
		return err
	}
	return nil
}
