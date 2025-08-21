package monitoring

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	RequestDuration *prometheus.HistogramVec
	RequestsTotal   *prometheus.CounterVec
	TasksTotal      *prometheus.GaugeVec
	ActiveTasks     prometheus.Gauge
	DatabaseConnections prometheus.Gauge
}

// NewMetrics creates new Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint", "status_code"},
		),
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		TasksTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tasks_total",
				Help: "Total number of tasks by status",
			},
			[]string{"status", "project_id"},
		),
		ActiveTasks: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_tasks_total",
				Help: "Total number of active tasks",
			},
		),
		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_active",
				Help: "Number of active database connections",
			},
		),
	}
}

// PrometheusMiddleware creates Gin middleware for Prometheus metrics
func PrometheusMiddleware(metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())

		metrics.RequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			statusCode,
		).Observe(duration)

		metrics.RequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			statusCode,
		).Inc()
	}
}

// UpdateTaskMetrics updates task-related metrics
func (m *Metrics) UpdateTaskMetrics(status string, projectID int, count float64) {
	m.TasksTotal.WithLabelValues(status, strconv.Itoa(projectID)).Set(count)
}

// UpdateActiveTasksMetric updates the active tasks metric
func (m *Metrics) UpdateActiveTasksMetric(count float64) {
	m.ActiveTasks.Set(count)
}

// UpdateDatabaseConnectionsMetric updates the database connections metric
func (m *Metrics) UpdateDatabaseConnectionsMetric(count float64) {
	m.DatabaseConnections.Set(count)
}