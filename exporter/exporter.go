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

	latency = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "speedtest_latency",
	})
	downloadSpeed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "speedtest_downloadSpeed",
	})
	uploadSpeed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "speedtest_uploadSpeed",
	})

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

// IMPLEMTATION OF COLLECTOR INTERFACE
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- latency.Desc()
	ch <- downloadSpeed.Desc()
	ch <- uploadSpeed.Desc()
}

// IMPLEMTATION OF COLLECT method
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// Check if cache is empty or it's been more than 5 minutes since the last test
	if e.cache.Timestamp.IsZero() || time.Since(e.cache.Timestamp) > 4*time.Minute {
		// If the cache is empty or expired, trigger a new speed test in a separate goroutine
		go e.runSpeedTest()
	}

	// Read from cache
	e.cacheMtx.RLock()
	defer e.cacheMtx.RUnlock()

	// set the value of the metric to the value of the cache

	ch <- prometheus.MustNewConstMetric(latency.Desc(), prometheus.GaugeValue, e.cache.Latency)
	ch <- prometheus.MustNewConstMetric(downloadSpeed.Desc(), prometheus.GaugeValue, e.cache.Download)
	ch <- prometheus.MustNewConstMetric(uploadSpeed.Desc(), prometheus.GaugeValue, e.cache.Upload)
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

		latency.Set(latestResult.Latency)
		downloadSpeed.Set(latestResult.Download)
		uploadSpeed.Set(latestResult.Upload)
		TestsConducted.Inc()
	}

	latestResult.Timestamp = time.Now()

	// Update the cache with the latest result
	e.cacheMtx.Lock()
	defer e.cacheMtx.Unlock()
	e.cache = &latestResult

	e.log.Info("Speed test completed", "Last run took", time.Since(latestResult.Timestamp))
}
