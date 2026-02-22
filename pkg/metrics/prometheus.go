package metrics

/* copy from https://github.com/zsais/go-gin-prometheus
edits:
- replace slog with a new logger interface
- remove push gateway
*/

import (
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// Standard default metrics
//
//	counter, counter_vec, gauge, gauge_vec,
//	histogram, histogram_vec, summary, summary_vec
var reqCnt = &Metric{
	ID:          "reqCnt",
	Name:        "req_total",
	Description: "How many HTTP requests processed, partitioned by status code and HTTP method.",
	Type:        "counter_vec",
	Args:        []string{"code", "method", "url", "ref"}}

var reqDur = &Metric{
	ID:          "reqDur",
	Name:        "req_dur_ms",
	Description: "The HTTP request latencies in milliseconds.",
	Type:        "histogram_vec",
	Args:        []string{"code", "method", "url", "ref"},
}

var resSz = &Metric{
	ID:          "resSz",
	Name:        "resp_sz_bytes",
	Description: "The HTTP response sizes in bytes.",
	Type:        "summary_vec",
	Args:        []string{"code", "method", "url", "ref"},
}

var reqSz = &Metric{
	ID:          "reqSz",
	Name:        "req_sz_bytes",
	Description: "The HTTP request sizes in bytes.",
	Type:        "summary_vec",
	Args:        []string{"code", "method", "url", "ref"},
}

var standardMetrics = []*Metric{
	reqCnt,
	reqDur,
	resSz,
	reqSz,
}

var defaultMetricPath = "/metrics"

type Logger interface {
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
}

type defaultLogger struct {
	*log.Logger
}

func (l *defaultLogger) Error(v ...interface{}) {
	l.Logger.Println(v...)
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.Logger.Printf(format, v...)
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{Logger: log.Default()}
}

/*
RequestCounterURLLabelMappingFn is a function which can be supplied to the middleware to control
the cardinality of the request counter's "url" label, which might be required in some contexts.
For instance, if for a "/customer/:name" route you don't want to generate a time series for every
possible customer name, you could use this function:

	func(c *gin.Context) string {
		url := c.Request.URL.Path
		for _, p := range c.Params {
			if p.Key == "name" {
				url = strings.Replace(url, p.Value, ":name", 1)
				break
			}
		}
		return url
	}

which would map "/customer/alice" and "/customer/bob" to their template "/customer/:name".
*/
type RequestCounterURLLabelMappingFn func(c *gin.Context) string

// Prometheus contains the metrics gathered by the instance and its path
type Prometheus struct {
	reqCnt        *prometheus.CounterVec
	reqDur        *prometheus.HistogramVec
	reqSz, resSz  *prometheus.SummaryVec
	router        *gin.Engine
	listenAddress string

	MetricsList []*Metric
	MetricsPath string

	ReqCntURLLabelMappingFn RequestCounterURLLabelMappingFn

	// gin.Context string to use as a prometheus URL label
	URLLabelFromContext string

	logger Logger
}

type NewPrometheusOptions struct {
	Subsystem               string
	MetricsList             []*Metric
	MetricsPath             string
	ReqCntURLLabelMappingFn func(c *gin.Context) string
	Logger                  Logger
}

// NewPrometheus generates a new set of metrics with a certain subsystem name
func NewPrometheus(options NewPrometheusOptions) *Prometheus {

	var metricsList = options.MetricsList

	if len(options.MetricsList) > 1 {
		panic("Too many args. NewPrometheus( string, <optional []*Metric> ).")
	}

	p := &Prometheus{
		MetricsList: append(metricsList, standardMetrics...),
		logger:      options.Logger,
	}

	if options.MetricsPath != "" {
		p.MetricsPath = options.MetricsPath
	} else {
		p.MetricsPath = defaultMetricPath
	}

	if options.ReqCntURLLabelMappingFn != nil {
		p.ReqCntURLLabelMappingFn = options.ReqCntURLLabelMappingFn
	} else {
		p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
			return c.Request.URL.Path // i.e. by default do nothing, i.e. return URL as is
		}
	}

	if p.logger == nil {
		p.logger = newDefaultLogger()
	}

	p.registerMetrics(options.Subsystem)

	return p
}

