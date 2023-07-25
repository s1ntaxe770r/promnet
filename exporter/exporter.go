package exporter

import (
	"math"
	"os"
	"sync"
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
		Prefix: "SpeedExporter⚡️",
	})
)

type Exporter struct {
	log      *log.Logger
	cacheMtx sync.RWMutex
	cache    *speedTestResult
}

type speedTestResult struct {
	Timestamp time.Time
	Latency   float64
	Download  float64
	Upload    float64
}

func NewExporter() *Exporter {
	return &Exporter{
		log:   logger,
		cache: &speedTestResult{},
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- downloadSpeed
	ch <- uploadSpeed
	ch <- latency
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// Check if cache is empty or it's been more than 5 minutes since the last test
	if e.cache.Timestamp.IsZero() || time.Since(e.cache.Timestamp) > 5*time.Minute {
		// If the cache is empty or expired, trigger a new speed test in a separate goroutine
		go e.runSpeedTest()
	}

	// Read from cache
	e.cacheMtx.RLock()
	defer e.cacheMtx.RUnlock()

	ch <- prometheus.MustNewConstMetric(latency, prometheus.GaugeValue, e.cache.Latency)
	ch <- prometheus.MustNewConstMetric(downloadSpeed, prometheus.GaugeValue, e.cache.Download)
	ch <- prometheus.MustNewConstMetric(uploadSpeed, prometheus.GaugeValue, e.cache.Upload)
}

func (e *Exporter) runSpeedTest() {
	e.log.Info("Starting speed test")
	serverList, err := speedTestClient.FetchServers()
	if err != nil {
		log.Error("Error fetching server list", err)
		return
	}
	targets, _ := serverList.FindServer([]int{})

	var latestResult speedTestResult

	for _, s := range targets {
		s.PingTest(nil)
		s.DownloadTest()
		s.UploadTest()

		latestResult.Latency = math.Round(float64(s.Latency) / 1000000)
		latestResult.Download = s.DLSpeed
		latestResult.Upload = s.ULSpeed
	}

	latestResult.Timestamp = time.Now()

	// Update the cache with the latest result
	e.cacheMtx.Lock()
	defer e.cacheMtx.Unlock()
	e.cache = &latestResult

	e.log.Info("Speed test completed", "Last run took", time.Since(latestResult.Timestamp))
}
