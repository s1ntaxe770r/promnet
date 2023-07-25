package exporter

import (
	"math"
	"os"
	"time"

	"github.com/charmbracelet/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/showwin/speedtest-go/speedtest"
)

var (
	speedTestClient = speedtest.New()

	latency = prometheus.NewDesc(
		prometheus.BuildFQName("speedtest", "", "latency"),
		"Internet latency",
		nil, nil,
	)

	downloadSpeed = prometheus.NewDesc(
		prometheus.BuildFQName("speedtest", "", "downloadSpeed"),
		"Internet download speed",
		nil, nil,
	)
	uploadSpeed = prometheus.NewDesc(
		prometheus.BuildFQName("speedtest", "", "uploadSpeed"),
		"Internet upload speed",
		nil, nil,
	)

	TestsConducted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tests_conducted",
		Help: "Number of tests conducted",
	})

	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "Baking 🍪 ",
	})
)

type Exporter struct {
	log *log.Logger
}

func NewExporter() *Exporter {
	return &Exporter{
		log: logger,
	}
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- downloadSpeed
	ch <- uploadSpeed
	ch <- latency
}

// Collect implements prometheus.Collector.

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	t := time.Now()
	e.log.Info("Starting speed test", "time", t.Format("2006-01-02 15:04:05"))
	serverList, err := speedTestClient.FetchServers()
	if err != nil {
		log.Error("Error fetching server list", err)
	}
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		s.PingTest(nil)
		s.DownloadTest()
		s.UploadTest()

		TestsConducted.Inc()

		ch <- prometheus.MustNewConstMetric(latency, prometheus.GaugeValue, math.Round(float64(s.Latency/1000000)))
		ch <- prometheus.MustNewConstMetric(downloadSpeed, prometheus.GaugeValue, math.Round(s.DLSpeed))
		ch <- prometheus.MustNewConstMetric(uploadSpeed, prometheus.GaugeValue, math.Round(s.ULSpeed))

	}
	log.Info("Speed test completed", "Last run took", time.Since(t))
}