// SetListenAddress for exposing metrics on address. If not set, it will be exposed at the
// same address of the gin engine that is being used
func (p *Prometheus) SetListenAddress(address string) {
	p.listenAddress = address
	if p.listenAddress != "" {
		p.router = gin.Default()
	}
}

// SetListenAddressWithRouter for using a separate router to expose metrics. (this keeps things like GET /metrics out of
// your content's access log).
func (p *Prometheus) SetListenAddressWithRouter(listenAddress string, r *gin.Engine) {
	p.listenAddress = listenAddress
	if len(p.listenAddress) > 0 {
		p.router = r
	}
}

// SetMetricsPath set metrics paths
func (p *Prometheus) SetMetricsPath(e *gin.Engine) {

	if p.listenAddress != "" {
		p.router.GET(p.MetricsPath, prometheusHandler())
		p.runServer()
	} else {
		e.GET(p.MetricsPath, prometheusHandler())
	}
}

// SetMetricsPathWithAuth set metrics paths with authentication
func (p *Prometheus) SetMetricsPathWithAuth(e *gin.Engine, accounts gin.Accounts) {

	if p.listenAddress != "" {
		p.router.GET(p.MetricsPath, gin.BasicAuth(accounts), prometheusHandler())
		p.runServer()
	} else {
		e.GET(p.MetricsPath, gin.BasicAuth(accounts), prometheusHandler())
	}

}

func (p *Prometheus) runServer() {
	if p.listenAddress != "" {
		go p.router.Run(p.listenAddress)
	}
}

func (p *Prometheus) registerMetrics(subsystem string) {

	for _, metricDef := range p.MetricsList {
		metric := NewMetric(metricDef, subsystem)
		if err := prometheus.Register(metric); err != nil {
			p.logger.Errorf("%s could not be registered in Prometheus, err=%v", metricDef.Name, err)
		}
		switch metricDef {
		case reqCnt:
			p.reqCnt = metric.(*prometheus.CounterVec)
		case reqDur:
			p.reqDur = metric.(*prometheus.HistogramVec)
		case resSz:
			p.resSz = metric.(*prometheus.SummaryVec)
		case reqSz:
			p.reqSz = metric.(*prometheus.SummaryVec)
		}
		metricDef.MetricCollector = metric
	}
}

// Use adds the middleware to a gin engine.
func (p *Prometheus) Use(e *gin.Engine) {
	e.Use(p.HandlerFunc())
	p.SetMetricsPath(e)
}

// UseWithAuth adds the middleware to a gin engine with BasicAuth.
func (p *Prometheus) UseWithAuth(e *gin.Engine, accounts gin.Accounts) {
	e.Use(p.HandlerFunc())
	p.SetMetricsPathWithAuth(e, accounts)
}

// HandlerFunc defines handler function for middleware
func (p *Prometheus) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == p.MetricsPath {
			c.Next()
			return
		}

		start := time.Now()
		reqSz := computeApproximateRequestSize(c.Request)

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := MillisecondsSince(start)
		resSz := float64(c.Writer.Size())

		url := p.ReqCntURLLabelMappingFn(c)
		// jlambert Oct 2018 - sidecar specific mod
		if len(p.URLLabelFromContext) > 0 {
			u, found := c.Get(p.URLLabelFromContext)
			if !found {
				u = "unknown"
			}
			url = u.(string)
		}

		ref := c.Request.Header.Get(RefererKey)

		p.reqDur.WithLabelValues(status, c.Request.Method, url, ref).Observe(elapsed)
		p.reqCnt.WithLabelValues(status, c.Request.Method, url, ref).Inc()
		p.reqSz.WithLabelValues(status, c.Request.Method, url, ref).Observe(float64(reqSz))
		p.resSz.WithLabelValues(status, c.Request.Method, url, ref).Observe(resSz)
	}
}
