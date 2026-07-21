package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests by method/path/status"},
		[]string{"method", "path", "status"},
	)
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request duration in seconds"},
		[]string{"method", "path"},
	)
	LLMEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "llm_events_total", Help: "Total LLM inference events recorded by model and status"},
		[]string{"model_id", "status"},
	)
	LLMTokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "llm_tokens_total", Help: "Total LLM tokens by model and direction (in/out)"},
		[]string{"model_id", "direction"},
	)
	LLMDecisionScoreSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "llm_decision_score_sum", Help: "Sum of composite decision scores by model"},
		[]string{"model_id"},
	)
)

func Init() {
	prometheus.MustRegister(RequestsTotal, RequestDuration, LLMEventsTotal, LLMTokensTotal, LLMDecisionScoreSum)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func Observe(method, path string, status int, dur time.Duration) {
	RequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	RequestDuration.WithLabelValues(method, path).Observe(dur.Seconds())
}
