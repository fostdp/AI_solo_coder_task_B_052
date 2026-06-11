package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	HTTPRequestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	InfluxDBWritesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "influxdb_writes_total",
		Help: "Total number of InfluxDB writes",
	}, []string{"status"})

	InfluxDBWriteErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "influxdb_write_errors_total",
		Help: "Total number of InfluxDB write errors",
	})

	InfluxDBWriteQueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "influxdb_write_queue_size",
		Help: "Current size of InfluxDB write queue",
	})

	AlertsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "alerts_total",
		Help: "Total number of alerts triggered",
	}, []string{"level", "channel"})

	LoRaPacketsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lora_packets_received_total",
		Help: "Total number of LoRa packets received",
	}, []string{"device_type", "status"})

	LoRaDuplicatesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lora_packets_duplicate_total",
		Help: "Total number of duplicate LoRa packets dropped",
	})

	PredictionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "termite_predictions_total",
		Help: "Total number of termite predictions",
	}, []string{"risk_level"})

	PipelineMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pipeline_messages_total",
		Help: "Total number of messages processed by pipeline stage",
	}, []string{"stage", "type"})

	PipelineErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pipeline_errors_total",
		Help: "Total number of pipeline errors",
	}, []string{"stage"})
)

func ObserveHTTPRequest(method, path, status string, duration time.Duration) {
	HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	HTTPRequestTotal.WithLabelValues(method, path, status).Inc()
}

func IncInfluxDBWrite(success bool) {
	status := "success"
	if !success {
		status = "error"
		InfluxDBWriteErrors.Inc()
	}
	InfluxDBWritesTotal.WithLabelValues(status).Inc()
}

func SetInfluxDBQueueSize(size int) {
	InfluxDBWriteQueueSize.Set(float64(size))
}

func IncAlert(level, channel string) {
	AlertsTotal.WithLabelValues(level, channel).Inc()
}

func IncLoRaPacket(deviceType, status string) {
	LoRaPacketsTotal.WithLabelValues(deviceType, status).Inc()
}

func IncLoRaDuplicate() {
	LoRaDuplicatesTotal.Inc()
}

func IncPrediction(riskLevel string) {
	PredictionTotal.WithLabelValues(riskLevel).Inc()
}

func IncPipelineMessage(stage, msgType string) {
	PipelineMessagesTotal.WithLabelValues(stage, msgType).Inc()
}

func IncPipelineError(stage string) {
	PipelineErrorsTotal.WithLabelValues(stage).Inc()
}
