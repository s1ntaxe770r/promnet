package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/robfig/cron/v3"
	"github.com/showwin/speedtest-go/speedtest"
)

var (
	speedTestClient = speedtest.New()

	latency = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "latency",
		Help: "Internet latency",
	})

	downloadSpeed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "downloadSpeed",
		Help: "Internet download speed",
	})

	uploadSpeed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "uploadSpeed",
		Help: "Internet upload speed",
	})

	lastRun = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lastRun",
		Help: "Last time the speed test was run",
	})
)

func RunSpeedTest() {
	serverList, err := speedTestClient.FetchServers()
	if err != nil {
		log.Error("Error fetching server list", err)
	}
	s := serverList[0]
	s.PingTest(nil)
	s.DownloadTest()
	s.UploadTest()

	// convert to latency time duration to float64
	latency.Set(float64(s.Latency / 1000000))
	downloadSpeed.Set(s.DLSpeed)
	uploadSpeed.Set(s.ULSpeed)
	lastRun.SetToCurrentTime()

}

func main() {
	port := flag.String("port", "9817", "listening port to expose metrics on")
	flag.Parse()
	c := cron.New()
	r := prometheus.NewRegistry()
	r.MustRegister(latency)
	r.MustRegister(downloadSpeed)
	r.MustRegister(uploadSpeed)
	r.MustRegister(lastRun)

	c.AddFunc("*/5 * * * *", func() {
		t := time.Now()
		log.Info("Starting speed test", "time", t.Format("2006-01-02 15:04:05"))
		RunSpeedTest()
		log.Info("Speed test completed", "Last run took", time.Since(t))
	})

	// start prometheus server in a goroutine
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
		log.Info("Starting prometheus server on", "port", *port)
		log.Fatal(http.ListenAndServe(":"+*port, nil))
	}()

	c.Run()
}
