package main

import (
	"flag"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/s1ntaxe770r/promnet/exporter"
)

func main() {
	port := flag.String("port", "9817", "listening port to expose metrics on")
	flag.Parse()
	exp := exporter.NewExporter()

	r := prometheus.NewRegistry()
	r.MustRegister(exp)
	r.MustRegister(exporter.TestsConducted)

	// c.AddFunc("*/5 * * * *", func() {
	// 	t := time.Now()
	// 	log.Info("Starting speed test", "time", t.Format("2006-01-02 15:04:05"))
	// 	log.Info("Speed test completed", "Last run took", time.Since(t))
	// })

	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	log.Info("Starting prometheus server on", "port", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))

}
