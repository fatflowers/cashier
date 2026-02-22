package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var HistogramBuckets = []float64{
	// --- Fast responses (0 - 500ms) ---
	25, 50, 75, 100, 150, 200, 300, 400, 500,

	// --- Medium responses around 700ms (500ms - 2s) ---
	750, 1000, 1250, 1500, 1750, 2000,

	// --- Slow responses (2s - 15s) ---
	2500, 3000, 4000, 5000, 7500, 10000, 15000,

	// --- Extended range: covers 60000ms+ (15s - 75s) ---
	20000,  // 20s
	30000,  // 30s
	45000,  // 45s
	60000,  // 60s
	75000,  // 75s
	90000,  // 90s
	120000, // 120s
}

// Metric is a definition for the name, description, type, ID, and
// prometheus.Collector type (i.e. CounterVec, Summary, etc) of each metric
type Metric struct {
	MetricCollector prometheus.Collector
	ID              string
	Name            string
	Description     string
	Type            string
	Args            []string
}

// NewMetric associates prometheus.Collector based on Metric.Type
func NewMetric(m *Metric, subsystem string) prometheus.Collector {
	var metric prometheus.Collector
	switch m.Type {
	case "counter_vec":
		metric = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "counter":
		metric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "gauge_vec":
		metric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "gauge":
		metric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "histogram_vec":
		metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "histogram":
		metric = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
				Buckets:   HistogramBuckets,
			},
		)
	case "summary_vec":
		metric = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "summary":
		metric = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	}
	return metric
}

var MetricsBusinessProcess = &Metric{
	ID:          "bpDur",
	Name:        "bp_dur",
	Description: "process latency in milliseconds",
	Type:        "histogram_vec",
	Args:        []string{"type", "subtype"},
}

const (
	RefererKey = "X-Referer"
)
